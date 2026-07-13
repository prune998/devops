package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/prune998/devops/GITHUB/go-github-api/internal/config"
	"github.com/prune998/devops/GITHUB/go-github-api/internal/lister"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	ctx := context.Background()

	client, err := lister.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath, cfg.MaxRetries)
	if err != nil {
		slog.Error("failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	repos, err := lister.Search(ctx, client, cfg.Org, cfg.Topic, cfg.ExcludeArchived, cfg.Limit)
	if err != nil {
		slog.Error("failed to search repositories", "error", err)
		os.Exit(1)
	}

	requests := client.Requests()
	slog.Info("search complete", "org", cfg.Org, "topic", cfg.Topic, "count", len(repos), "requests", requests)

	output := struct {
		Org          string               `json:"org"`
		Topic        string               `json:"topic"`
		Count        int                  `json:"count"`
		Requests     int64                `json:"requests"`
		Repositories *[]lister.Repository `json:"repositories,omitempty"`
	}{
		Org:      cfg.Org,
		Topic:    cfg.Topic,
		Count:    len(repos),
		Requests: requests,
	}
	// In quiet mode only the statistics are emitted; the full repository list
	// is omitted. Otherwise always render the list (as [] when empty).
	if !cfg.Quiet {
		if repos == nil {
			repos = []lister.Repository{}
		}
		output.Repositories = &repos
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		slog.Error("failed to encode JSON output", "error", err)
		os.Exit(1)
	}
}
