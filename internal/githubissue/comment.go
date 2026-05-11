package githubissue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Comment struct {
	ID        int64     `json:"id"`
	User      User      `json:"user"`
	Body      string    `json:"body"`
	URL       string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	Login string `json:"login"`
}

func (c *Client) ListComments(ctx context.Context, owner, repo string, issueNumber int) ([]Comment, error) {
	var allComments []Comment
	for page := 1; ; page++ {
		comments, err := c.listCommentsPage(ctx, owner, repo, issueNumber, page)
		if err != nil {
			return nil, err
		}
		allComments = append(allComments, comments...)
		if len(comments) < 100 {
			break
		}
	}

	return allComments, nil
}

func (c *Client) CreateComment(ctx context.Context, owner, repo string, issueNumber int, body string) (*Comment, error) {
	payload, err := json.Marshal(struct {
		Body string `json:"body"`
	}{Body: body})
	if err != nil {
		return nil, fmt.Errorf("build create comment payload: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	req, err := c.newRequest(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create comment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create comment on issue #%d: %w", issueNumber, err)
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if readErr != nil {
			return nil, fmt.Errorf("create comment on issue #%d failed with status %s and unreadable response body: %w", issueNumber, resp.Status, readErr)
		}
		return nil, fmt.Errorf("create comment on issue #%d failed with status %s: %s", issueNumber, resp.Status, string(responseBody))
	}
	if readErr != nil {
		return nil, fmt.Errorf("read create comment response: %w", readErr)
	}

	var comment Comment
	if err := json.Unmarshal(responseBody, &comment); err != nil {
		return nil, fmt.Errorf("parse create comment response: %w", err)
	}

	return &comment, nil
}

func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ggo")

	return req, nil
}
