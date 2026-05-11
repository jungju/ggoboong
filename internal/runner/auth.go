package runner

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jungju/ggoboong/internal/config"
	"github.com/jungju/ggoboong/internal/githubapp"
	"github.com/jungju/ggoboong/internal/githubissue"
)

func newIssueClient(ctx context.Context, cfg *config.Config, httpClient *http.Client) (*githubissue.Client, error) {
	appJWT, err := githubapp.GenerateJWT(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("generate GitHub App JWT: %w", err)
	}

	appClient := githubapp.NewClient(httpClient)
	token, err := appClient.CreateInstallationToken(ctx, cfg.GitHub.InstallationID, appJWT)
	if err != nil {
		return nil, fmt.Errorf("create installation access token: %w", err)
	}

	return githubissue.NewClient(httpClient, token), nil
}
