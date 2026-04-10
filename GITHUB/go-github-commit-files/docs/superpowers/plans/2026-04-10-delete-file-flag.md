# --delete-file Flag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a repeatable `--delete-file <repo-path>` flag so file deletions are included in the same atomic Git commit as uploads.

**Architecture:** Two tasks — config first (TDD), then committer + main.go. The Git Trees API signals deletion via a nil `SHA` field in a tree entry; no blob creation is needed for deletes. The `CommitFiles` signature gains a `deleteFiles []string` parameter.

**Tech Stack:** Go 1.26, `github.com/google/go-github/v69`, stdlib `flag`, `log/slog`

---

### Task 1: Config — add `DeleteFiles` field and `--delete-file` flag

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Add these three tests to `internal/config/config_test.go` (append after the existing tests):

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/... -run "TestParseFrom_deleteFile|TestParseFrom_deleteFileOnly|TestParseFrom_neitherFileNorDeleteFile" -v
```

Expected: FAIL — `cfg.DeleteFiles` field does not exist yet.

- [ ] **Step 3: Update `internal/config/config.go`**

Make these four targeted changes:

**3a.** Add `DeleteFiles []string` to the `Config` struct (after `Files []FilePair`):

```go
type Config struct {
	AppID          string
	PrivateKeyPath string
	InstallationID string
	Owner          string
	Repo           string
	Branch         string
	Files          []FilePair
	DeleteFiles    []string
	MaxRetries     int
	LogLevel       string
}
```

**3b.** Add a `deleteFiles fileFlags` variable and register the flag inside `ParseFrom`, right after `fs.Var(&files, ...)`:

```go
var files fileFlags
var deleteFiles fileFlags

appID := fs.String("app-id", envOr("GITHUB_APP_ID", ""), "GitHub App ID")
privateKey := fs.String("private-key", envOr("GITHUB_PRIVATE_KEY_PATH", ""), "Path to PEM private key file")
installationID := fs.String("installation-id", envOr("GITHUB_INSTALLATION_ID", ""), "App installation ID")
owner := fs.String("owner", envOr("GITHUB_REPO_ORG", ""), "Repo owner (org or user)")
repo := fs.String("repo", envOr("GITHUB_REPO_NAME", ""), "Repository name")
branch := fs.String("branch", envOr("GITHUB_REPO_BRANCH", ""), "Target branch")
maxRetriesStr := fs.String("max-retries", envOr("MAX_RETRIES", "5"), "Max retries per API call, 0 = no retries (default 5)")
logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "Log level: debug/info/warn/error")
fs.Var(&files, "file", "Repeatable src:dest file pair (e.g. ./local.yaml:path/in/repo.yaml)")
fs.Var(&deleteFiles, "delete-file", "Repeatable repo path to delete (e.g. old/config.yaml)")
```

**3c.** Populate `cfg.DeleteFiles` after populating `cfg.Files` (append after the `cfg.Files = append(...)` loop):

```go
cfg.DeleteFiles = []string(deleteFiles)
```

**3d.** Replace the empty-files guard with one that allows delete-only operations:

```go
if len(cfg.Files) == 0 && len(cfg.DeleteFiles) == 0 {
	return nil, fmt.Errorf("at least one --file or --delete-file is required")
}
```

- [ ] **Step 4: Run all config tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all 11 tests PASS (8 original + 3 new).

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add --delete-file flag to config"
```

---

### Task 2: Committer and main.go — wire deletion into the Git commit

**Files:**
- Modify: `internal/committer/committer.go`
- Modify: `main.go`

- [ ] **Step 1: Update `CommitFiles` signature in `internal/committer/committer.go`**

Change the function signature and the empty-slice guard:

```go
// CommitFiles atomically commits all files in a single Git commit.
// files contains upload pairs; deleteFiles contains repo paths to remove.
// Returns the new commit SHA on success.
func (c *Committer) CommitFiles(ctx context.Context, files []FilePair, deleteFiles []string) (string, error) {
	if len(files) == 0 && len(deleteFiles) == 0 {
		return "", fmt.Errorf("CommitFiles: files and deleteFiles must not both be empty")
	}
```

- [ ] **Step 2: Add deletion tree entries after the blob-creation loop**

In `CommitFiles`, after the closing `}` of the blob-creation loop (after line 109 in the current file), add:

```go
	// Step 3b: add a nil-SHA tree entry for each file to delete.
	// The Git Trees API interprets SHA=null as "remove this path from the tree".
	for _, path := range deleteFiles {
		slog.Info("queuing file for deletion", "path", path)
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(path),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  nil,
		})
	}
```

- [ ] **Step 3: Replace the commit message construction**

Find and replace the commit message block (currently lines 124–129):

```go
	// Step 5: build commit message
	var msgParts []string
	if len(files) > 0 {
		destPaths := make([]string, len(files))
		for i, fp := range files {
			destPaths[i] = fp.Dest
		}
		msgParts = append(msgParts, "update "+strings.Join(destPaths, ", "))
	}
	if len(deleteFiles) > 0 {
		msgParts = append(msgParts, "delete "+strings.Join(deleteFiles, ", "))
	}
	message := "chore: " + strings.Join(msgParts, ", ") + " via automation"
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: compile error — `main.go` still calls `CommitFiles` with the old one-argument signature. That's expected; fix it in the next step.

- [ ] **Step 5: Update `main.go`**

Make these two targeted changes:

**5a.** Add `delete_file_count` to the startup log and log each queued deletion. Replace the existing startup `slog.Info("starting go-github-commit-files", ...)` block and the files loop with:

```go
	slog.Info("starting go-github-commit-files",
		"owner", cfg.Owner,
		"repo", cfg.Repo,
		"branch", cfg.Branch,
		"file_count", len(cfg.Files),
		"delete_file_count", len(cfg.DeleteFiles),
		"max_retries", cfg.MaxRetries,
		"log_level", cfg.LogLevel,
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
		"private_key_path", cfg.PrivateKeyPath,
	)
```

**5b.** After the existing upload-files loop, add a loop to log deletion targets, and pass `cfg.DeleteFiles` to `CommitFiles`:

```go
	files := make([]committer.FilePair, len(cfg.Files))
	for i, fp := range cfg.Files {
		files[i] = committer.FilePair{Src: fp.Src, Dest: fp.Dest}
		slog.Info("file queued for commit", "src", fp.Src, "dest", fp.Dest)
	}

	for _, path := range cfg.DeleteFiles {
		slog.Info("file queued for deletion", "path", path)
	}

	c := committer.New(client, cfg.Owner, cfg.Repo, cfg.Branch, cfg.MaxRetries)
	commitSHA, err := c.CommitFiles(ctx, files, cfg.DeleteFiles)
```

- [ ] **Step 6: Verify full build and tests**

```bash
go build ./...
go test ./...
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 7: Verify help shows --delete-file**

```bash
./go-github-commit-files --help 2>&1 || true
```

Expected: output includes `-delete-file` flag description.

- [ ] **Step 8: Commit**

```bash
git add internal/committer/committer.go main.go
git commit -m "feat: support --delete-file in committer and main"
```
