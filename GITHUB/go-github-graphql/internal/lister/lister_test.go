package lister

import "testing"

func TestSearchQuery(t *testing.T) {
	got := SearchQuery("myorg", "argocd")
	want := "org:myorg topic:argocd"
	if got != want {
		t.Fatalf("SearchQuery = %q, want %q", got, want)
	}
}

func TestSearchQuery_Topic(t *testing.T) {
	got := SearchQuery("acme", "gitops")
	want := "org:acme topic:gitops"
	if got != want {
		t.Fatalf("SearchQuery = %q, want %q", got, want)
	}
}
