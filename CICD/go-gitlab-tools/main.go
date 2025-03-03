package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	pathpkg "path"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	tokenVAR       = flag.String("token", "GITLAB_TOKEN", "env var used to store token")
	gitlabURL      = flag.String("url", "", "URL to the gitlab server API (down to v4)")
	project        = flag.String("project", "", "project name (project name with namespace, ex: 'my/gitbal/project')")
	folderListPath = flag.String("folder", "", "path to the folder to list")
	file           = flag.String("file", "", "full path to file to download. ex: 'my/file.txt')")
	branch         = flag.String("branch", "default", "branch used to lookup file, use default if you don't know')")
	logLevel       = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none")

	updateTopic = flag.Bool("updateTopic", false, "set to true to add a new topic")

	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
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

	slog.Debug("Project found", "project", gitProjects)

	if *branch == "default" {
		*branch = gitProjects.DefaultBranch
	}

	// list topics
	slog.Info("topics found", "topics", gitProjects.Topics)

	// add a topic
	if *updateTopic {
		editOpts := &gitlab.EditProjectOptions{
			Topics: gitlab.Ptr(append(gitProjects.Topics, fmt.Sprintf("github:my-new-topic-%s", randSeq(4)))),
		}
		gitProjects, _, err = glClient.Projects.EditProject(gitProjects.ID, editOpts)
		if err != nil {
			slog.Error("error updating the Topics", "error", err)
		}
	}

	// get file list
	directories := []string{
		*folderListPath,
		pathpkg.Dir(*folderListPath),
	}

	done := 0
	for _, directory := range directories {
		if done > 0 {
			break
		}
		fmt.Printf("searching for %s\n", directory)

		glo := &gitlab.ListTreeOptions{
			ListOptions: gitlab.ListOptions{PerPage: 2},
			Path:        &directory,
			Ref:         branch,
			Recursive:   gitlab.Ptr(false),
		}

		nextPage := 0
		for {
			files, resp, err := glClient.Repositories.ListTree(gitProjects.ID, glo)
			if err != nil {
				slog.Error("error looking up project tree", "project", gitProjects.ID, "root", *folderListPath, "branch", *branch, "err", err)
				os.Exit(1)
			}
			slog.Info("Repo Tree found", "root", *folderListPath, "branch", *branch, "directory", directory, "count", len(files), "page", nextPage, "TotalItems", resp.TotalItems)

			if *folderListPath == directory {
				if resp.TotalItems > 0 {
					slog.Info("File found", "root", *folderListPath, "branch", *branch, "directory", directory, "files", files, "page", nextPage)
					done = 1
					break
				}
			}
			for i := range files {
				if files[i].Path == *folderListPath {
					slog.Info("File found", "root", *folderListPath, "branch", *branch, "directory", directory, "files", files, "page", nextPage)
					done = 1
					break
				}
			}

			nextPage++
			if resp.NextPage == 0 {
				// no future pages
				break
			}
			glo.Page = resp.NextPage
		}
	}

	os.Exit(0)

	opts := &gitlab.GetFileOptions{Ref: branch}
	// encodedFile := url.QueryEscape(fmt.Sprintf("%s", *file))
	encodedFile := *file

	f, _, err := glClient.RepositoryFiles.GetFile(gitProjects.ID, encodedFile, opts)
	if err != nil {
		slog.Error("error looking up PLAIN file", "file", encodedFile, "branch", *branch, "err", err)
		os.Exit(1)
	}
	slog.Debug("PLAIN File found found", "file", encodedFile, "branch", *branch, "length", f.Size)

	// get raw file
	rawTeamsData, _, err := glClient.RepositoryFiles.GetRawFile(gitProjects.ID, encodedFile, &gitlab.GetRawFileOptions{
		Ref: branch,
	})
	if err != nil {
		slog.Error("error looking up RAW file", "file", encodedFile, "branch", *branch, "err", err)
		os.Exit(1)
	}
	slog.Debug("RAW File found found", "file", encodedFile, "branch", *branch, "length", len(rawTeamsData))

}

// func init() {
// 	rand.Seed(time.Now().UnixNano())
// }

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
