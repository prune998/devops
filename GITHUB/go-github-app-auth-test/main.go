package main

import (
	"context"
	"fmt"
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
		}

		repos = append(repos, githubRepos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	slog.Info("all org repos found", "count", len(repos))

	// query repo
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
	fmt.Println(*resp.Request.URL)

}
