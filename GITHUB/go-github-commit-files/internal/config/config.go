package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FilePair holds a local source path and a repository destination path.
type FilePair struct {
	Src  string
	Dest string
}

// Config holds all resolved runtime configuration.
type Config struct {
	AppID          string
	PrivateKeyPath string
	InstallationID string
	Owner          string
	Repo           string
	Branch         string
	Files          []FilePair
	DeleteFiles    []string
	MaxRetries     int
	LogLevel       string
}

// fileFlags implements flag.Value for a repeatable --file flag.
type fileFlags []string

func (f *fileFlags) String() string { return strings.Join(*f, ", ") }
func (f *fileFlags) Set(v string) error {
	*f = append(*f, v)
	return nil
}

// parseFilePair splits "src:dest" into a FilePair, returning an error if the
// format is invalid or either side is empty.
func parseFilePair(s string) (FilePair, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return FilePair{}, fmt.Errorf("invalid file pair %q: expected non-empty src:dest", s)
	}
	return FilePair{Src: parts[0], Dest: parts[1]}, nil
}

// ParseFrom parses configuration from args and an environment lookup function.
// This signature enables unit testing without touching os.Args or real env vars.
func ParseFrom(args []string, getenv func(string) string) (*Config, error) {
	fs := flag.NewFlagSet("go-github-commit-files", flag.ContinueOnError)

	envOr := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	var files fileFlags
	var deleteFiles fileFlags

	appID := fs.String("app-id", envOr("GITHUB_APP_ID", ""), "GitHub App ID")
	privateKey := fs.String("private-key", envOr("GITHUB_PRIVATE_KEY_PATH", ""), "Path to PEM private key file")
	installationID := fs.String("installation-id", envOr("GITHUB_INSTALLATION_ID", ""), "App installation ID")
	owner := fs.String("owner", envOr("GITHUB_REPO_ORG", ""), "Repo owner (org or user)")
	repo := fs.String("repo", envOr("GITHUB_REPO_NAME", ""), "Repository name")
	branch := fs.String("branch", envOr("GITHUB_REPO_BRANCH", ""), "Target branch")
	maxRetriesStr := fs.String("max-retries", envOr("MAX_RETRIES", "5"), "Max retries per API call, 0 = no retries (default 5)")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "Log level: debug/info/warn/error")
	fs.Var(&files, "file", "Repeatable src:dest file pair (e.g. ./local.yaml:path/in/repo.yaml)")
	fs.Var(&deleteFiles, "delete-file", "Repeatable repo path to delete (e.g. old/config.yaml)")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	maxRetries, err := strconv.Atoi(*maxRetriesStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --max-retries value %q: %w", *maxRetriesStr, err)
	}

	cfg := &Config{
		AppID:          *appID,
		PrivateKeyPath: *privateKey,
		InstallationID: *installationID,
		Owner:          *owner,
		Repo:           *repo,
		Branch:         *branch,
		MaxRetries:     maxRetries,
		LogLevel:       *logLevel,
	}

	for _, f := range files {
		pair, err := parseFilePair(f)
		if err != nil {
			return nil, err
		}
		cfg.Files = append(cfg.Files, pair)
	}

	cfg.DeleteFiles = []string(deleteFiles)

	required := []struct{ flag, value string }{
		{"--app-id (GITHUB_APP_ID)", cfg.AppID},
		{"--private-key (GITHUB_PRIVATE_KEY_PATH)", cfg.PrivateKeyPath},
		{"--installation-id (GITHUB_INSTALLATION_ID)", cfg.InstallationID},
		{"--owner (GITHUB_REPO_ORG)", cfg.Owner},
		{"--repo (GITHUB_REPO_NAME)", cfg.Repo},
		{"--branch (GITHUB_REPO_BRANCH)", cfg.Branch},
	}
	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("required flag %s is not set", r.flag)
		}
	}
	if len(cfg.Files) == 0 && len(cfg.DeleteFiles) == 0 {
		return nil, fmt.Errorf("at least one --file or --delete-file is required")
	}

	return cfg, nil
}

// Parse parses configuration from os.Args and real environment variables.
func Parse() (*Config, error) {
	return ParseFrom(os.Args[1:], os.Getenv)
}
