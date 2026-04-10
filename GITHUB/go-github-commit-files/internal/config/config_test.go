package config

import (
	"testing"
)

func TestParseFilePair_valid(t *testing.T) {
	pair, err := parseFilePair("./local.yaml:configs/remote.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.Src != "./local.yaml" {
		t.Errorf("want Src=./local.yaml, got %q", pair.Src)
	}
	if pair.Dest != "configs/remote.yaml" {
		t.Errorf("want Dest=configs/remote.yaml, got %q", pair.Dest)
	}
}

func TestParseFilePair_missingColon(t *testing.T) {
	_, err := parseFilePair("localfile.yaml")
	if err == nil {
		t.Fatal("expected error for missing colon, got nil")
	}
}

func TestParseFilePair_emptyDest(t *testing.T) {
	_, err := parseFilePair("local.yaml:")
	if err == nil {
		t.Fatal("expected error for empty dest, got nil")
	}
}

func TestParseFilePair_emptySrc(t *testing.T) {
	_, err := parseFilePair(":dest.yaml")
	if err == nil {
		t.Fatal("expected error for empty src, got nil")
	}
}

func TestParseFrom_missingRequired(t *testing.T) {
	_, err := ParseFrom([]string{}, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}

func TestParseFrom_missingFiles(t *testing.T) {
	args := []string{
		"--app-id", "123",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "456",
		"--owner", "myorg",
		"--repo", "myrepo",
		"--branch", "main",
	}
	_, err := ParseFrom(args, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for missing --file, got nil")
	}
}

func TestParseFrom_valid(t *testing.T) {
	args := []string{
		"--app-id", "123",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "456",
		"--owner", "myorg",
		"--repo", "myrepo",
		"--branch", "main",
		"--file", "./a.yaml:path/a.yaml",
		"--file", "./b.yaml:path/b.yaml",
		"--log-level", "debug",
		"--max-retries", "3",
	}
	cfg, err := ParseFrom(args, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "123" {
		t.Errorf("want AppID=123, got %q", cfg.AppID)
	}
	if len(cfg.Files) != 2 {
		t.Errorf("want 2 files, got %d", len(cfg.Files))
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("want MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("want LogLevel=debug, got %q", cfg.LogLevel)
	}
}

func TestParseFrom_envVarFallback(t *testing.T) {
	getenv := func(key string) string {
		m := map[string]string{
			"GITHUB_APP_ID":           "999",
			"GITHUB_PRIVATE_KEY_PATH": "/key.pem",
			"GITHUB_INSTALLATION_ID":  "888",
			"GITHUB_REPO_ORG":         "theorg",
			"GITHUB_REPO_NAME":        "therepo",
			"GITHUB_REPO_BRANCH":      "develop",
		}
		return m[key]
	}
	args := []string{"--file", "./x.yaml:x.yaml"}
	cfg, err := ParseFrom(args, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "999" {
		t.Errorf("want AppID from env=999, got %q", cfg.AppID)
	}
	if cfg.Owner != "theorg" {
		t.Errorf("want Owner from env=theorg, got %q", cfg.Owner)
	}
}

func TestParseFrom_deleteFile(t *testing.T) {
	args := []string{
		"--app-id", "123",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "456",
		"--owner", "myorg",
		"--repo", "myrepo",
		"--branch", "main",
		"--file", "./a.yaml:path/a.yaml",
		"--delete-file", "old/file.yaml",
		"--delete-file", "another/old.yaml",
	}
	cfg, err := ParseFrom(args, func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.DeleteFiles) != 2 {
		t.Errorf("want 2 delete files, got %d", len(cfg.DeleteFiles))
	}
	if cfg.DeleteFiles[0] != "old/file.yaml" {
		t.Errorf("want DeleteFiles[0]=old/file.yaml, got %q", cfg.DeleteFiles[0])
	}
	if cfg.DeleteFiles[1] != "another/old.yaml" {
		t.Errorf("want DeleteFiles[1]=another/old.yaml, got %q", cfg.DeleteFiles[1])
	}
}

func TestParseFrom_deleteFileOnly(t *testing.T) {
	args := []string{
		"--app-id", "123",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "456",
		"--owner", "myorg",
		"--repo", "myrepo",
		"--branch", "main",
		"--delete-file", "old/file.yaml",
	}
	cfg, err := ParseFrom(args, func(string) string { return "" })
	if err != nil {
		t.Fatalf("expected delete-file-only to be valid, got error: %v", err)
	}
	if len(cfg.Files) != 0 {
		t.Errorf("want 0 upload files, got %d", len(cfg.Files))
	}
	if len(cfg.DeleteFiles) != 1 {
		t.Errorf("want 1 delete file, got %d", len(cfg.DeleteFiles))
	}
}

func TestParseFrom_neitherFileNorDeleteFile(t *testing.T) {
	args := []string{
		"--app-id", "123",
		"--private-key", "/tmp/key.pem",
		"--installation-id", "456",
		"--owner", "myorg",
		"--repo", "myrepo",
		"--branch", "main",
	}
	_, err := ParseFrom(args, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error when neither --file nor --delete-file is provided, got nil")
	}
}
