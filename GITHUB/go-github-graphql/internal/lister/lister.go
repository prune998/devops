package lister

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/shurcooL/githubv4"
)

// projectInfoExpression is the git revision:path expression used to detect the
// required project-info.yaml file at the root of a repository's default branch.
const projectInfoExpression = "HEAD:project-info.yaml"

// resetZone is the fixed UTC-4 zone used to render X-RateLimit-Reset timestamps
// as human-readable local times instead of raw epoch seconds.
var resetZone = time.FixedZone("UTC-4", -4*60*60)

// Stats accumulates counts describing every HTTP request the client makes to
// the GitHub API. It is safe for concurrent use.
type Stats struct {
	mu           sync.Mutex
	Requests     int         `json:"requests"`      // total HTTP requests issued (including retries)
	RateLimited  int         `json:"rate_limited"`  // requests that hit rate limiting and were retried
	StatusCounts map[int]int `json:"status_counts"` // number of responses per HTTP status code
}

// record notes a completed HTTP response with the given status code.
func (s *Stats) record(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Requests++
	if s.StatusCounts == nil {
		s.StatusCounts = make(map[int]int)
	}
	s.StatusCounts[status]++
}

// recordRateLimited notes that a request was rate limited and will be retried.
func (s *Stats) recordRateLimited() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RateLimited++
}

// Repository is a single repository result, shaped for JSON output.
type Repository struct {
	Name          string   `json:"name"`
	NameWithOwner string   `json:"nameWithOwner"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	Archived      bool     `json:"archived"`
	Topics        []string `json:"topics"`
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

// rateLimitTransport wraps an http.RoundTripper. It logs GitHub's
// X-RateLimit-* response headers after each request (so every GraphQL query
// surfaces the current rate-limit state) and retries requests rejected by
// rate limiting, waiting until the limit resets, up to maxRetries times.
type rateLimitTransport struct {
	next       http.RoundTripper
	maxRetries int
	stats      *Stats
	logger     *slog.Logger
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := t.next.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		t.stats.record(resp.StatusCode)
		if attrs := rateLimitAttrs(resp.Header); len(attrs) > 0 {
			t.logger.Info("github rate limit", attrs...)
		}

		wait, limited := retryAfter(resp, time.Now())
		if !limited || attempt >= t.maxRetries {
			return resp, nil
		}
		t.stats.recordRateLimited()

		// Body must be drained/closed and the request body reset before the
		// transport can reissue the request.
		resp.Body.Close()
		if req.GetBody != nil {
			body, gerr := req.GetBody()
			if gerr != nil {
				return resp, nil // cannot safely retry; surface the last response
			}
			req.Body = body
		}

		t.logger.Warn("github rate limit reached, waiting before retry",
			"attempt", attempt+1, "max_retries", t.maxRetries, "wait", wait.String())

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}
}

// retryAfter reports how long to wait before retrying a rate-limited response
// and whether the response indicates rate limiting at all. It handles GitHub's
// secondary limit (Retry-After, in seconds) and primary limit
// (X-RateLimit-Remaining: 0, waiting until X-RateLimit-Reset). now is passed in
// for testability.
func retryAfter(resp *http.Response, now time.Time) (time.Duration, bool) {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}

	// Secondary rate limit: explicit Retry-After delay in seconds.
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second, true
		}
	}

	// Primary rate limit: no budget left, wait until the reset time.
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
			if epoch, err := strconv.ParseInt(v, 10, 64); err == nil {
				wait := time.Unix(epoch, 0).Sub(now) + time.Second // small buffer past reset
				if wait < 0 {
					wait = 0
				}
				return wait, true
			}
		}
	}

	return 0, false
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
		value := h.Get(name)
		// Render the reset time as a human-readable UTC-4 timestamp rather than
		// the raw epoch seconds GitHub sends.
		if name == "X-Ratelimit-Reset" {
			if epoch, err := strconv.ParseInt(value, 10, 64); err == nil {
				value = time.Unix(epoch, 0).In(resetZone).Format("2006-01-02 15:04:05 MST")
			}
		}
		attrs = append(attrs, name, value)
	}
	return attrs
}

// NewClient builds a GitHub App installation-authenticated GraphQL v4 client
// whose transport logs rate-limit headers after each request and retries
// rate-limited requests up to maxRetries times. The returned Stats accumulates
// request counts for every call the client makes to the GitHub API.
//
// logger receives the per-request rate-limit header logs and retry notices; it
// is kept separate from the global logger so those API-call details can stay
// visible even when general logging is silenced (e.g. quiet mode). A nil logger
// falls back to slog.Default().
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string, maxRetries int, logger *slog.Logger) (*githubv4.Client, *Stats, error) {
	if logger == nil {
		logger = slog.Default()
	}

	k, err := key.FromFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading private key: %w", err)
	}

	install, err := inst.NewConfig(appID, installationID, k)
	if err != nil {
		return nil, nil, fmt.Errorf("creating installation config: %w", err)
	}

	httpClient := install.Client(ctx)
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	stats := &Stats{}
	httpClient.Transport = &rateLimitTransport{next: base, maxRetries: maxRetries, stats: stats, logger: logger}

	return githubv4.NewClient(httpClient), stats, nil
}

// repositoryNode is the shared shape of a repository as returned by both the
// search and organization GraphQL endpoints. object(expression: $file) is used
// to detect project-info.yaml and repositoryTopics carries the repo's topics.
type repositoryNode struct {
	Name          string
	NameWithOwner string
	URL           string
	Description   string
	IsArchived    bool
	ProjectInfo   struct {
		Typename string `graphql:"__typename"`
	} `graphql:"object(expression: $file)"`
	RepositoryTopics struct {
		Nodes []struct {
			Topic struct {
				Name string
			}
		}
	} `graphql:"repositoryTopics(first: 100)"`
}

// qualifies reports whether the node contains project-info.yaml at the root of
// its default branch. A missing path resolves to a null object (empty
// __typename); a present file resolves to a Blob.
func (n repositoryNode) qualifies() bool {
	return n.ProjectInfo.Typename == "Blob"
}

// topics returns the repository's topic names.
func (n repositoryNode) topics() []string {
	topics := make([]string, 0, len(n.RepositoryTopics.Nodes))
	for _, tn := range n.RepositoryTopics.Nodes {
		topics = append(topics, tn.Topic.Name)
	}
	return topics
}

// toRepository converts the node into the JSON-shaped Repository.
func (n repositoryNode) toRepository() Repository {
	return Repository{
		Name:          n.Name,
		NameWithOwner: n.NameWithOwner,
		URL:           n.URL,
		Description:   n.Description,
		Archived:      n.IsArchived,
		Topics:        n.topics(),
	}
}

// sortByNameWithOwner sorts repos in place by NameWithOwner for deterministic,
// alphabetical output.
func sortByNameWithOwner(repos []Repository) {
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].NameWithOwner < repos[j].NameWithOwner
	})
}

// contains reports whether s is present in list.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Search returns every repository in org carrying the given topic, using the
// GraphQL search endpoint and paging through all results. When excludeArchived
// is true, archived repositories are omitted. When limit is greater than 0, at
// most limit repositories are returned and paging stops once that many have
// been collected. Results are sorted by NameWithOwner.
func Search(ctx context.Context, client *githubv4.Client, org, topic string, excludeArchived bool, limit int) ([]Repository, error) {
	var query struct {
		Search struct {
			Nodes []struct {
				Repository repositoryNode `graphql:"... on Repository"`
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
paging:
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("querying repositories: %w", err)
		}

		for _, n := range query.Search.Nodes {
			if !n.Repository.qualifies() {
				continue
			}

			repos = append(repos, n.Repository.toRepository())

			if limit > 0 && len(repos) >= limit {
				break paging
			}
		}

		if !query.Search.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	sortByNameWithOwner(repos)

	return repos, nil
}

// SearchOrg returns every repository in org carrying the given topic, using the
// GraphQL organization endpoint (organization.repositories) instead of the
// search endpoint. The organization endpoint has no topic qualifier, so topic
// filtering is applied client-side against each repository's topics. When
// excludeArchived is true, archived repositories are filtered out server-side
// via the connection's isArchived argument. When limit is greater than 0, at
// most limit repositories are returned and paging stops once that many have
// been collected. Results are sorted by NameWithOwner.
func SearchOrg(ctx context.Context, client *githubv4.Client, org, topic string, excludeArchived bool, limit int) ([]Repository, error) {
	var query struct {
		Organization struct {
			Repositories struct {
				Nodes    []repositoryNode
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"repositories(first: 100, after: $after, isArchived: $isArchived, privacy: PRIVATE, orderBy: {field: NAME, direction: ASC})"`
		} `graphql:"organization(login: $org)"`
	}

	// isArchived is nullable: nil means no filtering, a non-nil false filters
	// archived repositories out server-side.
	var isArchived *githubv4.Boolean
	if excludeArchived {
		v := githubv4.Boolean(false)
		isArchived = &v
	}

	variables := map[string]interface{}{
		"org":        githubv4.String(org),
		"after":      (*githubv4.String)(nil),
		"file":       githubv4.String(projectInfoExpression),
		"isArchived": isArchived,
	}

	var repos []Repository
paging:
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("querying organization repositories: %w", err)
		}

		for _, n := range query.Organization.Repositories.Nodes {
			if !n.qualifies() {
				continue
			}

			repo := n.toRepository()

			// The organization endpoint cannot filter by topic, so enforce it here.
			if topic != "" && !contains(repo.Topics, topic) {
				continue
			}

			repos = append(repos, repo)

			if limit > 0 && len(repos) >= limit {
				break paging
			}
		}

		if !query.Organization.Repositories.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Organization.Repositories.PageInfo.EndCursor)
	}

	sortByNameWithOwner(repos)

	return repos, nil
}
