package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jungju/ggoboong/internal/config"
	"github.com/jungju/ggoboong/internal/githubissue"
)

type IssuesOptions struct {
	Owner               string
	Repo                string
	State               string
	Labels              []string
	WithoutLabels       []string
	Limit               int
	JSON                bool
	IncludeComments     bool
	LastCommenterNot    string
	LastCommentContains string
	LastCommentMatches  string
	UpdatedAfter        string
	ConfigPath          string
	Stdout              io.Writer
	HTTPClient          *http.Client
}

func Issues(ctx context.Context, opts IssuesOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if err := validateIssuesOptions(opts); err != nil {
		return err
	}

	cfg, configPath, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if configPath != "" && !opts.JSON {
		fmt.Fprintf(opts.Stdout, "using config: %s\n", configPath)
	}

	issueClient, err := newIssueClient(ctx, cfg, opts.HTTPClient)
	if err != nil {
		return err
	}
	updatedAfter, err := parseUpdatedAfter(opts.UpdatedAfter)
	if err != nil {
		return err
	}

	issues, err := issueClient.ListIssues(ctx, opts.Owner, opts.Repo, githubissue.ListIssuesOptions{
		State:               opts.State,
		Labels:              normalizeLabels(opts.Labels),
		WithoutLabels:       normalizeLabels(opts.WithoutLabels),
		Limit:               opts.Limit,
		UpdatedAfter:        updatedAfter,
		IncludeLastComment:  opts.JSON,
		IncludeComments:     opts.IncludeComments,
		LastCommenterNot:    strings.TrimSpace(opts.LastCommenterNot),
		LastCommentContains: strings.TrimSpace(opts.LastCommentContains),
		LastCommentMatches:  strings.TrimSpace(opts.LastCommentMatches),
	})
	if err != nil {
		return fmt.Errorf("list issues: %w", err)
	}

	if opts.JSON {
		encoder := json.NewEncoder(opts.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(formatIssuesJSON(issues, opts.IncludeComments))
	}

	if len(issues) == 0 {
		fmt.Fprintln(opts.Stdout, "no issues found")
		return nil
	}

	for _, issue := range issues {
		fmt.Fprintf(opts.Stdout, "#%d [%s] %s\n", issue.Number, issue.State, issue.Title)
		if labels := formatLabels(issue.Labels); labels != "" {
			fmt.Fprintf(opts.Stdout, "  labels: %s\n", labels)
		}
		fmt.Fprintf(opts.Stdout, "  %s\n", issue.URL)
	}

	return nil
}

func validateIssuesOptions(opts IssuesOptions) error {
	if opts.Owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if opts.Repo == "" {
		return fmt.Errorf("--repo is required")
	}
	switch opts.State {
	case "", "open", "closed", "all":
	default:
		return fmt.Errorf("--state must be one of open, closed, all")
	}
	if opts.Limit < 0 {
		return fmt.Errorf("--limit must be 0 or greater")
	}
	if opts.IncludeComments && !opts.JSON {
		return fmt.Errorf("--include-comments requires --json")
	}
	if _, err := parseUpdatedAfter(opts.UpdatedAfter); err != nil {
		return err
	}
	return nil
}

type issueJSON struct {
	Number        int            `json:"number"`
	Title         string         `json:"title"`
	URL           string         `json:"url"`
	Labels        []string       `json:"labels"`
	State         string         `json:"state"`
	LastCommenter string         `json:"lastCommenter"`
	LastCommentAt string         `json:"lastCommentAt"`
	Comments      *[]commentJSON `json:"comments,omitempty"`
}

type commentJSON struct {
	ID        int64  `json:"id"`
	User      string `json:"user"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func formatIssuesJSON(issues []githubissue.Issue, includeComments bool) []issueJSON {
	output := make([]issueJSON, 0, len(issues))
	for _, issue := range issues {
		item := issueJSON{
			Number: issue.Number,
			Title:  issue.Title,
			URL:    issue.URL,
			Labels: labelNames(issue.Labels),
			State:  issue.State,
		}
		if issue.LastComment != nil {
			item.LastCommenter = issue.LastComment.User.Login
			item.LastCommentAt = formatTime(issue.LastComment.CreatedAt)
		}
		if includeComments {
			comments := formatCommentsJSON(issue.LoadedComments)
			item.Comments = &comments
		}
		output = append(output, item)
	}
	return output
}

func formatCommentsJSON(comments []githubissue.Comment) []commentJSON {
	output := make([]commentJSON, 0, len(comments))
	for _, comment := range comments {
		output = append(output, commentJSON{
			ID:        comment.ID,
			User:      comment.User.Login,
			Body:      comment.Body,
			URL:       comment.URL,
			CreatedAt: formatTime(comment.CreatedAt),
			UpdatedAt: formatTime(comment.UpdatedAt),
		})
	}
	return output
}

func parseUpdatedAfter(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}

	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("--updated-after must be RFC3339 or YYYY-MM-DD")
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func normalizeLabels(values []string) []string {
	seen := make(map[string]struct{})
	var labels []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			label := strings.TrimSpace(part)
			if label == "" {
				continue
			}
			key := strings.ToLower(label)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			labels = append(labels, label)
		}
	}
	return labels
}

func formatLabels(labels []githubissue.Label) string {
	return strings.Join(labelNames(labels), ", ")
}

func labelNames(labels []githubissue.Label) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Name != "" {
			names = append(names, label.Name)
		}
	}
	sort.Strings(names)
	return names
}
