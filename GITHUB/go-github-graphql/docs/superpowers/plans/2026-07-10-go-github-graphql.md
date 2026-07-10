# go-github-graphql Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Go CLI that lists all repositories in a GitHub org carrying a given topic (default `argocd`), authenticating as a GitHub App and querying the GraphQL API via `shurcooL/githubv4`, emitting JSON.

**Architecture:** Three units — `internal/config` (flags + env fallback, testable parse), `internal/lister` (auth client build, paginated GraphQL search, pure query-string builder), and `main.go` (wiring + JSON output). Logging via slog to stderr; results as JSON to stdout.

**Tech Stack:** Go 1.26, `github.com/shurcooL/githubv4`, `github.com/beatlabs/github-auth`, `golang.org/x/oauth2` (transitive), stdlib `flag`, `encoding/json`, `log/slog`.

## Global Constraints

- Go module path: `github.com/prune998/devops/GITHUB/go-github-graphql` (already in `go.mod`).
- Go version: `1.26.4` (matches `go.mod`).
- App auth library: `github.com/beatlabs/github-auth` (`key.FromFile`, `inst.NewConfig`, `install.Client(ctx)`) — same as sibling `go-github-app-auth-test`.
- Config pattern: stdlib `flag` with `envOr` helper; testable `ParseFrom(args []string, getenv func(string) string)` + `Parse()` — same as sibling `go-github-commit-files`.
- Logging: `log/slog` JSON handler to `os.Stderr`. Results JSON goes to `os.Stdout` only.
- Required config: `AppID`, `PrivateKeyPath`, `InstallationID`, `Org`. `Topic` defaults to `argocd`, `LogLevel` to `info`.

---

### Task 1: Config package

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: nothing (uses stdlib only).
- Produces:
  - `type Config struct { AppID, PrivateKeyPath, InstallationID, Org, Topic, LogLevel string }`
  - `func ParseFrom(args []string, getenv func(string) string) (*Config, error)`
  - `func Parse() (*Config, error)`

- [ ] **Step 1: Write the failing test**

```go
package config

import "testing"

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
		"GITHUB_APP_ID":            "1",
		"GITHUB_PRIVATE_KEY_PATH":  "/k.pem",
		"GITHUB_INSTALLATION_ID":   "2",
		"GITHUB_REPO_ORG":          "envorg",
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

func TestParseFrom_MissingRequired(t *testing.T) {
	_, err := ParseFrom([]string{"--app-id", "1"}, env(nil))
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/`
Expected: FAIL (build error — `ParseFrom` undefined).

- [ ] **Step 3: Write minimal implementation**

```go
package config

import (
	"flag"
	"fmt"
	"os"
)

// Config holds all resolved runtime configuration.
type Config struct {
	AppID          string
	PrivateKeyPath string
	InstallationID string
	Org            string
	Topic          string
	LogLevel       string
}

// ParseFrom parses configuration from args and an environment lookup function.
// This signature enables unit testing without touching os.Args or real env vars.
func ParseFrom(args []string, getenv func(string) string) (*Config, error) {
	fs := flag.NewFlagSet("go-github-graphql", flag.ContinueOnError)

	envOr := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	appID := fs.String("app-id", envOr("GITHUB_APP_ID", ""), "GitHub App ID")
	privateKey := fs.String("private-key", envOr("GITHUB_PRIVATE_KEY_PATH", ""), "Path to PEM private key file")
	installationID := fs.String("installation-id", envOr("GITHUB_INSTALLATION_ID", ""), "App installation ID")
	org := fs.String("org", envOr("GITHUB_REPO_ORG", ""), "GitHub organization to search")
	topic := fs.String("topic", envOr("GITHUB_TOPIC", "argocd"), "Repository topic to filter by (default argocd)")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "Log level: debug/info/warn/error")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	cfg := &Config{
		AppID:          *appID,
		PrivateKeyPath: *privateKey,
		InstallationID: *installationID,
		Org:            *org,
		Topic:          *topic,
		LogLevel:       *logLevel,
	}

	required := []struct{ flag, value string }{
		{"--app-id (GITHUB_APP_ID)", cfg.AppID},
		{"--private-key (GITHUB_PRIVATE_KEY_PATH)", cfg.PrivateKeyPath},
		{"--installation-id (GITHUB_INSTALLATION_ID)", cfg.InstallationID},
		{"--org (GITHUB_REPO_ORG)", cfg.Org},
	}
	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("required flag %s is not set", r.flag)
		}
	}

	return cfg, nil
}

// Parse parses configuration from os.Args and real environment variables.
func Parse() (*Config, error) {
	return ParseFrom(os.Args[1:], os.Getenv)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config package with flags and env fallback"
```

---

### Task 2: Lister — query builder + repository type

**Files:**
- Create: `internal/lister/lister.go`
- Test: `internal/lister/lister_test.go`

**Interfaces:**
- Consumes: nothing yet (pure helper + type).
- Produces:
  - `type Repository struct { Name string \`json:"name"\`; URL string \`json:"url"\`; Description string \`json:"description"\`; Archived bool \`json:"archived"\` }`
  - `func SearchQuery(org, topic string) string` — returns `"org:<org> topic:<topic>"`.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lister/`
Expected: FAIL (build error — `SearchQuery` undefined).

- [ ] **Step 3: Write minimal implementation**

```go
package lister

import "fmt"

// Repository is a single repository result, shaped for JSON output.
type Repository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

// SearchQuery builds the GitHub search query string that matches all
// repositories in org carrying the given topic.
func SearchQuery(org, topic string) string {
	return fmt.Sprintf("org:%s topic:%s", org, topic)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lister/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/lister/lister.go internal/lister/lister_test.go
git commit -m "feat: add lister query builder and repository type"
```

---

### Task 3: Lister — auth client and paginated GraphQL search

**Files:**
- Modify: `internal/lister/lister.go`
- Modify: `go.mod` / `go.sum` (via `go mod tidy` / `go get`)

**Interfaces:**
- Consumes: `Repository`, `SearchQuery` from Task 2.
- Produces:
  - `func NewClient(ctx context.Context, appID, installationID, privateKeyPath string) (*githubv4.Client, error)` — builds the App-authenticated GraphQL client.
  - `func Search(ctx context.Context, client *githubv4.Client, org, topic string) ([]Repository, error)` — runs the paginated search and returns all matches.

Note: `NewClient` and `Search` perform network/file I/O and are not unit-tested (consistent with sibling projects). They are exercised via `go build` / `go vet` and manual run. The pure `SearchQuery` from Task 2 remains the tested seam.

- [ ] **Step 1: Add dependencies**

Run:
```bash
go get github.com/shurcooL/githubv4@latest
go get github.com/beatlabs/github-auth@v0.0.0-20240915075323-ab38acb2d2a1
```
Expected: `go.mod` gains both modules; no build errors.

- [ ] **Step 2: Implement the client builder and search**

Append to `internal/lister/lister.go` (add imports `context`, `github.com/beatlabs/github-auth/app/inst`, `github.com/beatlabs/github-auth/key`, `github.com/shurcooL/githubv4`):

```go
// NewClient builds a GitHub App installation-authenticated GraphQL v4 client.
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string) (*githubv4.Client, error) {
	k, err := key.FromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading private key: %w", err)
	}

	install, err := inst.NewConfig(appID, installationID, k)
	if err != nil {
		return nil, fmt.Errorf("creating installation config: %w", err)
	}

	return githubv4.NewClient(install.Client(ctx)), nil
}

// Search returns every repository in org carrying the given topic, paging
// through all results.
func Search(ctx context.Context, client *githubv4.Client, org, topic string) ([]Repository, error) {
	var query struct {
		Search struct {
			Nodes []struct {
				Repository struct {
					Name        string
					URL         string
					Description string
					IsArchived  bool
				} `graphql:"... on Repository"`
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"search(query: $q, type: REPOSITORY, first: 100, after: $after)"`
	}

	variables := map[string]interface{}{
		"q":     githubv4.String(SearchQuery(org, topic)),
		"after": (*githubv4.String)(nil),
	}

	var repos []Repository
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("querying repositories: %w", err)
		}

		for _, n := range query.Search.Nodes {
			repos = append(repos, Repository{
				Name:        n.Repository.Name,
				URL:         n.Repository.URL,
				Description: n.Repository.Description,
				Archived:    n.Repository.IsArchived,
			})
		}

		if !query.Search.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	return repos, nil
}
```

- [ ] **Step 3: Verify it builds and vets**

Run: `go build ./... && go vet ./... && go test ./internal/lister/`
Expected: builds cleanly, vet reports nothing, Task 2 tests still PASS.

- [ ] **Step 4: Commit**

```bash
go mod tidy
git add internal/lister/lister.go go.mod go.sum
git commit -m "feat: add App-authenticated paginated GraphQL search"
```

---

### Task 4: main.go wiring and JSON output

**Files:**
- Create: `main.go`

**Interfaces:**
- Consumes: `config.Parse`, `config.Config`, `lister.NewClient`, `lister.Search`, `lister.Repository`.
- Produces: the `main` entrypoint (no exported API).

- [ ] **Step 1: Implement main**

```go
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/prune998/devops/GITHUB/go-github-graphql/internal/config"
	"github.com/prune998/devops/GITHUB/go-github-graphql/internal/lister"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	ctx := context.Background()

	client, err := lister.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath)
	if err != nil {
		slog.Error("failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	repos, err := lister.Search(ctx, client, cfg.Org, cfg.Topic)
	if err != nil {
		slog.Error("failed to search repositories", "error", err)
		os.Exit(1)
	}

	slog.Info("search complete", "org", cfg.Org, "topic", cfg.Topic, "count", len(repos))

	output := struct {
		Org          string              `json:"org"`
		Topic        string              `json:"topic"`
		Count        int                 `json:"count"`
		Repositories []lister.Repository `json:"repositories"`
	}{
		Org:          cfg.Org,
		Topic:        cfg.Topic,
		Count:        len(repos),
		Repositories: repos,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		slog.Error("failed to encode JSON output", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it builds and vets**

Run: `go build ./... && go vet ./...`
Expected: builds cleanly, no vet findings.

- [ ] **Step 3: Verify usage error path (no network needed)**

Run: `go run . --help 2>&1 | head; echo "exit=$?"`
Expected: flag usage lines printed; running with no required flags exits non-zero with a "required flag" error on stderr:
`go run . 2>&1; echo "exit=$?"` → prints `configuration error ... required flag --app-id ...` and `exit=1`.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire config, lister, and JSON output in main"
```

---

### Task 5: README and full-suite verification

**Files:**
- Create: `README.md`

**Interfaces:**
- Consumes: nothing (documentation).
- Produces: nothing.

- [ ] **Step 1: Write README**

````markdown
# go-github-graphql

Lists all repositories in a GitHub organization that carry a given topic
(default `argocd`), using GitHub App authentication and the GitHub GraphQL API
v4 (`shurcooL/githubv4`). Results are printed as JSON to stdout; logs go to
stderr.

## Configuration

Each option is a CLI flag with an environment-variable fallback. The flag wins
when both are set.

| Flag                | Env var                   | Required | Default  |
|---------------------|---------------------------|----------|----------|
| `--app-id`          | `GITHUB_APP_ID`           | yes      | —        |
| `--private-key`     | `GITHUB_PRIVATE_KEY_PATH` | yes      | —        |
| `--installation-id` | `GITHUB_INSTALLATION_ID`  | yes      | —        |
| `--org`             | `GITHUB_REPO_ORG`         | yes      | —        |
| `--topic`           | `GITHUB_TOPIC`            | no       | `argocd` |
| `--log-level`       | `LOG_LEVEL`               | no       | `info`   |

## Usage

```bash
go run . \
  --app-id 123456 \
  --private-key ./app.private-key.pem \
  --installation-id 654321 \
  --org my-org \
  --topic argocd
```

Example output:

```json
{
  "org": "my-org",
  "topic": "argocd",
  "count": 1,
  "repositories": [
    {
      "name": "app-a",
      "url": "https://github.com/my-org/app-a",
      "description": "Example app",
      "archived": false
    }
  ]
}
```
````

- [ ] **Step 2: Run the full test suite and vet**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything builds, vet clean, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add README for go-github-graphql"
```

---

## Self-Review

**Spec coverage:**
- GitHub App auth → Task 3 (`NewClient` via `beatlabs/github-auth`). ✓
- GraphQL via `shurcooL/githubv4` → Task 3 (`Search`). ✓
- Filter by topic using `search` → Task 2 (`SearchQuery`) + Task 3. ✓
- Configurable topic, default `argocd` → Task 1. ✓
- Flags + env fallback → Task 1. ✓
- JSON output shape (org/topic/count/repositories) → Task 4. ✓
- Logs to stderr, JSON to stdout → Task 4. ✓
- Error handling / exit codes → Task 1 (required validation) + Task 4 (exit 1 paths). ✓
- Package split (config/lister/main) → Tasks 1–4. ✓
- Tests for config precedence + query builder → Tasks 1 & 2. ✓

**Placeholder scan:** No TBD/TODO; every code step has complete code. ✓

**Type consistency:** `Config` fields, `Repository` fields, `SearchQuery(org, topic)`, `NewClient(ctx, appID, installationID, privateKeyPath)`, and `Search(ctx, client, org, topic)` are used consistently across Tasks 1–4. ✓
