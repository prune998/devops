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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	ctx := context.Background()

	client, err := lister.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath)
	if err != nil {
		slog.Error("failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	repos, err := lister.Search(ctx, client, cfg.Org, cfg.Topic)
	if err != nil {
		slog.Error("failed to search repositories", "error", err)
		os.Exit(1)
	}

	slog.Info("search complete", "org", cfg.Org, "topic", cfg.Topic, "count", len(repos))

	output := struct {
		Org          string              `json:"org"`
		Topic        string              `json:"topic"`
		Count        int                 `json:"count"`
		Repositories []lister.Repository `json:"repositories"`
	}{
		Org:          cfg.Org,
		Topic:        cfg.Topic,
		Count:        len(repos),
		Repositories: repos,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		slog.Error("failed to encode JSON output", "error", err)
		os.Exit(1)
	}
}
