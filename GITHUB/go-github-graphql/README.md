# go-github-graphql

Lists all repositories in a GitHub organization that carry a given topic
(default `argocd`) **and** contain a `project-info.yaml` file at the root of
their default branch, using GitHub App authentication and the GitHub GraphQL
API v4 (`shurcooL/githubv4`). Each repository's full topic list is included in
the output, and archived repositories can optionally be excluded. Results are
printed as JSON to stdout; logs go to stderr, including the `X-RateLimit-*`
response headers after each GraphQL query.

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
| `--max-retries`     | `MAX_RETRIES`             | no       | `5`      |
| `--log-level`       | `LOG_LEVEL`               | no       | `info`   |

Set `--exclude-archived` (or `EXCLUDE_ARCHIVED=true`) to omit archived
repositories from the results.

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
