package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParseFrom_FlagsProvided(t *testing.T) {
	cfg, err := ParseFrom([]string{
		"--app-id", "111",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "222",
		"--org", "myorg",
		"--topic", "gitops",
	}, env(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "111" || cfg.Org != "myorg" || cfg.Topic != "gitops" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestParseFrom_EnvFallback(t *testing.T) {
	cfg, err := ParseFrom(nil, env(map[string]string{
		"GITHUB_APP_ID":           "1",
		"GITHUB_PRIVATE_KEY_PATH": "/k.pem",
		"GITHUB_INSTALLATION_ID":  "2",
		"GITHUB_REPO_ORG":         "envorg",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Org != "envorg" {
		t.Fatalf("expected env org, got %q", cfg.Org)
	}
	if cfg.Topic != "argocd" {
		t.Fatalf("expected default topic argocd, got %q", cfg.Topic)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log-level info, got %q", cfg.LogLevel)
	}
}

func TestParseFrom_FlagOverridesEnv(t *testing.T) {
	cfg, err := ParseFrom([]string{"--org", "flagorg"}, env(map[string]string{
		"GITHUB_APP_ID":           "1",
		"GITHUB_PRIVATE_KEY_PATH": "/k.pem",
		"GITHUB_INSTALLATION_ID":  "2",
		"GITHUB_REPO_ORG":         "envorg",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Org != "flagorg" {
		t.Fatalf("expected flag to override env, got %q", cfg.Org)
	}
}

func TestParseFrom_ExcludeArchived(t *testing.T) {
	base := map[string]string{
		"GITHUB_APP_ID":           "1",
		"GITHUB_PRIVATE_KEY_PATH": "/k.pem",
		"GITHUB_INSTALLATION_ID":  "2",
		"GITHUB_REPO_ORG":         "org",
	}

	// Default: archived repos are included.
	cfg, err := ParseFrom(nil, env(base))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ExcludeArchived {
		t.Fatal("expected ExcludeArchived to default to false")
	}

	// Flag sets it.
	cfg, err = ParseFrom([]string{"--exclude-archived"}, env(base))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.ExcludeArchived {
		t.Fatal("expected --exclude-archived to set ExcludeArchived")
	}

	// Env var sets it.
	envMap := map[string]string{"EXCLUDE_ARCHIVED": "true"}
	for k, v := range base {
		envMap[k] = v
	}
	cfg, err = ParseFrom(nil, env(envMap))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.ExcludeArchived {
		t.Fatal("expected EXCLUDE_ARCHIVED=true to set ExcludeArchived")
	}
}

func TestParseFrom_MissingRequired(t *testing.T) {
	_, err := ParseFrom([]string{"--app-id", "1"}, env(nil))
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
	if !strings.Contains(err.Error(), "required flags not set") {
		t.Fatalf("expected required-flags error, got %q", err)
	}
	// All three still-missing required flags should be reported at once.
	for _, want := range []string{"--private-key", "--installation-id", "--org"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to mention %s, got %q", want, err)
		}
	}
}
