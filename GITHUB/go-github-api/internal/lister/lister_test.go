package lister

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v89/github"
)

// roundTripFunc adapts a function to an http.RoundTripper for tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRateLimitTransport_CountsRequests(t *testing.T) {
	var calls int
	next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})

	rt := &rateLimitTransport{next: next, maxRetries: 5}
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/foo", nil)

	for i := 1; i <= 3; i++ {
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp.Body.Close()
		if got := rt.count.Load(); got != int64(i) {
			t.Fatalf("after %d requests count = %d, want %d", i, got, i)
		}
	}
	if calls != 3 {
		t.Fatalf("next called %d times, want 3", calls)
	}
}

// newStubClient builds a Client whose go-github requests are served by fn.
func newStubClient(fn roundTripFunc) *Client {
	gh, _ := github.NewClient(github.WithHTTPClient(&http.Client{Transport: fn}))
	return &Client{
		gh:        gh,
		transport: &rateLimitTransport{},
	}
}

// jsonResp builds a JSON HTTP response with the given status and body.
func jsonResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHasProjectInfo(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    bool
		wantErr bool
	}{
		{"present", http.StatusOK, `{"type":"file","name":"project-info.yaml"}`, true, false},
		{"absent", http.StatusNotFound, `{"message":"Not Found"}`, false, false},
		{"server error", http.StatusInternalServerError, `{"message":"boom"}`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newStubClient(func(req *http.Request) (*http.Response, error) {
				return jsonResp(tt.status, tt.body), nil
			})
			got, err := c.hasProjectInfo(context.Background(), "acme", "app")
			if (err != nil) != tt.wantErr {
				t.Fatalf("hasProjectInfo err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("hasProjectInfo = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchQuery(t *testing.T) {
	got := SearchQuery("myorg", "argocd", false)
	want := "org:myorg topic:argocd"
	if got != want {
		t.Fatalf("SearchQuery = %q, want %q", got, want)
	}
}

func TestSearchQuery_Topic(t *testing.T) {
	got := SearchQuery("acme", "gitops", false)
	want := "org:acme topic:gitops"
	if got != want {
		t.Fatalf("SearchQuery = %q, want %q", got, want)
	}
}

func TestSearchQuery_ExcludeArchived(t *testing.T) {
	got := SearchQuery("myorg", "argocd", true)
	want := "org:myorg topic:argocd archived:false"
	if got != want {
		t.Fatalf("SearchQuery = %q, want %q", got, want)
	}
}

func TestRateLimitAttrs(t *testing.T) {
	h := http.Header{}
	h.Set("X-RateLimit-Limit", "5000")
	h.Set("X-RateLimit-Remaining", "4999")
	h.Set("X-RateLimit-Reset", "1700000000")
	h.Set("Content-Type", "application/json")

	got := rateLimitAttrs(h)
	want := []any{
		"X-Ratelimit-Limit", "5000",
		"X-Ratelimit-Remaining", "4999",
		"X-Ratelimit-Reset", "2023-11-14T18:13:20-04:00", // epoch 1700000000 as RFC 3339 UTC-4
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rateLimitAttrs = %v, want %v", got, want)
	}
}

func TestFormatResetHeader(t *testing.T) {
	want := "2023-11-14T18:13:20-04:00"
	if got := formatResetHeader("1700000000"); got != want {
		t.Fatalf("formatResetHeader(epoch) = %q, want %q", got, want)
	}
	// A non-numeric value is passed through unchanged.
	if got := formatResetHeader("not-a-number"); got != "not-a-number" {
		t.Fatalf("formatResetHeader(invalid) = %q, want unchanged", got)
	}
}

func TestRateLimitAttrs_None(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")

	got := rateLimitAttrs(h)
	if len(got) != 0 {
		t.Fatalf("rateLimitAttrs = %v, want empty", got)
	}
}

func resp(status int, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Header: h}
}

func TestRetryAfter(t *testing.T) {
	now := time.Unix(1_000, 0)

	tests := []struct {
		name      string
		resp      *http.Response
		wantWait  time.Duration
		wantLimit bool
	}{
		{
			name:      "ok response is not rate limited",
			resp:      resp(http.StatusOK, map[string]string{"X-RateLimit-Remaining": "0"}),
			wantWait:  0,
			wantLimit: false,
		},
		{
			name:      "403 without rate-limit signals is not retried",
			resp:      resp(http.StatusForbidden, nil),
			wantWait:  0,
			wantLimit: false,
		},
		{
			name:      "secondary limit honors Retry-After",
			resp:      resp(http.StatusForbidden, map[string]string{"Retry-After": "30"}),
			wantWait:  30 * time.Second,
			wantLimit: true,
		},
		{
			name:      "too many requests honors Retry-After",
			resp:      resp(http.StatusTooManyRequests, map[string]string{"Retry-After": "5"}),
			wantWait:  5 * time.Second,
			wantLimit: true,
		},
		{
			name:      "primary limit waits until reset plus buffer",
			resp:      resp(http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1060"}),
			wantWait:  61 * time.Second, // (1060-1000) + 1s buffer
			wantLimit: true,
		},
		{
			name:      "reset in the past clamps to zero",
			resp:      resp(http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "900"}),
			wantWait:  0,
			wantLimit: true,
		},
		{
			name:      "remaining budget is not rate limited",
			resp:      resp(http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "42", "X-RateLimit-Reset": "1060"}),
			wantWait:  0,
			wantLimit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWait, gotLimit := retryAfter(tt.resp, now)
			if gotWait != tt.wantWait || gotLimit != tt.wantLimit {
				t.Fatalf("retryAfter = (%v, %v), want (%v, %v)", gotWait, gotLimit, tt.wantWait, tt.wantLimit)
			}
		})
	}
}
