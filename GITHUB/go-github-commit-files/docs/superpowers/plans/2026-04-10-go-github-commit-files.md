# go-github-commit-files Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that uses a GitHub App token to atomically commit multiple local files to a branch via the Git Trees API, with rate-limit-aware retry and structured JSON logging.

**Architecture:** Four focused packages under `internal/` — `config` (flag+env parsing), `githubapp` (auth), `retry` (retry wrapper), `committer` (Git Trees API logic) — wired by `main.go`. Each package has a single responsibility and is independently testable.

**Tech Stack:** Go 1.26, `github.com/google/go-github/v69`, `github.com/beatlabs/github-auth`, stdlib `log/slog`, `flag`, `errors`, `math`, `encoding/base64`

---

### Task 1: Bootstrap project

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `main.go` (stub)
- Create: `internal/config/` (empty dir via placeholder)
- Create: `internal/githubapp/` (empty dir via placeholder)
- Create: `internal/retry/` (empty dir via placeholder)
- Create: `internal/committer/` (empty dir via placeholder)

- [ ] **Step 1: Create go.mod**

```
module github.com/prune998/devops/GITHUB/go-github-commit-files

go 1.26

require (
	github.com/beatlabs/github-auth v0.0.0-20240915075323-ab38acb2d2a1
	github.com/google/go-github/v69 v69.2.0
)
```

Save to `go.mod`.

- [ ] **Step 2: Create .gitignore**

```
go-github-commit-files
*.pem
*.key
```

Save to `.gitignore`.

- [ ] **Step 3: Create stub main.go**

```go
package main

func main() {}
```

Save to `main.go`.

- [ ] **Step 4: Create internal package directories**

```bash
mkdir -p internal/config internal/githubapp internal/retry internal/committer
```

- [ ] **Step 5: Download dependencies**

```bash
go mod tidy
```

Expected: `go.sum` created, no errors.

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

Expected: no output, exit 0.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum .gitignore main.go
git commit -m "chore: bootstrap go-github-commit-files project"
```

---

### Task 2: Config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/config/config_test.go`:

```go
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
			"GITHUB_APP_ID":          "999",
			"GITHUB_PRIVATE_KEY_PATH": "/key.pem",
			"GITHUB_INSTALLATION_ID": "888",
			"GITHUB_REPO_ORG":        "theorg",
			"GITHUB_REPO_NAME":       "therepo",
			"GITHUB_REPO_BRANCH":     "develop",
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/...
```

Expected: FAIL — `config.go` does not exist yet.

- [ ] **Step 3: Implement config.go**

Create `internal/config/config.go`:

```go
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FilePair holds a local source path and a repository destination path.
type FilePair struct {
	Src  string
	Dest string
}

// Config holds all resolved runtime configuration.
type Config struct {
	AppID          string
	PrivateKeyPath string
	InstallationID string
	Owner          string
	Repo           string
	Branch         string
	Files          []FilePair
	MaxRetries     int
	LogLevel       string
}

// fileFlags implements flag.Value for a repeatable --file flag.
type fileFlags []string

func (f *fileFlags) String() string { return strings.Join(*f, ", ") }
func (f *fileFlags) Set(v string) error {
	*f = append(*f, v)
	return nil
}

// parseFilePair splits "src:dest" into a FilePair, returning an error if the
// format is invalid or either side is empty.
func parseFilePair(s string) (FilePair, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return FilePair{}, fmt.Errorf("invalid file pair %q: expected non-empty src:dest", s)
	}
	return FilePair{Src: parts[0], Dest: parts[1]}, nil
}

// ParseFrom parses configuration from args and an environment lookup function.
// This signature enables unit testing without touching os.Args or real env vars.
func ParseFrom(args []string, getenv func(string) string) (*Config, error) {
	fs := flag.NewFlagSet("go-github-commit-files", flag.ContinueOnError)

	envOr := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	var files fileFlags

	appID := fs.String("app-id", envOr("GITHUB_APP_ID", ""), "GitHub App ID")
	privateKey := fs.String("private-key", envOr("GITHUB_PRIVATE_KEY_PATH", ""), "Path to PEM private key file")
	installationID := fs.String("installation-id", envOr("GITHUB_INSTALLATION_ID", ""), "App installation ID")
	owner := fs.String("owner", envOr("GITHUB_REPO_ORG", ""), "Repo owner (org or user)")
	repo := fs.String("repo", envOr("GITHUB_REPO_NAME", ""), "Repository name")
	branch := fs.String("branch", envOr("GITHUB_REPO_BRANCH", ""), "Target branch")
	maxRetriesStr := fs.String("max-retries", envOr("MAX_RETRIES", "5"), "Max API retry attempts")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "Log level: debug/info/warn/error")
	fs.Var(&files, "file", "Repeatable src:dest file pair (e.g. ./local.yaml:path/in/repo.yaml)")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	maxRetries, err := strconv.Atoi(*maxRetriesStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --max-retries value %q: %w", *maxRetriesStr, err)
	}

	cfg := &Config{
		AppID:          *appID,
		PrivateKeyPath: *privateKey,
		InstallationID: *installationID,
		Owner:          *owner,
		Repo:           *repo,
		Branch:         *branch,
		MaxRetries:     maxRetries,
		LogLevel:       *logLevel,
	}

	for _, f := range files {
		pair, err := parseFilePair(f)
		if err != nil {
			return nil, err
		}
		cfg.Files = append(cfg.Files, pair)
	}

	required := []struct{ flag, value string }{
		{"--app-id (GITHUB_APP_ID)", cfg.AppID},
		{"--private-key (GITHUB_PRIVATE_KEY_PATH)", cfg.PrivateKeyPath},
		{"--installation-id (GITHUB_INSTALLATION_ID)", cfg.InstallationID},
		{"--owner (GITHUB_REPO_ORG)", cfg.Owner},
		{"--repo (GITHUB_REPO_NAME)", cfg.Repo},
		{"--branch (GITHUB_REPO_BRANCH)", cfg.Branch},
	}
	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("required flag %s is not set", r.flag)
		}
	}
	if len(cfg.Files) == 0 {
		return nil, fmt.Errorf("at least one --file src:dest pair is required")
	}

	return cfg, nil
}

// Parse parses configuration from os.Args and real environment variables.
func Parse() (*Config, error) {
	return ParseFrom(os.Args[1:], os.Getenv)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with flag+env parsing"
```

---

### Task 3: GitHub App client package

**Files:**
- Create: `internal/githubapp/client.go`

No unit tests for this package — it wraps external auth library with no logic of its own; behavior is verified in integration.

- [ ] **Step 1: Implement client.go**

Create `internal/githubapp/client.go`:

```go
package githubapp

import (
	"context"
	"fmt"

	ghauth "github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/google/go-github/v69/github"
)

// NewClient authenticates as a GitHub App installation and returns a configured
// *github.Client. appID and installationID are the numeric IDs as strings.
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string) (*github.Client, error) {
	k, err := key.FromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading private key from %s: %w", privateKeyPath, err)
	}

	install, err := ghauth.NewConfig(appID, installationID, k)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub App installation config: %w", err)
	}

	return github.NewClient(install.Client(ctx)), nil
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/githubapp/
git commit -m "feat: add githubapp client package"
```

---

### Task 4: Retry package

**Files:**
- Create: `internal/retry/retry.go`
- Create: `internal/retry/retry_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/retry/retry_test.go`:

```go
package retry

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
)

// TestMain replaces sleepFn with an instant channel so retry tests don't
// actually wait seconds for backoff delays.
func TestMain(m *testing.M) {
	sleepFn = func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
	os.Exit(m.Run())
}

func TestDo_successOnFirstCall(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call, got %d", calls)
	}
}

func TestDo_successAfterRetry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		if calls < 3 {
			// Simulate a transient 5xx response
			return &github.ErrorResponse{
				Response: &http.Response{StatusCode: 503},
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestDo_exhaustsRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		return &github.ErrorResponse{
			Response: &http.Response{StatusCode: 500},
		}
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestDo_nonRetryableErrorReturnedImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("not retryable")
	err := Do(context.Background(), 5, "test_op", func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call (immediate return), got %d", calls)
	}
}

func TestDo_rateLimitErrorRetried(t *testing.T) {
	calls := 0
	// Set reset time in the past so there's no real sleep
	pastReset := github.Timestamp{Time: time.Now().Add(-5 * time.Second)}
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		if calls == 1 {
			return &github.RateLimitError{
				Rate:    github.Rate{Reset: pastReset},
				Message: "rate limit exceeded",
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("want 2 calls, got %d", calls)
	}
}

func TestDo_contextCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	err := Do(ctx, 5, "test_op", func() error {
		calls++
		cancel() // cancel context right before the retry wait
		return &github.ErrorResponse{
			Response: &http.Response{StatusCode: 500},
		}
	})
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

func TestWaitDuration_rateLimitError(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Second)
	err := &github.RateLimitError{
		Rate: github.Rate{Reset: github.Timestamp{Time: resetAt}},
	}
	dur, retry := waitDuration(err, 1)
	if !retry {
		t.Error("expected retry=true for rate limit error")
	}
	// Should be roughly 10s + 1s jitter
	if dur < 9*time.Second || dur > 12*time.Second {
		t.Errorf("unexpected wait duration %v, want ~11s", dur)
	}
}

func TestWaitDuration_5xxError(t *testing.T) {
	err := &github.ErrorResponse{
		Response: &http.Response{StatusCode: 500},
	}
	dur, retry := waitDuration(err, 1)
	if !retry {
		t.Error("expected retry=true for 5xx error")
	}
	if dur != 1*time.Second {
		t.Errorf("want 1s backoff on attempt 1, got %v", dur)
	}

	dur2, _ := waitDuration(err, 2)
	if dur2 != 2*time.Second {
		t.Errorf("want 2s backoff on attempt 2, got %v", dur2)
	}

	dur3, _ := waitDuration(err, 3)
	if dur3 != 4*time.Second {
		t.Errorf("want 4s backoff on attempt 3, got %v", dur3)
	}
}

func TestWaitDuration_4xxError_notRetried(t *testing.T) {
	err := &github.ErrorResponse{
		Response: &http.Response{StatusCode: 404},
	}
	_, retry := waitDuration(err, 1)
	if retry {
		t.Error("expected retry=false for 4xx error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/retry/...
```

Expected: FAIL — `retry.go` does not exist yet.

- [ ] **Step 3: Implement retry.go**

Create `internal/retry/retry.go`:

```go
package retry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/go-github/v69/github"
)

const maxBackoffSeconds = 60

// sleepFn is the function used to sleep between retries.
// It is a package variable so tests can replace it with an instant channel.
var sleepFn = time.After

// Do calls fn up to maxAttempts times. It retries on GitHub rate limit errors
// and transient HTTP 5xx errors. All other errors are returned immediately.
// Every attempt, wait, and outcome is logged.
func Do(ctx context.Context, maxAttempts int, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		slog.Info("api call attempt", "operation", operation, "attempt", attempt, "max_attempts", maxAttempts)

		err := fn()
		if err == nil {
			slog.Info("api call succeeded", "operation", operation, "attempt", attempt)
			return nil
		}
		lastErr = err

		waitDur, shouldRetry := waitDuration(err, attempt)
		if !shouldRetry {
			slog.Error("api call failed, error is not retryable",
				"operation", operation,
				"attempt", attempt,
				"error", err,
			)
			return err
		}

		if attempt == maxAttempts {
			slog.Error("api call failed, max attempts reached",
				"operation", operation,
				"attempt", attempt,
				"error", err,
			)
			break
		}

		slog.Warn("api call failed, will retry",
			"operation", operation,
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"wait_duration", waitDur.String(),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting to retry %s: %w", operation, ctx.Err())
		case <-sleepFn(waitDur):
			slog.Info("retry wait complete, retrying", "operation", operation, "next_attempt", attempt+1)
		}
	}

	return fmt.Errorf("operation %s failed after %d attempts: %w", operation, maxAttempts, lastErr)
}

// waitDuration returns the duration to wait before the next attempt and
// whether the error is retryable. It handles GitHub rate limit errors,
// abuse rate limit errors, and transient 5xx HTTP errors.
func waitDuration(err error, attempt int) (time.Duration, bool) {
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		wait := time.Until(rateLimitErr.Rate.Reset.Time) + time.Second // +1s jitter
		if wait < time.Second {
			wait = time.Second
		}
		slog.Warn("github rate limit hit",
			"reset_at", rateLimitErr.Rate.Reset.Time,
			"wait", wait.String(),
		)
		return wait, true
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		wait := abuseErr.GetRetryAfter()
		if wait <= 0 {
			wait = time.Second
		}
		slog.Warn("github abuse rate limit hit", "retry_after", wait.String())
		return wait, true
	}

	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		if errResp.Response.StatusCode >= 500 {
			seconds := math.Min(math.Pow(2, float64(attempt-1)), float64(maxBackoffSeconds))
			wait := time.Duration(seconds) * time.Second
			slog.Warn("github transient error",
				"status_code", errResp.Response.StatusCode,
				"backoff", wait.String(),
			)
			return wait, true
		}
		return 0, false
	}

	return 0, false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/retry/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/retry/
git commit -m "feat: add retry package with rate-limit-aware backoff"
```

---

### Task 5: Committer package

**Files:**
- Create: `internal/committer/committer.go`

The committer calls the GitHub API in sequence; unit testing requires a live GitHub App or a full HTTP mock server. Correctness is verified by running the binary against a real repo. No unit tests for this package.

- [ ] **Step 1: Implement committer.go**

Create `internal/committer/committer.go`:

```go
package committer

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/retry"
)

// FilePair holds a local source path and a repository destination path.
type FilePair struct {
	Src  string
	Dest string
}

// Committer commits files to a GitHub repository branch using the Git Trees API.
type Committer struct {
	client     *github.Client
	owner      string
	repo       string
	branch     string
	maxRetries int
}

// New creates a Committer for the given repository and branch.
func New(client *github.Client, owner, repo, branch string, maxRetries int) *Committer {
	return &Committer{
		client:     client,
		owner:      owner,
		repo:       repo,
		branch:     branch,
		maxRetries: maxRetries,
	}
}

// CommitFiles atomically commits all files in a single Git commit.
// Returns the new commit SHA on success.
func (c *Committer) CommitFiles(ctx context.Context, files []FilePair) (string, error) {
	// Step 1: resolve branch HEAD to a commit SHA
	slog.Info("fetching branch ref",
		"owner", c.owner, "repo", c.repo, "branch", c.branch)
	var ref *github.Reference
	if err := retry.Do(ctx, c.maxRetries, "get_ref", func() error {
		var err error
		ref, _, err = c.client.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+c.branch)
		return err
	}); err != nil {
		return "", fmt.Errorf("getting branch ref refs/heads/%s: %w", c.branch, err)
	}
	commitSHA := ref.GetObject().GetSHA()
	slog.Info("branch ref resolved", "branch", c.branch, "commit_sha", commitSHA)

	// Step 2: resolve commit to its tree SHA
	slog.Info("fetching base commit", "commit_sha", commitSHA)
	var baseCommit *github.Commit
	if err := retry.Do(ctx, c.maxRetries, "get_commit", func() error {
		var err error
		baseCommit, _, err = c.client.Git.GetCommit(ctx, c.owner, c.repo, commitSHA)
		return err
	}); err != nil {
		return "", fmt.Errorf("getting commit %s: %w", commitSHA, err)
	}
	treeSHA := baseCommit.GetTree().GetSHA()
	slog.Info("base tree resolved", "tree_sha", treeSHA)

	// Step 3: create a blob for each file
	var treeEntries []*github.TreeEntry
	for _, fp := range files {
		slog.Info("reading local file", "src", fp.Src, "dest", fp.Dest)
		content, err := os.ReadFile(fp.Src)
		if err != nil {
			return "", fmt.Errorf("reading file %s: %w", fp.Src, err)
		}

		encoded := base64.StdEncoding.EncodeToString(content)
		slog.Info("creating blob", "dest", fp.Dest, "bytes", len(content))
		var blob *github.Blob
		if err := retry.Do(ctx, c.maxRetries, "create_blob:"+fp.Dest, func() error {
			var err error
			blob, _, err = c.client.Git.CreateBlob(ctx, c.owner, c.repo, &github.Blob{
				Content:  github.String(encoded),
				Encoding: github.String("base64"),
			})
			return err
		}); err != nil {
			return "", fmt.Errorf("creating blob for %s: %w", fp.Dest, err)
		}
		slog.Info("blob created", "dest", fp.Dest, "sha", blob.GetSHA())

		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.String(fp.Dest),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  blob.SHA,
		})
	}

	// Step 4: create a new tree from all blobs
	slog.Info("creating tree", "base_tree_sha", treeSHA, "entry_count", len(treeEntries))
	var newTree *github.Tree
	if err := retry.Do(ctx, c.maxRetries, "create_tree", func() error {
		var err error
		newTree, _, err = c.client.Git.CreateTree(ctx, c.owner, c.repo, treeSHA, treeEntries)
		return err
	}); err != nil {
		return "", fmt.Errorf("creating tree: %w", err)
	}
	slog.Info("tree created", "sha", newTree.GetSHA())

	// Step 5: build commit message from destination paths
	destPaths := make([]string, len(files))
	for i, fp := range files {
		destPaths[i] = fp.Dest
	}
	message := "chore: update " + strings.Join(destPaths, ", ") + " via automation"

	// Step 6: create the commit object
	slog.Info("creating commit", "message", message, "tree_sha", newTree.GetSHA())
	var newCommit *github.Commit
	if err := retry.Do(ctx, c.maxRetries, "create_commit", func() error {
		var err error
		newCommit, _, err = c.client.Git.CreateCommit(ctx, c.owner, c.repo, &github.Commit{
			Message: github.String(message),
			Tree:    &github.Tree{SHA: newTree.SHA},
			Parents: []*github.Commit{{SHA: github.String(commitSHA)}},
		}, &github.CreateCommitOptions{})
		return err
	}); err != nil {
		return "", fmt.Errorf("creating commit: %w", err)
	}
	slog.Info("commit created", "sha", newCommit.GetSHA(), "message", message)

	// Step 7: update the branch ref to point to the new commit
	slog.Info("updating branch ref", "branch", c.branch, "commit_sha", newCommit.GetSHA())
	if err := retry.Do(ctx, c.maxRetries, "update_ref", func() error {
		_, _, err := c.client.Git.UpdateRef(ctx, c.owner, c.repo, &github.Reference{
			Ref:    github.String("refs/heads/" + c.branch),
			Object: &github.GitObject{SHA: newCommit.SHA},
		}, false)
		return err
	}); err != nil {
		return "", fmt.Errorf("updating branch ref: %w", err)
	}

	commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", c.owner, c.repo, newCommit.GetSHA())
	slog.Info("branch updated successfully",
		"branch", c.branch,
		"commit_sha", newCommit.GetSHA(),
		"url", commitURL,
	)

	return newCommit.GetSHA(), nil
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/committer/
git commit -m "feat: add committer package using Git Trees API"
```

---

### Task 6: Wire main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Implement main.go**

Replace the stub `main.go` with:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/committer"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/config"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/githubapp"
)

func main() {
	// Bootstrap a minimal logger before config is parsed so early errors are visible.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Parse()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Re-initialise logger with the configured level.
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))

	slog.Info("starting go-github-commit-files",
		"owner", cfg.Owner,
		"repo", cfg.Repo,
		"branch", cfg.Branch,
		"file_count", len(cfg.Files),
		"max_retries", cfg.MaxRetries,
		"log_level", cfg.LogLevel,
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
		"private_key_path", cfg.PrivateKeyPath,
	)

	ctx := context.Background()

	slog.Info("creating GitHub App client",
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
	)
	client, err := githubapp.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath)
	if err != nil {
		slog.Error("failed to create GitHub App client", "error", err)
		os.Exit(1)
	}
	slog.Info("GitHub App client created")

	files := make([]committer.FilePair, len(cfg.Files))
	for i, fp := range cfg.Files {
		files[i] = committer.FilePair{Src: fp.Src, Dest: fp.Dest}
		slog.Info("file queued for commit", "src", fp.Src, "dest", fp.Dest)
	}

	c := committer.New(client, cfg.Owner, cfg.Repo, cfg.Branch, cfg.MaxRetries)
	commitSHA, err := c.CommitFiles(ctx, files)
	if err != nil {
		slog.Error("failed to commit files", "error", err)
		os.Exit(1)
	}

	slog.Info("done",
		"commit_sha", commitSHA,
		"url", "https://github.com/"+cfg.Owner+"/"+cfg.Repo+"/commit/"+commitSHA,
	)
}
```

- [ ] **Step 2: Verify full build**

```bash
go build ./...
```

Expected: no errors. Binary `go-github-commit-files` produced.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS, no failures.

- [ ] **Step 4: Verify help output**

```bash
./go-github-commit-files --help 2>&1 || true
```

Expected: usage text listing all flags.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: wire main.go — complete go-github-commit-files"
```

---

## Usage Example

```bash
export GITHUB_APP_ID=123456
export GITHUB_PRIVATE_KEY_PATH=/secrets/app.pem
export GITHUB_INSTALLATION_ID=78901234

./go-github-commit-files \
  --owner myorg \
  --repo myrepo \
  --branch main \
  --file ./configs/app.yaml:configs/app.yaml \
  --file ./configs/db.yaml:configs/db.yaml \
  --log-level debug
```

Expected log output (JSON, stderr):

```json
{"time":"...","level":"INFO","msg":"starting go-github-commit-files","owner":"myorg","repo":"myrepo","branch":"main","file_count":2,...}
{"time":"...","level":"INFO","msg":"creating GitHub App client",...}
{"time":"...","level":"INFO","msg":"GitHub App client created"}
{"time":"...","level":"INFO","msg":"file queued for commit","src":"./configs/app.yaml","dest":"configs/app.yaml"}
{"time":"...","level":"INFO","msg":"file queued for commit","src":"./configs/db.yaml","dest":"configs/db.yaml"}
{"time":"...","level":"INFO","msg":"fetching branch ref",...}
{"time":"...","level":"INFO","msg":"branch ref resolved","branch":"main","commit_sha":"abc123"}
...
{"time":"...","level":"INFO","msg":"done","commit_sha":"def456","url":"https://github.com/myorg/myrepo/commit/def456"}
```
