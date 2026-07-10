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
