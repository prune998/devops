package committer

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/prune998/devops/GITHUB/go-github-commit-files/internal/retry"
)

// FilePair holds a local source path and a repository destination path.
type FilePair struct {
	Src  string
	Dest string
}

// Committer commits files to a GitHub repository branch using the Git Trees API.
type Committer struct {
	client     *github.Client
	owner      string
	repo       string
	branch     string
	maxRetries int
}

// New creates a Committer for the given repository and branch.
func New(client *github.Client, owner, repo, branch string, maxRetries int) *Committer {
	return &Committer{
		client:     client,
		owner:      owner,
		repo:       repo,
		branch:     branch,
		maxRetries: maxRetries,
	}
}

// CommitFiles atomically commits all files in a single Git commit.
// files contains upload pairs; deleteFiles contains repo paths to remove.
// Returns the new commit SHA on success.
func (c *Committer) CommitFiles(ctx context.Context, files []FilePair, deleteFiles []string) (string, error) {
	if len(files) == 0 && len(deleteFiles) == 0 {
		return "", fmt.Errorf("CommitFiles: files and deleteFiles must not both be empty")
	}

	// Step 1: resolve branch HEAD to a commit SHA
	slog.Info("fetching branch ref",
		"owner", c.owner, "repo", c.repo, "branch", c.branch)
	var ref *github.Reference
	if err := retry.Do(ctx, c.maxRetries+1, "get_ref", func() error {
		var err error
		ref, _, err = c.client.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+c.branch)
		return err
	}); err != nil {
		slog.Error("failed to get branch ref", "branch", c.branch, "error", err)
		return "", fmt.Errorf("getting branch ref refs/heads/%s: %w", c.branch, err)
	}
	commitSHA := ref.GetObject().GetSHA()
	slog.Info("branch ref resolved", "branch", c.branch, "commit_sha", commitSHA)

	// Step 2: resolve commit to its tree SHA
	slog.Info("fetching base commit", "commit_sha", commitSHA)
	var baseCommit *github.Commit
	if err := retry.Do(ctx, c.maxRetries+1, "get_commit", func() error {
		var err error
		baseCommit, _, err = c.client.Git.GetCommit(ctx, c.owner, c.repo, commitSHA)
		return err
	}); err != nil {
		slog.Error("failed to get base commit", "commit_sha", commitSHA, "error", err)
		return "", fmt.Errorf("getting commit %s: %w", commitSHA, err)
	}
	treeSHA := baseCommit.GetTree().GetSHA()
	slog.Info("base tree resolved", "tree_sha", treeSHA)

	// Step 3: create a blob for each file
	var treeEntries []*github.TreeEntry
	for _, fp := range files {
		slog.Info("reading local file", "src", fp.Src, "dest", fp.Dest)
		content, err := os.ReadFile(fp.Src)
		if err != nil {
			slog.Error("failed to read local file", "src", fp.Src, "error", err)
			return "", fmt.Errorf("reading file %s: %w", fp.Src, err)
		}

		encoded := base64.StdEncoding.EncodeToString(content)
		slog.Info("creating blob", "dest", fp.Dest, "bytes", len(content))
		var blob *github.Blob
		if err := retry.Do(ctx, c.maxRetries+1, "create_blob:"+fp.Dest, func() error {
			var err error
			blob, _, err = c.client.Git.CreateBlob(ctx, c.owner, c.repo, &github.Blob{
				Content:  github.Ptr(encoded),
				Encoding: github.Ptr("base64"),
			})
			return err
		}); err != nil {
			slog.Error("failed to create blob", "dest", fp.Dest, "error", err)
			return "", fmt.Errorf("creating blob for %s: %w", fp.Dest, err)
		}
		slog.Info("blob created", "dest", fp.Dest, "sha", blob.GetSHA())

		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(fp.Dest),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  github.Ptr(blob.GetSHA()),
		})
	}

	// Step 3b: add a nil-SHA tree entry for each file to delete.
	// The Git Trees API interprets SHA=null as "remove this path from the tree".
	for _, path := range deleteFiles {
		slog.Info("queuing file for deletion", "path", path)
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.Ptr(path),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  nil,
		})
	}

	// Step 4: create a new tree from all blobs
	slog.Info("creating tree", "base_tree_sha", treeSHA, "entry_count", len(treeEntries))
	var newTree *github.Tree
	if err := retry.Do(ctx, c.maxRetries+1, "create_tree", func() error {
		var err error
		newTree, _, err = c.client.Git.CreateTree(ctx, c.owner, c.repo, treeSHA, treeEntries)
		return err
	}); err != nil {
		slog.Error("failed to create tree", "error", err)
		return "", fmt.Errorf("creating tree: %w", err)
	}
	slog.Info("tree created", "sha", newTree.GetSHA())

	// Step 5: build commit message
	var msgParts []string
	if len(files) > 0 {
		destPaths := make([]string, len(files))
		for i, fp := range files {
			destPaths[i] = fp.Dest
		}
		msgParts = append(msgParts, "update "+strings.Join(destPaths, ", "))
	}
	if len(deleteFiles) > 0 {
		msgParts = append(msgParts, "delete "+strings.Join(deleteFiles, ", "))
	}
	message := "chore: " + strings.Join(msgParts, ", ") + " via automation"

	// Step 6: create the commit object
	slog.Info("creating commit", "message", message, "tree_sha", newTree.GetSHA())
	var newCommit *github.Commit
	if err := retry.Do(ctx, c.maxRetries+1, "create_commit", func() error {
		var err error
		newCommit, _, err = c.client.Git.CreateCommit(ctx, c.owner, c.repo, &github.Commit{
			Message: github.Ptr(message),
			Tree:    &github.Tree{SHA: github.Ptr(newTree.GetSHA())},
			Parents: []*github.Commit{{SHA: github.Ptr(commitSHA)}},
		}, &github.CreateCommitOptions{})
		return err
	}); err != nil {
		slog.Error("failed to create commit", "error", err)
		return "", fmt.Errorf("creating commit: %w", err)
	}
	slog.Info("commit created", "sha", newCommit.GetSHA(), "message", message)

	// Step 7: update the branch ref to point to the new commit
	slog.Info("updating branch ref", "branch", c.branch, "commit_sha", newCommit.GetSHA())
	if err := retry.Do(ctx, c.maxRetries+1, "update_ref", func() error {
		_, _, err := c.client.Git.UpdateRef(ctx, c.owner, c.repo, &github.Reference{
			Ref:    github.Ptr("refs/heads/" + c.branch),
			Object: &github.GitObject{SHA: github.Ptr(newCommit.GetSHA())},
		}, false)
		return err
	}); err != nil {
		slog.Error("failed to update branch ref", "branch", c.branch, "error", err)
		return "", fmt.Errorf("updating branch ref: %w", err)
	}

	commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", c.owner, c.repo, newCommit.GetSHA())
	slog.Info("branch updated successfully",
		"branch", c.branch,
		"commit_sha", newCommit.GetSHA(),
		"url", commitURL,
	)

	return newCommit.GetSHA(), nil
}
