package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/google/go-github/v69/github"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// load from a argo file
	appID := os.Getenv("GITHUB_APP_ID")
	privateKeyPath := os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	installationID := os.Getenv("GITHUB_INSTALLATION_ID")
	repoOrg := os.Getenv("GITHUB_REPO_ORG")
	repoName := os.Getenv("GITHUB_REPO_NAME")
	repoBranch := os.Getenv("GITHUB_REPO_BRANCH")

	key, err := key.FromFile(privateKeyPath)

	if err != nil {
		slog.Info("error with key file", "error", err)
		os.Exit(1)
	}

	install, err := inst.NewConfig(appID, installationID, key)
	if err != nil {
		slog.Info("error creating HTTP client", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	// Get an *http.Client
	client := install.Client(ctx)

	// create github client
	ghClient := github.NewClient(client)

	// list all org repos
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	repos := []*github.Repository{}

	for {
		githubRepos, resp, err := ghClient.Repositories.ListByOrg(ctx, repoOrg, opt)
		if err != nil {
			slog.Error("error listing repositories for org", "error", err)
			break
		}

		if githubRepos != nil {
			repos = append(repos, githubRepos...)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage

	}

	slog.Info("all org repos found", "count", len(repos))

	// query repos
	opts := &github.RepositoryListAllOptions{}
	a, _, err := ghClient.Repositories.ListAll(ctx, opts)
	if err != nil {
		slog.Info("can't list repos", "error", err)
		os.Exit(1)
	}

	// for _, b := range a {
	// 	fmt.Println(*b.HTMLURL)
	// }
	fmt.Println(len(a))

	repo, resp, err := ghClient.Repositories.Get(ctx, repoOrg, repoName)
	if err != nil {
		if resp != nil && resp.Response.StatusCode == http.StatusNotFound {
			slog.Info("repo not found", "error", err)
			os.Exit(1)
		}
		slog.Info("error fetching repo", "error", err)
		os.Exit(1)
	}

	fmt.Println(*repo.CloneURL)
	// fmt.Println(*resp.Request.URL)

	// Get the tree recursively
	treeSHA := "main"
	recursive := true

	tree, _, err := ghClient.Git.GetTree(ctx, repoOrg, repoName, treeSHA, recursive)
	if err != nil {
		log.Fatalf("Error getting git tree: %v", err)
	}

	slog.Info("Successfully retrieved recursive tree")

	// Iterate over the entries to see all files and directories
	for _, entry := range tree.Entries {
		// fmt.Printf("Path: %s, Type: %s, SHA: %s\n", *entry.Path, *entry.Type, *entry.SHA)

		if *entry.Path == "deployments/kargo" {
			slog.Info("type of the file", "size", entry.GetSize(), "type", entry.GetType())
			chartFolderContent, _, _, err := ghClient.Repositories.GetContents(ctx, repoOrg, repoName, *entry.Path, &github.RepositoryContentGetOptions{
				Ref: treeSHA,
			})
			if err != nil {
				log.Fatalf("Error getting kargo folder: %v", err)
			}
			if chartFolderContent != nil {
				slog.Info("size of the file", "size", *chartFolderContent.Size, "file", *entry.Path)
			}
		}
		if *entry.Path == "deployments/kargo/Chart.yaml" {
			slog.Info("type of the file", "size", entry.GetSize(), "type", *entry.Type, "content", entry.GetContent())

			chartFolderContent, _, _, err := ghClient.Repositories.GetContents(ctx, repoOrg, repoName, *entry.Path, &github.RepositoryContentGetOptions{
				Ref: treeSHA,
			})
			if err != nil {
				log.Fatalf("Error getting kargo folder: %v", err)
			}
			if chartFolderContent != nil {
				slog.Info("size of the file", "size", *chartFolderContent.Size, "file", *entry.Path, "content", *chartFolderContent.Content)
			}
		}
	}

	// try to push a new document into the selected branch
	fileContent := []byte(`name: Kargo Promotion for QA
on:
		push:
				branches:
						- main
jobs:
		promote-to-qa:
				runs-on: ubuntu-latest
				steps:
						- name: Promote with Kargo
								run: echo "Promoting to QA environment"
`)
	filePath := ".github/workflows/kargo-promotion-qa.yaml"

	createOpts := &github.RepositoryContentFileOptions{
		Message: github.String("feat: add kargo promotion workflow for qa"),
		Content: fileContent,
		Branch:  github.String(repoBranch),
	}

	// Try to get the existing file to check if it exists
	existingFile, _, _, err := ghClient.Repositories.GetContents(ctx, repoOrg, repoName, filePath, &github.RepositoryContentGetOptions{
		Ref: repoBranch,
	})

	if err != nil {
		// File doesn't exist, create it
		_, _, err = ghClient.Repositories.CreateFile(ctx, repoOrg, repoName, filePath, createOpts)
		if err != nil {
			slog.Error("could not create file", "error", err, "path", filePath)
			os.Exit(1)
		}
		slog.Info("successfully created file", "path", filePath, "branch", repoBranch)
	} else {
		// File exists, update it
		createOpts.SHA = existingFile.SHA
		_, _, err = ghClient.Repositories.UpdateFile(ctx, repoOrg, repoName, filePath, createOpts)
		if err != nil {
			slog.Error("could not update file", "error", err, "path", filePath)
			os.Exit(1)
		}
		slog.Info("successfully updated file", "path", filePath, "branch", repoBranch)
	}

	slog.Info("successfully pushed file to branch", "path", filePath, "branch", repoBranch)
}
