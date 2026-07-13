package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/prune998/devops/GITHUB/go-github-graphql/internal/config"
	"github.com/prune998/devops/GITHUB/go-github-graphql/internal/lister"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	// The rate-limit header logger tracks the configured level in normal mode
	// but is forced to Info in quiet mode so every GitHub API call's headers
	// stay visible even though general logging is silenced.
	headerLevel := level

	// Quiet mode silences all but error logs so only the statistics output,
	// the per-request API headers, and any fatal error reach the operator.
	if cfg.Quiet {
		level = slog.LevelError
		headerLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	headerLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: headerLevel}))

	ctx := context.Background()

	client, stats, err := lister.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath, cfg.MaxRetries, headerLogger)
	if err != nil {
		slog.Error("failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	repos, err := lister.Search(ctx, client, cfg.Org, cfg.Topic, cfg.ExcludeArchived)
	if err != nil {
		slog.Error("failed to search repositories", "error", err)
		os.Exit(1)
	}

	slog.Info("search complete", "org", cfg.Org, "topic", cfg.Topic, "count", len(repos))

	// A single JSON document carries the results and the GitHub API request
	// statistics. In quiet mode the repository listing is omitted, but the rest
	// of the document (including the statistics) is still emitted.
	output := struct {
		Org          string              `json:"org"`
		Topic        string              `json:"topic"`
		Count        int                 `json:"count"`
		Repositories []lister.Repository `json:"repositories,omitempty"`
		Stats        *lister.Stats       `json:"stats"`
	}{
		Org:   cfg.Org,
		Topic: cfg.Topic,
		Count: len(repos),
		Stats: stats,
	}
	if !cfg.Quiet {
		output.Repositories = repos
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		slog.Error("failed to encode JSON output", "error", err)
		os.Exit(1)
	}
}
