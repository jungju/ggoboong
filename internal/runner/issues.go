package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/jungju/ggoboong/internal/config"
	"github.com/jungju/ggoboong/internal/githubissue"
)

type IssuesOptions struct {
	Owner         string
	Repo          string
	State         string
	Labels        []string
	WithoutLabels []string
	ConfigPath    string
	Stdout        io.Writer
	HTTPClient    *http.Client
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
	if configPath != "" {
		fmt.Fprintf(opts.Stdout, "using config: %s\n", configPath)
	}

	issueClient, err := newIssueClient(ctx, cfg, opts.HTTPClient)
	if err != nil {
		return err
	}

	issues, err := issueClient.ListIssues(ctx, opts.Owner, opts.Repo, githubissue.ListIssuesOptions{
		State:         opts.State,
		Labels:        normalizeLabels(opts.Labels),
		WithoutLabels: normalizeLabels(opts.WithoutLabels),
	})
	if err != nil {
		return fmt.Errorf("list issues: %w", err)
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
		return nil
	default:
		return fmt.Errorf("--state must be one of open, closed, all")
	}
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
	if len(labels) == 0 {
		return ""
	}

	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Name != "" {
			names = append(names, label.Name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
