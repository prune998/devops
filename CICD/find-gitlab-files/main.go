package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"

	"github.com/hashicorp/go-retryablehttp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	// "github.com/argoproj/argo-cd/v2/applicationset/utils"
)

type GitlabProvider struct {
	client                *gitlab.Client
	organization          string
	allBranches           bool
	includeSubgroups      bool
	includeSharedProjects bool
	topic                 string
}

// An abstract repository from an API provider.
type Repository struct {
	Organization string
	Repository   string
	URL          string
	Branch       string
	SHA          string
	Labels       []string
	RepositoryId interface{}
}

var (
	tokenVAR       = flag.String("token", "GITLAB_TOKEN", "env var used to store token")
	gitlabURL      = flag.String("url", "", "URL to the gitlab server API (down to v4)")
	project        = flag.String("project", "", "project name (project name with namespace, ex: 'my/gitbal/project' or ID)")
	folderListPath = flag.String("folder", "", "path to the folder to list")
	branch         = flag.String("branch", "default", "branch used to lookup file, use default if you don't know')")
)

func (g *GitlabProvider) RepoHasPath(repo *Repository, path string) (bool, error) {

	p, _, err := g.client.Projects.GetProject(repo.Repository, nil)
	if err != nil {
		return false, fmt.Errorf("error in GetProject: %v", err)
	}

	directories := []string{
		path,
		pathpkg.Dir(path),
	}

	// branch should not be "default" but the real "default branch" as set in the project
	if repo.Branch == "default" {
		repo.Branch = p.DefaultBranch
	}

	for _, directory := range directories {
		fmt.Printf("searching in folder %s\n", directory)

		options := &gitlab.ListTreeOptions{
			Path:      &directory,
			Ref:       &repo.Branch,
			Recursive: gitlab.Ptr(false),
		}

		for {
			treeNode, resp, err := g.client.Repositories.ListTree(p.ID, options)
			if err != nil {
				// latest Gitlab versions return 404 Not Found when the Path is a file. Ignore the error and continue searching in the parent folder
				if errors.Is(err, gitlab.ErrNotFound) {
					break
				}
				return false, fmt.Errorf("error in ListTree: %v", err)
			}
			if path == directory {
				if resp.TotalItems > 0 {
					return true, nil
				}
			}
			for i := range treeNode {
				if treeNode[i].Path == path {
					return true, nil
				}
			}
			if resp.NextPage == 0 {
				// no future pages
				break
			}
			options.Page = resp.NextPage
		}
	}
	return false, nil
}

func (g *GitlabProvider) RepoHasPathFromParent(repo *Repository, path string) (bool, error) {
	p, _, err := g.client.Projects.GetProject(repo.Repository, nil)
	if err != nil {
		return false, fmt.Errorf("error in GetProject: %v", err)
	}

	directories := []string{
		pathpkg.Dir(path),
	}

	// branch should not be "default" but the real "default branch" as set in the project
	if repo.Branch == "default" {
		repo.Branch = p.DefaultBranch
	}

	for _, directory := range directories {
		fmt.Printf("searching in folder %s\n", directory)

		options := &gitlab.ListTreeOptions{
			Path:      &directory,
			Ref:       &repo.Branch,
			Recursive: gitlab.Ptr(false),
		}

		for {
			treeNode, resp, err := g.client.Repositories.ListTree(p.ID, options)
			if err != nil {
				return false, fmt.Errorf("error in ListTree: %v", err)
			}

			// if path == directory {
			// 	if resp.TotalItems > 0 {
			// 		return true, nil
			// 	}
			// }
			for i := range treeNode {
				if treeNode[i].Path == path {
					return true, nil
				}
			}
			if resp.NextPage == 0 {
				// no future pages
				break
			}
			options.Page = resp.NextPage
		}
	}
	return false, nil
}

func main() {
	// Initialize GitLab client
	flag.Parse()
	token := os.Getenv(*tokenVAR)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = tr

	// client, err := gitlab.NewClient(token, gitlab.WithBaseURL(*gitlabURL), gitlab.WithHTTPClient(retryClient.HTTPClient))
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(*gitlabURL))
	if err != nil {
		fmt.Printf("Error initializing GitLab client: %v\n", err)
		os.Exit(1)
	}

	provider := GitlabProvider{
		client:                client,
		organization:          *project,
		allBranches:           false,
		includeSubgroups:      false,
		includeSharedProjects: false,
		topic:                 "",
	}

	repo := Repository{
		Organization: filepath.Dir(*project),
		Repository:   *project,
		Branch:       *branch,
	}

	res, err := provider.RepoHasPathFromParent(&repo, *folderListPath)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)
}
