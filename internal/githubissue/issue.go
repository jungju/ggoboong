package githubissue

import (
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
	token      string
}

func NewClient(httpClient *http.Client, token string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    DefaultAPIBaseURL,
		token:      token,
	}
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"html_url"`
}

func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", c.baseURL, owner, repo, number)
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create issue request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request issue #%d: %w", number, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if readErr != nil {
			return nil, fmt.Errorf("request issue #%d failed with status %s and unreadable response body: %w", number, resp.Status, readErr)
		}
		return nil, fmt.Errorf("request issue #%d failed with status %s: %s", number, resp.Status, string(body))
	}
	if readErr != nil {
		return nil, fmt.Errorf("read issue response: %w", readErr)
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parse issue response: %w", err)
	}

	return &issue, nil
}
