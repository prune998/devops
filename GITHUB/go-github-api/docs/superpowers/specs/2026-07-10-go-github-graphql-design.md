# Design: go-github-graphql

Date: 2026-07-10

## Purpose

A Go CLI that lists all repositories in a GitHub organization carrying a given
topic (default `argocd`). It authenticates as a GitHub App and queries the
GitHub GraphQL API v4 via `github.com/shurcooL/githubv4`. Output is JSON on
stdout.

## Conventions followed

Mirrors the sibling projects in this workspace:

- `github.com/beatlabs/github-auth` for GitHub App installation auth, producing
  an `*http.Client` (same as `go-github-app-auth-test`).
- Stdlib `flag` with an `envOr` helper (flags with env-var fallback) and a
  testable `ParseFrom(args, getenv)` + `Parse()` in `internal/config` (same as
  `go-github-commit-files`).
- `log/slog` JSON handler to stderr for logging.

## Architecture

Package split:

- `internal/config` — parses configuration from CLI flags with env-var
  fallback. Exposes `ParseFrom(args []string, getenv func(string) string)` for
  testing and `Parse()` for production.
- `internal/lister` — builds the authenticated `*http.Client`, wraps it in
  `githubv4.NewClient`, and runs the paginated search query. Returns the
  collected repositories. Also exposes a pure `SearchQuery(org, topic)` helper
  that builds the search query string.
- `main.go` — wires config → lister → JSON output on stdout; logging on stderr.

### Config fields

| Field            | Flag                 | Env var                      | Required | Default  |
|------------------|----------------------|------------------------------|----------|----------|
| `AppID`          | `--app-id`           | `GITHUB_APP_ID`              | yes      | —        |
| `PrivateKeyPath` | `--private-key`      | `GITHUB_PRIVATE_KEY_PATH`    | yes      | —        |
| `InstallationID` | `--installation-id`  | `GITHUB_INSTALLATION_ID`     | yes      | —        |
| `Org`            | `--org`              | `GITHUB_REPO_ORG`            | yes      | —        |
| `Topic`          | `--topic`            | `GITHUB_TOPIC`               | no       | `argocd` |
| `LogLevel`       | `--log-level`        | `LOG_LEVEL`                  | no       | `info`   |

## Data flow

1. Parse config.
2. Load PEM key via `key.FromFile(PrivateKeyPath)`.
3. `inst.NewConfig(appID, installationID, key)` → `install.Client(ctx)` yields an
   `*http.Client`.
4. `githubv4.NewClient(httpClient)`.
5. Run the search query with pagination, collecting all pages.
6. Marshal results to indented JSON on stdout.

## GraphQL query

```go
var query struct {
    Search struct {
        RepositoryCount int
        Nodes           []struct {
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
```

Variables:

- `$q githubv4.String` = `"org:<Org> topic:<Topic>"` (built by `SearchQuery`).
- `$after *githubv4.String` = nil on the first page, then `EndCursor`.

Loop while `PageInfo.HasNextPage`, appending `Nodes` each iteration.

## Output

Indented JSON to stdout:

```json
{
  "org": "myorg",
  "topic": "argocd",
  "count": 2,
  "repositories": [
    {
      "name": "app-a",
      "url": "https://github.com/myorg/app-a",
      "description": "...",
      "archived": false
    }
  ]
}
```

Logs (via slog) go to stderr so stdout remains clean JSON.

## Error handling

- Missing any required config field → error, exit 1. `Topic` defaults to
  `argocd`, `LogLevel` to `info`.
- Key load, client creation, or query errors → logged via slog, exit 1.
- Zero matches → valid JSON with `"count": 0`, exit 0.

## Testing

- `internal/config`: table-driven tests for `ParseFrom` covering flag/env
  precedence, required-field validation, and the `topic` default.
- `internal/lister`: unit test the pure `SearchQuery(org, topic)` builder. The
  network call itself stays thin and is not unit-tested (consistent with the
  sibling projects).

## Dependencies

- `github.com/shurcooL/githubv4`
- `github.com/beatlabs/github-auth`
- `golang.org/x/oauth2` (transitive, used by githubv4/auth)
