package githubissue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	State          string    `json:"state"`
	URL            string    `json:"html_url"`
	Labels         []Label   `json:"labels"`
	UpdatedAt      time.Time `json:"updated_at"`
	CommentCount   int       `json:"comments"`
	LastComment    *Comment  `json:"-"`
	LoadedComments []Comment `json:"-"`
	PullRequest    *struct{} `json:"pull_request,omitempty"`
}

type Label struct {
	Name string `json:"name"`
}

type ListIssuesOptions struct {
	State               string
	Labels              []string
	WithoutLabels       []string
	Limit               int
	UpdatedAfter        time.Time
	IncludeLastComment  bool
	IncludeComments     bool
	LastCommenterNot    string
	LastCommentContains string
	LastCommentMatches  string
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
	lastCommentPattern, err := compileLastCommentPattern(opts.LastCommentMatches)
	if err != nil {
		return nil, err
	}

	state := opts.State
	if state == "" {
		state = "open"
	}

	lastCommenterNot := strings.TrimSpace(opts.LastCommenterNot)
	lastCommentContains := strings.TrimSpace(opts.LastCommentContains)
	needsLastComment := opts.IncludeLastComment || opts.IncludeComments || lastCommenterNot != "" || lastCommentContains != "" || lastCommentPattern != nil
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
		if !opts.UpdatedAfter.IsZero() {
			query.Set("since", opts.UpdatedAfter.UTC().Format(time.RFC3339))
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
			if needsLastComment {
				if err := c.loadIssueComments(ctx, owner, repo, &issue, opts.IncludeComments); err != nil {
					return nil, err
				}
			}
			if lastCommenterNot != "" && issue.LastComment != nil && LoginMatches(issue.LastComment.User.Login, lastCommenterNot) {
				continue
			}
			if lastCommentContains != "" && (issue.LastComment == nil || !strings.Contains(issue.LastComment.Body, lastCommentContains)) {
				continue
			}
			if lastCommentPattern != nil && (issue.LastComment == nil || !lastCommentPattern.MatchString(issue.LastComment.Body)) {
				continue
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

func (c *Client) loadIssueComments(ctx context.Context, owner, repo string, issue *Issue, includeComments bool) error {
	if issue.CommentCount == 0 {
		if includeComments {
			issue.LoadedComments = []Comment{}
		}
		return nil
	}

	if includeComments {
		comments, err := c.ListComments(ctx, owner, repo, issue.Number)
		if err != nil {
			return err
		}
		issue.LoadedComments = comments
		if len(issue.LoadedComments) > 0 {
			issue.LastComment = &issue.LoadedComments[len(issue.LoadedComments)-1]
		}
		return nil
	}

	comments, err := c.listCommentsPage(ctx, owner, repo, issue.Number, commentPageForCount(issue.CommentCount))
	if err != nil {
		return err
	}
	if len(comments) == 0 {
		return nil
	}

	lastComment := comments[len(comments)-1]
	issue.LastComment = &lastComment
	return nil
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

func compileLastCommentPattern(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile last comment pattern: %w", err)
	}
	return compiled, nil
}

func LoginMatches(actual, wanted string) bool {
	actual = normalizeLogin(actual)
	wanted = normalizeLogin(wanted)
	return actual != "" && wanted != "" && actual == wanted
}

func normalizeLogin(login string) string {
	login = strings.ToLower(strings.TrimSpace(login))
	return strings.TrimSuffix(login, "[bot]")
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
