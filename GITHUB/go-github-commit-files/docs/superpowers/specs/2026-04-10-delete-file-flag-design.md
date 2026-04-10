# Design: --delete-file flag

**Date:** 2026-04-10  
**Status:** Approved

## Overview

Add a repeatable `--delete-file` flag to `go-github-commit-files` so that file deletions can be included in the same atomic Git commit as uploads. Uses the Git Trees API's nil-SHA mechanism — no extra API calls required for deletions.

## Changes

### `internal/config/config.go`

- Add `DeleteFiles []string` field to `Config` struct
- Add repeatable `--delete-file <repo-path>` flag (no env var fallback — list flags have no natural env var shape)
- Empty-slice guard: error only if both `Files` and `DeleteFiles` are empty

### `internal/committer/committer.go`

- Update signature: `CommitFiles(ctx context.Context, files []FilePair, deleteFiles []string) (string, error)`
- After building blob tree entries from `files`, append one `*github.TreeEntry` per path in `deleteFiles`:
  ```go
  &github.TreeEntry{
      Path: github.Ptr(path),
      Mode: github.Ptr("100644"),
      Type: github.Ptr("blob"),
      SHA:  nil, // nil SHA = delete this path from the tree
  }
  ```
- No blob creation for deletions — only tree entries needed
- Each deletion path logged at Info level before adding the entry
- Empty guard updated: return error only if both `files` and `deleteFiles` are empty

**Commit message format:**
- Both uploads and deletions: `chore: update <f1>, <f2>, delete <d1>, <d2> via automation`
- Uploads only: `chore: update <f1>, <f2> via automation` (unchanged)
- Deletions only: `chore: delete <d1>, <d2> via automation`

### `main.go`

- Pass `cfg.DeleteFiles` as the second argument to `CommitFiles`
- Log each delete path at Info level: `"file queued for deletion"`

### Tests

- `internal/config/config_test.go`: one new test `TestParseFrom_deleteFile` verifying `--delete-file` is parsed into `Config.DeleteFiles`
- `internal/config/config_test.go`: one new test `TestParseFrom_deleteFileOnly` verifying that `--delete-file` alone (no `--file`) is accepted
- `internal/config/config_test.go`: update `TestParseFrom_missingFiles` to verify that passing neither `--file` nor `--delete-file` still errors

## Unchanged

- `internal/retry/retry.go` — no changes
- `internal/githubapp/client.go` — no changes
- All existing tests continue to pass unmodified
