package lister

import (
	"net/http"
	"reflect"
	"testing"
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
