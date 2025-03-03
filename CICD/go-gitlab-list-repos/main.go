package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"gitlab.bouncex.net/devops/idp/gitlab-webhook.git/logging"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GitProject struct {
	ID                *int    `json:"id" yaml:"id"`
	Name              *string `json:"name" yaml:"name"`
	Path              *string `json:"path" yaml:"path"`
	PathWithNamespace *string `json:"path_with_namespace" yaml:"path_with_namespace"`
	Namespace         *string `json:"namespace" yaml:"namespace"`
	Archived          *bool   `json:"archived" yaml:"archived"`
}

var (
	tokenVAR  = flag.String("token", "GITLAB_TOKEN", "env var used to store token")
	gitlabURL = flag.String("url", "", "URL to the gitlab server API (down to v4, like  https://gitlab.company.net/api/v4")

	archived   = flag.Bool("archived", false, "also list archived projects")
	deleted    = flag.Bool("deleted", false, "also list projects marked for deletion")
	subgroup   = flag.String("subgroup", "", "if set, only list projects under this subgroup")
	visibility = flag.String("visibility", "", "one of '' for all, or public, internal, or private")
	personal   = flag.Bool("personal", false, "also include user's project in personal space")

	output = flag.String("output", "path", "what to output. supported options: path, pathwithnamespace, name, json")

	logLevel  = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none")
	logFormat = flag.String("logFormat", "json", "one of JSON, PLAIN, NONE")
)

func main() {
	flag.Parse()
	logging.InitDefaultLogger()
	logging.SetGlobalLogLevel(*logLevel)
	logging.SetGlobalFormat(*logFormat)
	slog.SetDefault(slog.With("app", "go-gitlab-list-repos"))

	accessToken := os.Getenv(*tokenVAR)

	// Connect to Gitlab
	tr := http.DefaultTransport.(*http.Transport).Clone()
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = tr

	client, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(*gitlabURL), gitlab.WithHTTPClient(retryClient.HTTPClient))
	if err != nil {
		slog.Error("Error initializing GitLab client", "error", err)
		os.Exit(1)
	}

	slog.Debug("listing all Gitlab Projects")
	// Define the channel to send entries to the worker goroutine
	dataChan := make(chan []*gitlab.Project)
	errCh := make(chan error)
	// Define a WaitGroup to wait for all data to be processed
	var wg sync.WaitGroup

	// Launch the project fetcher
	wg.Add(1)
	go listProjects(client, *archived, *deleted, *subgroup, *visibility, dataChan, errCh, &wg)

	// parse the results
	go func() {
		for {
			entry := <-dataChan
			processProjects(entry)
		}
	}()

	err = <-errCh
	if err != nil {
		slog.Error("error happened", "err", err)
		os.Exit(1)
	}

	wg.Wait()
	slog.Debug("done without error")

	// outputData := []string{}

	// pp.Println(projects)
}

func processProjects(projects []*gitlab.Project) {
	for _, project := range projects {

		if !*personal && project.Namespace.Kind == "user" {
			continue
		}

		switch *output {
		case "pathWithNamespace":
			fmt.Println(project.PathWithNamespace)

		case "name":
			fmt.Println(project.Name)

		case "json":
			p := &GitProject{
				Name:              &project.Name,
				Path:              &project.Path,
				PathWithNamespace: &project.PathWithNamespace,
				Namespace:         &project.Namespace.FullPath,
				Archived:          &project.Archived,
				ID:                &project.ID,
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(p); err != nil {
				slog.Error("JSON parse error ", "error", err)
				continue
			}

		default:
			fmt.Println(project.Path)
		}
	}
}

func listProjects(client *gitlab.Client, archived, deleted bool, subgroup, visibility string, data chan<- []*gitlab.Project, errCh chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	opts := &gitlab.ListProjectsOptions{
		Archived:             gitlab.Ptr(archived),
		IncludePendingDelete: gitlab.Ptr(deleted),
		IncludeHidden:        gitlab.Ptr(true),
		OrderBy:              gitlab.Ptr("path"),
		Sort:                 gitlab.Ptr("asc"),
		Simple:               gitlab.Ptr(true),
	}
	if visibility != "" {
		opts.Visibility = gitlab.Ptr(gitlab.VisibilityValue(visibility))
	}

	PageOptions := []gitlab.RequestOptionFunc{}
	// allProjects := []*gitlab.Project{}

	for {
		projects, resp, err := client.Projects.ListProjects(opts, PageOptions...)
		if err != nil {
			errCh <- fmt.Errorf("Error getting GitLab projects: %v\n", err)
			return
		}
		// allProjects = append(allProjects, projects...)
		data <- projects

		if resp.NextLink == "" {
			break
		}

		PageOptions = []gitlab.RequestOptionFunc{
			gitlab.WithKeysetPaginationParameters(resp.NextLink),
		}
	}

	errCh <- nil
	return
}

// func listProjects(client *gitlab.Client, archived, deleted bool, subgroup, visibility string, data chan<- []*gitlab.Project) ([]*gitlab.Project, error) {
// 	opts := &gitlab.ListProjectsOptions{
// 		Archived:             gitlab.Ptr(archived),
// 		IncludePendingDelete: gitlab.Ptr(deleted),
// 		IncludeHidden:        gitlab.Ptr(true),
// 		OrderBy:              gitlab.Ptr("path"),
// 		Sort:                 gitlab.Ptr("asc"),
// 	}
// 	if visibility != "" {
// 		opts.Visibility = gitlab.Ptr(gitlab.VisibilityValue(visibility))
// 	}

// 	PageOptions := []gitlab.RequestOptionFunc{}
// 	// allProjects := []*gitlab.Project{}

// 	for {
// 		projects, resp, err := client.Projects.ListProjects(opts, PageOptions...)
// 		if err != nil {
// 			return nil, fmt.Errorf("Error getting GitLab projects: %v\n", err)
// 		}
// 		// allProjects = append(allProjects, projects...)
// 		data <- projects

// 		if resp.NextLink == "" {
// 			break
// 		}

// 		PageOptions = []gitlab.RequestOptionFunc{
// 			gitlab.WithKeysetPaginationParameters(resp.NextLink),
// 		}
// 	}

// 	return allProjects, nil
// }
