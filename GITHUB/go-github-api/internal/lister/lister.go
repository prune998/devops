package lister

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/google/go-github/v89/github"
)

// projectInfoPath is the repository-root path of the file a repository must
// contain (on its default branch) to be included in the results.
const projectInfoPath = "project-info.yaml"

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

// rateLimitTransport wraps an http.RoundTripper. It logs GitHub's
// X-RateLimit-* response headers after each request (so every REST call
// surfaces the current rate-limit state) and retries requests rejected by
// rate limiting, waiting until the limit resets, up to maxRetries times.
type rateLimitTransport struct {
	next       http.RoundTripper
	maxRetries int
	count      atomic.Int64 // total HTTP requests sent, including retries
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		t.count.Add(1)
		resp, err := t.next.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		if attrs := rateLimitAttrs(resp.Header); len(attrs) > 0 {
			slog.Info("github rate limit", attrs...)
		}

		wait, limited := retryAfter(resp, time.Now())
		if !limited || attempt >= t.maxRetries {
			return resp, nil
		}

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

		slog.Warn("github rate limit reached, waiting before retry",
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

// utcMinus4 is the fixed UTC-4 zone used to render the rate-limit reset time.
var utcMinus4 = time.FixedZone("UTC-4", -4*60*60)

// rateLimitAttrs extracts X-RateLimit-* headers from h into a slog-style
// key/value slice, sorted by header name for deterministic output. The
// X-Ratelimit-Reset value (a Unix epoch) is rendered as an RFC 3339 UTC-4 date
// rather than a raw timestamp.
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
		if name == "X-Ratelimit-Reset" {
			value = formatResetHeader(value)
		}
		attrs = append(attrs, name, value)
	}
	return attrs
}

// formatResetHeader renders a Unix-epoch reset timestamp as an RFC 3339 date
// in the UTC-4 zone. The original value is returned unchanged when it is not a
// valid epoch.
func formatResetHeader(value string) string {
	epoch, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}
	return time.Unix(epoch, 0).In(utcMinus4).Format(time.RFC3339)
}

// Client wraps a go-github REST client. The underlying *http.Client uses a
// rateLimitTransport that logs rate-limit headers, retries rate-limited
// requests, and counts every HTTP request sent to GitHub.
type Client struct {
	gh        *github.Client
	transport *rateLimitTransport
}

// Requests returns the total number of HTTP requests this client has sent to
// GitHub, including retried requests.
func (c *Client) Requests() int64 {
	return c.transport.count.Load()
}

// NewClient builds a GitHub App installation-authenticated go-github REST
// client whose transport logs rate-limit headers after each request and retries
// rate-limited requests up to maxRetries times.
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string, maxRetries int) (*Client, error) {
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
	rt := &rateLimitTransport{next: base, maxRetries: maxRetries}
	httpClient.Transport = rt

	gh, err := github.NewClient(github.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("creating github client: %w", err)
	}

	return &Client{gh: gh, transport: rt}, nil
}

// Search returns every repository in org carrying the given topic, paging
// through all results via the REST search endpoint. Each candidate is kept only
// if it contains project-info.yaml at the root of its default branch. When
// excludeArchived is true, archived repositories are omitted. When limit is
// greater than zero, at most limit repositories are returned and the search
// stops early once the limit is reached; limit <= 0 means return all matches.
func Search(ctx context.Context, client *Client, org, topic string, excludeArchived bool, limit int) ([]Repository, error) {
	query := SearchQuery(org, topic, excludeArchived)
	opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}

	var repos []Repository
	for {
		result, resp, err := client.gh.Search.Repositories(ctx, query, opts)
		if err != nil {
			return nil, fmt.Errorf("querying repositories: %w", err)
		}

		for _, r := range result.Repositories {
			// Keep only repositories that contain project-info.yaml at the root
			// of their default branch (the REST search cannot filter on file
			// presence, so each candidate is probed with a contents request).
			has, err := client.hasProjectInfo(ctx, r.GetOwner().GetLogin(), r.GetName())
			if err != nil {
				return nil, fmt.Errorf("checking %s for %s: %w", r.GetFullName(), projectInfoPath, err)
			}
			if !has {
				continue
			}

			// Preserve a non-nil slice so the JSON output is [] rather than null.
			topics := make([]string, 0, len(r.Topics))
			topics = append(topics, r.Topics...)

			repos = append(repos, Repository{
				Name:        r.GetName(),
				URL:         r.GetHTMLURL(),
				Description: r.GetDescription(),
				Archived:    r.GetArchived(),
				Topics:      topics,
			})

			if limit > 0 && len(repos) >= limit {
				return repos, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return repos, nil
}

// hasProjectInfo reports whether owner/repo contains project-info.yaml at the
// root of its default branch. The contents endpoint returns 200 when the file
// exists and 404 when it does not.
func (c *Client) hasProjectInfo(ctx context.Context, owner, repo string) (bool, error) {
	_, _, _, err := c.gh.Repositories.GetContents(ctx, owner, repo, projectInfoPath, nil)
	if err != nil {
		var erErr *github.ErrorResponse
		if errors.As(err, &erErr) && erErr.Response != nil && erErr.Response.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
