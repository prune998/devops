package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/committer"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/config"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/githubapp"
)

func main() {
	// Bootstrap a minimal logger before config is parsed so early errors are visible.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Parse()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Re-initialise logger with the configured level.
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))

	slog.Info("starting go-github-commit-files",
		"owner", cfg.Owner,
		"repo", cfg.Repo,
		"branch", cfg.Branch,
		"file_count", len(cfg.Files),
		"delete_file_count", len(cfg.DeleteFiles),
		"max_retries", cfg.MaxRetries,
		"log_level", cfg.LogLevel,
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
		"private_key_path", cfg.PrivateKeyPath,
	)

	ctx := context.Background()

	slog.Info("creating GitHub App client",
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
	)
	client, err := githubapp.NewClient(ctx, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath)
	if err != nil {
		slog.Error("failed to create GitHub App client", "error", err)
		os.Exit(1)
	}
	slog.Info("GitHub App client created")

	files := make([]committer.FilePair, len(cfg.Files))
	for i, fp := range cfg.Files {
		files[i] = committer.FilePair{Src: fp.Src, Dest: fp.Dest}
		slog.Info("file queued for commit", "src", fp.Src, "dest", fp.Dest)
	}

	for _, path := range cfg.DeleteFiles {
		slog.Info("file queued for deletion", "path", path)
	}

	c := committer.New(client, cfg.Owner, cfg.Repo, cfg.Branch, cfg.MaxRetries)
	commitSHA, err := c.CommitFiles(ctx, files, cfg.DeleteFiles)
	if err != nil {
		slog.Error("failed to commit files", "error", err)
		os.Exit(1)
	}

	slog.Info("done",
		"commit_sha", commitSHA,
		"url", "https://github.com/"+cfg.Owner+"/"+cfg.Repo+"/commit/"+commitSHA,
	)
}
