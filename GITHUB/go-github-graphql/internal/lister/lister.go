package lister

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/shurcooL/githubv4"
)

// projectInfoExpression is the git revision:path expression used to detect the
// required project-info.yaml file at the root of a repository's default branch.
const projectInfoExpression = "HEAD:project-info.yaml"

// Repository is a single repository result, shaped for JSON output.
type Repository struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Archived    bool     `json:"archived"`
	Topics      []string `json:"topics"`
}

// SearchQuery builds the GitHub search query string that matches all
// repositories in org carrying the given topic. When excludeArchived is true,
// the archived:false qualifier is appended so archived repositories are
// filtered out server-side.
func SearchQuery(org, topic string, excludeArchived bool) string {
	q := fmt.Sprintf("org:%s topic:%s", org, topic)
	if excludeArchived {
		q += " archived:false"
	}
	return q
}

// rateLimitTransport wraps an http.RoundTripper and logs GitHub's
// X-RateLimit-* response headers after each request, so every GraphQL query
// surfaces the current rate-limit state.
type rateLimitTransport struct {
	next http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if attrs := rateLimitAttrs(resp.Header); len(attrs) > 0 {
		slog.Info("github rate limit", attrs...)
	}
	return resp, nil
}

// rateLimitAttrs extracts X-RateLimit-* headers from h into a slog-style
// key/value slice, sorted by header name for deterministic output.
func rateLimitAttrs(h http.Header) []any {
	var names []string
	for name := range h {
		if strings.HasPrefix(name, "X-Ratelimit-") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	attrs := make([]any, 0, len(names)*2)
	for _, name := range names {
		attrs = append(attrs, name, h.Get(name))
	}
	return attrs
}

// NewClient builds a GitHub App installation-authenticated GraphQL v4 client
// whose transport logs rate-limit headers after each request.
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string) (*githubv4.Client, error) {
	k, err := key.FromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading private key: %w", err)
	}

	install, err := inst.NewConfig(appID, installationID, k)
	if err != nil {
		return nil, fmt.Errorf("creating installation config: %w", err)
	}

	httpClient := install.Client(ctx)
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = &rateLimitTransport{next: base}

	return githubv4.NewClient(httpClient), nil
}

// Search returns every repository in org carrying the given topic, paging
// through all results. When excludeArchived is true, archived repositories are
// omitted.
func Search(ctx context.Context, client *githubv4.Client, org, topic string, excludeArchived bool) ([]Repository, error) {
	var query struct {
		Search struct {
			Nodes []struct {
				Repository struct {
					Name        string
					URL         string
					Description string
					IsArchived  bool
					ProjectInfo struct {
						Typename string `graphql:"__typename"`
					} `graphql:"object(expression: $file)"`
					RepositoryTopics struct {
						Nodes []struct {
							Topic struct {
								Name string
							}
						}
					} `graphql:"repositoryTopics(first: 100)"`
				} `graphql:"... on Repository"`
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"search(query: $q, type: REPOSITORY, first: 100, after: $after)"`
	}

	variables := map[string]interface{}{
		"q":     githubv4.String(SearchQuery(org, topic, excludeArchived)),
		"after": (*githubv4.String)(nil),
		"file":  githubv4.String(projectInfoExpression),
	}

	var repos []Repository
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("querying repositories: %w", err)
		}

		for _, n := range query.Search.Nodes {
			// Keep only repositories that contain project-info.yaml at the root
			// of their default branch. A missing path resolves to a null object
			// (empty __typename); a present file resolves to a Blob.
			if n.Repository.ProjectInfo.Typename != "Blob" {
				continue
			}

			topics := make([]string, 0, len(n.Repository.RepositoryTopics.Nodes))
			for _, tn := range n.Repository.RepositoryTopics.Nodes {
				topics = append(topics, tn.Topic.Name)
			}

			repos = append(repos, Repository{
				Name:        n.Repository.Name,
				URL:         n.Repository.URL,
				Description: n.Repository.Description,
				Archived:    n.Repository.IsArchived,
				Topics:      topics,
			})
		}

		if !query.Search.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	return repos, nil
}
