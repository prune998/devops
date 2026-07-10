package lister

import (
	"context"
	"fmt"

	"github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/shurcooL/githubv4"
)

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
