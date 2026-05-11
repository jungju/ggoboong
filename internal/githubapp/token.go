package githubapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const DefaultAPIBaseURL = "https://api.github.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    DefaultAPIBaseURL,
	}
}

type installationTokenResponse struct {
	Token string `json:"token"`
}

func (c *Client) CreateInstallationToken(ctx context.Context, installationID int64, appJWT string) (string, error) {
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.baseURL, installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("create installation token request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ggo")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request installation access token: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if readErr != nil {
			return "", fmt.Errorf("request installation access token failed with status %s and unreadable response body: %w", resp.Status, readErr)
		}
		return "", fmt.Errorf("request installation access token failed with status %s: %s", resp.Status, string(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("read installation token response: %w", readErr)
	}

	var parsed installationTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse installation token response: %w", err)
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("installation token response did not include a token")
	}

	return parsed.Token, nil
}
