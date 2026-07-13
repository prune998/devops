# go-github-api

Lists all repositories in a GitHub organization that carry a given topic
(default `argocd`) **and** contain a `project-info.yaml` file at the root of
their default branch, using GitHub App authentication and the GitHub REST API
v3 via the [`google/go-github`](https://github.com/google/go-github) library.
Each repository's full topic list is included in the output,
and archived repositories can optionally be excluded. Results are printed as
JSON to stdout; logs go to stderr, including the `X-RateLimit-*` response
headers after each REST request.

Candidate repositories are found with the REST search endpoint
(`GET /search/repositories`), and each candidate is probed with a contents
request (`GET /repos/{owner}/{repo}/contents/project-info.yaml`) to confirm the
file exists on the default branch — the search API cannot filter on file
presence.

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
| `--exclude-archived`| `EXCLUDE_ARCHIVED`        | no       | `false`  |
| `--quiet`           | `QUIET`                   | no       | `false`  |
| `--limit`           | `LIMIT`                   | no       | `0`      |
| `--max-retries`     | `MAX_RETRIES`             | no       | `5`      |
| `--log-level`       | `LOG_LEVEL`               | no       | `info`   |

Set `--exclude-archived` (or `EXCLUDE_ARCHIVED=true`) to omit archived
repositories from the results.

Set `--limit` (or `LIMIT`) to cap the number of matching repositories grabbed;
the search stops early once the limit is reached. The default `0` returns all
matches.

Set `--quiet` (or `QUIET=true`) to output only the statistics (`org`, `topic`,
`count`, `requests`) and omit the full `repositories` list:

```json
{
  "org": "my-org",
  "topic": "argocd",
  "count": 1,
  "requests": 4
}
```

The `requests` field reports the total number of HTTP requests sent to GitHub
during the run (search pages plus one `project-info.yaml` probe per candidate
repository, including any retries triggered by rate limiting).

When GitHub rate limiting is reached, requests are retried up to
`--max-retries` times, waiting until the limit resets (`X-RateLimit-Reset`)
or honoring a `Retry-After` header. Set `--max-retries 0` to disable retries.

## Usage

```bash
go run . \
  --app-id 123456 \
  --private-key ./app.private-key.pem \
  --installation-id 654321 \
  --org my-org \
  --topic argocd \
  --exclude-archived
```

Example output:

```json
{
  "org": "my-org",
  "topic": "argocd",
  "count": 1,
  "requests": 4,
  "repositories": [
    {
      "name": "app-a",
      "url": "https://github.com/my-org/app-a",
      "description": "Example app",
      "archived": false,
      "topics": ["argocd", "gitops"]
    }
  ]
}
```
