package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/xanzy/go-gitlab"
)

var (
	tokenVAR       = flag.String("token", "GITLAB_TOKEN", "env var used to store token")
	gitlabURL      = flag.String("url", "", "URL to the gitlab server API (down to v4)")
	project        = flag.String("project", "", "project name (full path, ex: 'my/gitbal/project')")
	folderListPath = flag.String("folder", "", "path to the folder to list")
	file           = flag.String("file", "", "full path to file to download. ex: 'my/file.txt')")
	branch         = flag.String("branch", "default", "branch used to lookup file, use default if you don't know')")
	logLevel       = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none")
)

func main() {
	flag.Parse()
	switch strings.ToUpper(*logLevel) {
	case "TRACE":
	case "DEBUG":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	case "ERR":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	case "WARN":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	case "NONE":
		slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})))
	default:
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}
	slog.SetDefault(slog.With("app", "go-gitlab-tools"))

	accessToken := os.Getenv(*tokenVAR)
	glClient, _ := gitlab.NewClient(accessToken, gitlab.WithBaseURL(*gitlabURL))

	slog.Info("looking up Project Name", "project", *project)

	gitProjects, _, err := glClient.Projects.GetProject(*project, &gitlab.GetProjectOptions{})
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	slog.Info("Project found", "project", gitProjects)

	if *branch == "default" {
		*branch = gitProjects.DefaultBranch
	}
	// get file list
	glo := &gitlab.ListTreeOptions{
		Path:      folderListPath,
		Ref:       branch,
		Recursive: gitlab.Ptr(true),
	}

	files, _, err := glClient.Repositories.ListTree(gitProjects.ID, glo)
	if err != nil {
		slog.Error("error looking up project tree", "project", gitProjects.ID, "root", *folderListPath, "branch", *branch, "err", err)
		os.Exit(1)
	}
	slog.Info("Repo Tree found", "root", *folderListPath, "branch", *branch, "list", files)

	opts := &gitlab.GetFileOptions{Ref: branch}
	// encodedFile := url.QueryEscape(fmt.Sprintf("%s", *file))
	encodedFile := *file

	f, _, err := glClient.RepositoryFiles.GetFile(gitProjects.ID, encodedFile, opts)
	if err != nil {
		slog.Error("error looking up PLAIN file", "file", encodedFile, "branch", *branch, "err", err)
		os.Exit(1)
	}
	slog.Info("PLAIN File found found", "file", encodedFile, "branch", *branch, "length", f.Size)

	// get raw file
	rawTeamsData, _, err := glClient.RepositoryFiles.GetRawFile(gitProjects.ID, encodedFile, &gitlab.GetRawFileOptions{
		Ref: branch,
	})
	if err != nil {
		slog.Error("error looking up RAW file", "file", encodedFile, "branch", *branch, "err", err)
		os.Exit(1)
	}
	slog.Info("RAW File found found", "file", encodedFile, "branch", *branch, "length", len(rawTeamsData))
}
