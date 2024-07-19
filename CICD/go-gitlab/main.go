package main

import (
	"fmt"
	"os"

	"github.com/xanzy/go-gitlab"
)

func main() {
	accessToken := os.Getenv("GITLAB_TOKEN")
	gitlabURL := os.Getenv("GITLAB_URL")
	glClient, _ := gitlab.NewClient(accessToken, gitlab.WithBaseURL(gitlabURL))

	projectID := "my/gitbal/project"

	fmt.Printf("Project Name: '%s'\n", projectID)

	gitProjects, _, err := glClient.Projects.GetProject(projectID, &gitlab.GetProjectOptions{})
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project: %s\n", gitProjects)

	ref := "main"
	fileName := "README.md"
	opts := &gitlab.GetFileOptions{Ref: &ref}

	f, _, err := glClient.RepositoryFiles.GetFile(projectID, fileName, opts)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	fmt.Printf("File: %v\n", f)
}
