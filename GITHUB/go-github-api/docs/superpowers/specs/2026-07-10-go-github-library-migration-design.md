# Migrate go-github-api to the google/go-github library

**Date:** 2026-07-10
**Status:** Approved

## Goal

Replace the hand-rolled REST calls in `internal/lister/lister.go` with the
[`google/go-github`](https://github.com/google/go-github) library (pinned to
`v75`), while preserving all existing behavior: CLI flags, JSON output shape,
rate-limit logging, retry-with-wait, request counting, and the
`project-info.yaml` filter.

## Scope

Changes are confined to `internal/lister/lister.go` (plus its test file and
`go.mod`/`go.sum`). `main.go`, `internal/config`, flags, environment variables,
and the emitted JSON document are unchanged.

## Decisions

- **Rate limiting:** keep the existing custom `rateLimitTransport`. go-github
  surfaces rate-limit state and typed errors but does not auto-wait/retry, so
  the transport is retained to preserve header logging, `--max-retries`
  retry-with-wait, and the request counter. The go-github client is wired on
  top of the same installation-authenticated `*http.Client`.
- **Version:** `github.com/google/go-github/v75`.

## Changes

### `NewClient`

- Build the installation-authenticated `*http.Client` and wrap it in
  `rateLimitTransport` exactly as today.
- Wrap that client with `github.NewClient(httpClient)`.
- `Client` struct holds the `*github.Client` and the `*rateLimitTransport`
  (for `Requests()`), replacing the raw `*http.Client` + `baseURL` fields.

### `Search`

- Replace the manual `GET /search/repositories` + `Link`-header regex paging
  with `client.gh.Search.Repositories(ctx, SearchQuery(...), opts)`, looping on
  `resp.NextPage` with `opts.ListOptions.PerPage = 100`.
- Each `*github.Repository` already carries `Name`, `HTMLURL`, `Description`,
  `Archived`, and `Topics`; map these onto `Repository`.
- Preserve the `--limit` early-stop: return once `len(repos) >= limit` when
  `limit > 0`.

### `hasProjectInfo`

- Replace the manual contents probe with
  `client.gh.Repositories.GetContents(ctx, owner, repo, "project-info.yaml", nil)`.
- Return `true` on success, `false` when the error is a `*github.ErrorResponse`
  with `Response.StatusCode == 404`, and the error otherwise.
- Split `full_name` ("owner/repo") into owner and repo for the call.

### Deletions

Remove now-obsolete code: `getJSON`, `get`, `nextLink`, `linkNextRe`,
`searchResponse`, `searchRepo`, `defaultBaseURL`, `acceptHeader`.

### Unchanged

`rateLimitTransport`, `retryAfter`, `rateLimitAttrs`, `formatResetHeader`,
`SearchQuery`, `Repository`, `utcMinus4`.

## Testing

- Existing lister tests target the transport and helpers that remain, so they
  continue to pass unchanged.
- Add a test for the new 404-detection logic in `hasProjectInfo` using a
  `roundTripFunc` stub returning 200 / 404 / 500 and asserting
  true / false / error.

## Verification

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `go work sync` so the workspace still builds.
