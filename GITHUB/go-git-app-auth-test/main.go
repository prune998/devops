package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	git_http "github.com/go-git/go-git/v5/plumbing/transport/http"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/k0kubun/pp"
)

func main() {

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	appID := os.Getenv("GITHUB_APP_ID")
	privateKeyPath := os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	installationID := os.Getenv("GITHUB_INSTALLATION_ID")
	repoURL := os.Getenv("GITHUB_REPO_URL")

	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		panic(err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		panic(err)
	}

	token, err := generateJWT(appID, privateKey)
	if err != nil {
		panic(err)
	}
	slog.Info("token generate", "token", token)
	pp.Println(token)

	installationToken, err := getInstallationToken(token, appID, installationID)
	if err != nil {
		panic(err)
	}

	auth := &git_http.BasicAuth{
		Username: "go-git-app", // can be any non-empty string
		Password: installationToken,
	}

	slog.Info("auth", "creds", auth)
	pp.Println(auth)

	_, err = git.PlainCloneContext(context.Background(), "repo", false, &git.CloneOptions{
		URL:  repoURL,
		Auth: auth,
	})
	if err != nil {
		slog.Error("error cloning repo", "error", err)
	}

	fmt.Println("Repository cloned successfully!")
}

func generateJWT(appID string, privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func getInstallationToken(token, appID, installationID string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationID), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to get installation token: %s", resp.Status)
	}

	var response struct {
		Token string `json:"token"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	return response.Token, nil
}
