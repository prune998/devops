# Design: go-github-commit-files

**Date:** 2026-04-10  
**Status:** Approved

## Overview

A Go CLI application that uses a GitHub App token to atomically commit multiple local files to a specific branch in a GitHub repository. Built on `go-github/v69` and `beatlabs/github-auth`, consistent with the sibling `go-github-app-auth-test` project.

## Architecture

Four small packages under `internal/`, wired by `main.go`:

```
go-github-commit-files/
├── main.go                      # wires config → logger → client → committer
├── go.mod
├── internal/
│   ├── config/config.go         # CLI flags + env var fallback, file pair parsing
│   ├── githubapp/client.go      # GitHub App auth → *github.Client
│   ├── retry/retry.go           # rate-limit-aware retry wrapper
│   └── committer/committer.go   # Git Trees API: blob → tree → commit → ref update
```

## Configuration

CLI flags with env var fallback. Flags take precedence over env vars. All resolved values are logged at startup (private key path shown, not contents).

| Flag | Env Var | Required | Default | Description |
|------|---------|----------|---------|-------------|
| `--app-id` | `GITHUB_APP_ID` | yes | — | GitHub App ID |
| `--private-key` | `GITHUB_PRIVATE_KEY_PATH` | yes | — | Path to PEM private key file |
| `--installation-id` | `GITHUB_INSTALLATION_ID` | yes | — | App installation ID |
| `--owner` | `GITHUB_REPO_ORG` | yes | — | Repo owner (org or user) |
| `--repo` | `GITHUB_REPO_NAME` | yes | — | Repository name |
| `--branch` | `GITHUB_REPO_BRANCH` | yes | — | Target branch |
| `--file` | — | yes (1+) | — | Repeatable: `src:dest` local-to-remote path pair |
| `--max-retries` | `MAX_RETRIES` | no | `5` | Max API call retry attempts |
| `--log-level` | `LOG_LEVEL` | no | `info` | Log level: debug/info/warn/error |

Files are specified as `--file /local/path.yaml:repo/dest/path.yaml`, repeatable for multiple files.

## Logging

- `log/slog` with `slog.NewJSONHandler` writing to stderr
- Log level configurable at startup via `--log-level` / `LOG_LEVEL`
- Logged events:
  - App startup with resolved config
  - Each API call: attempt number, operation name, file path
  - Each retry: reason (rate limit / abuse limit / transient), wait duration, attempt count
  - Each blob created (with dest path and SHA)
  - Tree created (with SHA)
  - Commit created (with SHA and message)
  - Ref updated (branch now points to new commit)
  - Final success with commit SHA and GitHub URL

## Commit Flow (Git Trees API)

One atomic commit — all files land together or none do:

1. Get branch HEAD → current commit SHA and tree SHA
2. For each `src:dest` pair: read local file, POST blob to GitHub API
3. POST new tree with all blob entries, using current tree as base
4. POST new commit object pointing to new tree  
   - Auto-generated message: `chore: update <dest1>, <dest2> via automation`
5. PATCH branch ref to new commit SHA

## Retry Logic

Every GitHub API call is wrapped with retry. Behavior by error type:

| Error | Strategy |
|-------|----------|
| `*github.RateLimitError` | Sleep until `Rate.Reset` + 1s jitter, retry |
| `*github.AbuseRateLimitError` | Sleep `RetryAfter` duration, retry |
| HTTP 5xx transient | Exponential backoff: 1s → 2s → 4s → 8s…, capped at 60s |
| Other errors | Fail immediately (no retry) |

- Max retries: configurable, default 5
- Every retry logged at WARN level with: attempt number, wait duration, error reason
- Exhausted retries return a wrapped error with full attempt history

## Error Handling

- Missing required config → log error + exit 1 before any API calls
- Invalid `--file` pair format (missing `:`) → log error + exit 1
- Local file not readable → log error + exit 1
- API errors after max retries → log error + exit 1
- Partial state is not possible (atomic commit via Git Trees API)

## Dependencies

Consistent with sibling project:

```
github.com/beatlabs/github-auth   # GitHub App JWT auth
github.com/google/go-github/v69   # GitHub API client
```

No additional dependencies — retry and config use stdlib only (`flag`, `os`, `time`, `log/slog`).
