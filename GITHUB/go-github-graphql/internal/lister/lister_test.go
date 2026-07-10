package lister

import (
	"net/http"
	"reflect"
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
	want := []any{
		"X-Ratelimit-Limit", "5000",
		"X-Ratelimit-Remaining", "4999",
		"X-Ratelimit-Reset", "1700000000",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rateLimitAttrs = %v, want %v", got, want)
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
