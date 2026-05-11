package githubissue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	URL         string    `json:"html_url"`
	Labels      []Label   `json:"labels"`
	Comments    int       `json:"comments"`
	PullRequest *struct{} `json:"pull_request,omitempty"`
}

type Label struct {
	Name string `json:"name"`
}

type ListIssuesOptions struct {
	State            string
	Labels           []string
	WithoutLabels    []string
	Limit            int
	LastCommenterNot string
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

func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]Issue, error) {
	if opts.Limit < 0 {
		return nil, fmt.Errorf("limit must be 0 or greater")
	}

	state := opts.State
	if state == "" {
		state = "open"
	}

	lastCommenterNot := strings.TrimSpace(opts.LastCommenterNot)
	perPage := perPageForLimit(opts.Limit)

	var allIssues []Issue
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("state", state)
		query.Set("per_page", strconv.Itoa(perPage))
		query.Set("page", fmt.Sprintf("%d", page))
		if len(opts.Labels) > 0 {
			query.Set("labels", strings.Join(opts.Labels, ","))
		}

		url := fmt.Sprintf("%s/repos/%s/%s/issues?%s", c.baseURL, owner, repo, query.Encode())
		req, err := c.newRequest(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create list issues request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request issues: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if closeErr != nil && readErr == nil {
			readErr = closeErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if readErr != nil {
				return nil, fmt.Errorf("request issues failed with status %s and unreadable response body: %w", resp.Status, readErr)
			}
			return nil, fmt.Errorf("request issues failed with status %s: %s", resp.Status, string(body))
		}
		if readErr != nil {
			return nil, fmt.Errorf("read issues response: %w", readErr)
		}

		var issues []Issue
		if err := json.Unmarshal(body, &issues); err != nil {
			return nil, fmt.Errorf("parse issues response: %w", err)
		}

		for _, issue := range issues {
			if issue.PullRequest != nil {
				continue
			}
			if !hasAllLabels(issue.Labels, opts.Labels) {
				continue
			}
			if hasAnyLabel(issue.Labels, opts.WithoutLabels) {
				continue
			}
			if lastCommenterNot != "" {
				matches, err := c.lastCommenterMatches(ctx, owner, repo, issue, lastCommenterNot)
				if err != nil {
					return nil, err
				}
				if matches {
					continue
				}
			}
			allIssues = append(allIssues, issue)
			if opts.Limit > 0 && len(allIssues) >= opts.Limit {
				return allIssues, nil
			}
		}

		if len(issues) < perPage {
			break
		}
	}

	return allIssues, nil
}

func (c *Client) lastCommenterMatches(ctx context.Context, owner, repo string, issue Issue, login string) (bool, error) {
	if issue.Comments == 0 {
		return false, nil
	}

	comments, err := c.listCommentsPage(ctx, owner, repo, issue.Number, commentPageForCount(issue.Comments))
	if err != nil {
		return false, err
	}
	if len(comments) == 0 {
		return false, nil
	}

	lastComment := comments[len(comments)-1]
	return strings.EqualFold(lastComment.User.Login, login), nil
}

func (c *Client) listCommentsPage(ctx context.Context, owner, repo string, issueNumber, page int) ([]Comment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100&page=%d", c.baseURL, owner, repo, issueNumber, page)
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create comments request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request comments for issue #%d: %w", issueNumber, err)
	}

	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if closeErr != nil && readErr == nil {
		readErr = closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if readErr != nil {
			return nil, fmt.Errorf("request comments for issue #%d failed with status %s and unreadable response body: %w", issueNumber, resp.Status, readErr)
		}
		return nil, fmt.Errorf("request comments for issue #%d failed with status %s: %s", issueNumber, resp.Status, string(body))
	}
	if readErr != nil {
		return nil, fmt.Errorf("read comments response: %w", readErr)
	}

	var comments []Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("parse comments response: %w", err)
	}

	return comments, nil
}

func perPageForLimit(limit int) int {
	if limit > 0 && limit < 100 {
		return limit
	}
	return 100
}

func commentPageForCount(count int) int {
	if count <= 0 {
		return 1
	}
	return ((count - 1) / 100) + 1
}

func hasAllLabels(labels []Label, names []string) bool {
	if len(names) == 0 {
		return true
	}
	if len(labels) == 0 {
		return false
	}

	actual := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		actual[strings.ToLower(label.Name)] = struct{}{}
	}

	for _, name := range names {
		if _, ok := actual[strings.ToLower(name)]; !ok {
			return false
		}
	}

	return true
}

func hasAnyLabel(labels []Label, names []string) bool {
	if len(labels) == 0 || len(names) == 0 {
		return false
	}

	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[strings.ToLower(name)] = struct{}{}
	}

	for _, label := range labels {
		if _, ok := wanted[strings.ToLower(label.Name)]; ok {
			return true
		}
	}

	return false
}
