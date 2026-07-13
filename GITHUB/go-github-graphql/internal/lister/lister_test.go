package lister

import (
	"bytes"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

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
	// X-Ratelimit-Reset is rendered as a UTC-4 timestamp; 1700000000 is
	// 2023-11-14 22:13:20 UTC, i.e. 18:13:20 at UTC-4.
	want := []any{
		"X-Ratelimit-Limit", "5000",
		"X-Ratelimit-Remaining", "4999",
		"X-Ratelimit-Reset", "2023-11-14 18:13:20 UTC-4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rateLimitAttrs = %v, want %v", got, want)
	}
}

func TestRateLimitAttrs_ResetNonNumeric(t *testing.T) {
	// A non-numeric reset value is passed through unchanged rather than dropped.
	h := http.Header{}
	h.Set("X-RateLimit-Reset", "not-a-number")

	got := rateLimitAttrs(h)
	want := []any{"X-Ratelimit-Reset", "not-a-number"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rateLimitAttrs = %v, want %v", got, want)
	}
}

func TestStatsRecord(t *testing.T) {
	s := &Stats{}
	s.record(200)
	s.record(200)
	s.record(403)
	s.recordRateLimited()

	if s.Requests != 3 {
		t.Fatalf("Requests = %d, want 3", s.Requests)
	}
	if s.RateLimited != 1 {
		t.Fatalf("RateLimited = %d, want 1", s.RateLimited)
	}
	if s.StatusCounts[200] != 2 || s.StatusCounts[403] != 1 {
		t.Fatalf("StatusCounts = %v, want {200:2, 403:1}", s.StatusCounts)
	}
}

// stubRoundTripper returns a canned response for every request.
type stubRoundTripper struct {
	resp *http.Response
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	// A fresh body per call so the transport can drain/close it on retries.
	r := *s.resp
	r.Body = http.NoBody
	return &r, nil
}

func TestTransportRecordsStats(t *testing.T) {
	stats := &Stats{}
	tr := &rateLimitTransport{
		next:       stubRoundTripper{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}}},
		maxRetries: 0,
		stats:      stats,
		logger:     slog.Default(),
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/graphql", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}

	if stats.Requests != 1 || stats.StatusCounts[http.StatusOK] != 1 {
		t.Fatalf("stats = %+v, want 1 request with a 200", stats)
	}
	if stats.RateLimited != 0 {
		t.Fatalf("RateLimited = %d, want 0", stats.RateLimited)
	}
}

func TestTransportLogsHeaders(t *testing.T) {
	// The transport's dedicated logger emits the rate-limit headers regardless
	// of the global logger's level, so API-call headers survive quiet mode.
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	h := http.Header{}
	h.Set("X-RateLimit-Limit", "5000")
	h.Set("X-RateLimit-Remaining", "4999")

	tr := &rateLimitTransport{
		next:       stubRoundTripper{resp: &http.Response{StatusCode: http.StatusOK, Header: h}},
		maxRetries: 0,
		stats:      &Stats{},
		logger:     logger,
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/graphql", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}

	if !strings.Contains(buf.String(), "github rate limit") {
		t.Fatalf("expected rate-limit headers to be logged, got %q", buf.String())
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
