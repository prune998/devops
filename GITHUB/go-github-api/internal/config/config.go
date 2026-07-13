package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all resolved runtime configuration.
type Config struct {
	AppID           string
	PrivateKeyPath  string
	InstallationID  string
	Org             string
	Topic           string
	LogLevel        string
	ExcludeArchived bool
	MaxRetries      int
	Quiet           bool
	Limit           int
}

// ParseFrom parses configuration from args and an environment lookup function.
// This signature enables unit testing without touching os.Args or real env vars.
func ParseFrom(args []string, getenv func(string) string) (*Config, error) {
	fs := flag.NewFlagSet("go-github-api", flag.ContinueOnError)

	envOr := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	envBool := func(key string, def bool) bool {
		if v := getenv(key); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return def
	}

	appID := fs.String("app-id", envOr("GITHUB_APP_ID", ""), "GitHub App ID")
	privateKey := fs.String("private-key", envOr("GITHUB_PRIVATE_KEY_PATH", ""), "Path to PEM private key file")
	installationID := fs.String("installation-id", envOr("GITHUB_INSTALLATION_ID", ""), "App installation ID")
	org := fs.String("org", envOr("GITHUB_REPO_ORG", ""), "GitHub organization to search")
	topic := fs.String("topic", envOr("GITHUB_TOPIC", "argocd"), "Repository topic to filter by")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "Log level: debug/info/warn/error")
	excludeArchived := fs.Bool("exclude-archived", envBool("EXCLUDE_ARCHIVED", false), "Exclude archived repositories from results")
	maxRetriesStr := fs.String("max-retries", envOr("MAX_RETRIES", "5"), "Max retries when GitHub rate limiting is reached, 0 = no retries")
	quiet := fs.Bool("quiet", envBool("QUIET", false), "Only output repository statistics, not the full repository list")
	limitStr := fs.String("limit", envOr("LIMIT", "0"), "Max number of repositories to grab, 0 = all")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	maxRetries, err := strconv.Atoi(*maxRetriesStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --max-retries value %q: %w", *maxRetriesStr, err)
	}
	if maxRetries < 0 {
		return nil, fmt.Errorf("--max-retries must be >= 0, got %d", maxRetries)
	}

	limit, err := strconv.Atoi(*limitStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --limit value %q: %w", *limitStr, err)
	}
	if limit < 0 {
		return nil, fmt.Errorf("--limit must be >= 0, got %d", limit)
	}

	cfg := &Config{
		AppID:           *appID,
		PrivateKeyPath:  *privateKey,
		InstallationID:  *installationID,
		Org:             *org,
		Topic:           *topic,
		LogLevel:        *logLevel,
		ExcludeArchived: *excludeArchived,
		MaxRetries:      maxRetries,
		Quiet:           *quiet,
		Limit:           limit,
	}

	required := []struct{ flag, value string }{
		{"--app-id (GITHUB_APP_ID)", cfg.AppID},
		{"--private-key (GITHUB_PRIVATE_KEY_PATH)", cfg.PrivateKeyPath},
		{"--installation-id (GITHUB_INSTALLATION_ID)", cfg.InstallationID},
		{"--org (GITHUB_REPO_ORG)", cfg.Org},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.flag)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required flags not set: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// Parse parses configuration from os.Args and real environment variables.
func Parse() (*Config, error) {
	return ParseFrom(os.Args[1:], os.Getenv)
}
