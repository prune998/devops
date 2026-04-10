package githubapp

import (
	"context"
	"fmt"

	ghauth "github.com/beatlabs/github-auth/app/inst"
	"github.com/beatlabs/github-auth/key"
	"github.com/google/go-github/v69/github"
)

// NewClient authenticates as a GitHub App installation and returns a configured
// *github.Client. appID and installationID are the numeric IDs as strings.
func NewClient(ctx context.Context, appID, installationID, privateKeyPath string) (*github.Client, error) {
	k, err := key.FromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading private key from %s: %w", privateKeyPath, err)
	}

	install, err := ghauth.NewConfig(appID, installationID, k)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub App installation config: %w", err)
	}

	return github.NewClient(install.Client(ctx)), nil
}
