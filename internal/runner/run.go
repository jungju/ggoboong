package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jungju/ggoboong/internal/config"
	"github.com/jungju/ggoboong/internal/githubapp"
	"github.com/jungju/ggoboong/internal/githubissue"
)

type Options struct {
	Owner      string
	Repo       string
	Issue      int
	ConfigPath string
	DryRun     bool
	Stdout     io.Writer
	HTTPClient *http.Client
}

func Run(ctx context.Context, opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}

	if err := validateOptions(opts); err != nil {
		return err
	}

	cfg, configPath, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if configPath != "" {
		fmt.Fprintf(opts.Stdout, "using config: %s\n", configPath)
	}

	dryRun := cfg.Bot.DryRun || opts.DryRun

	appJWT, err := githubapp.GenerateJWT(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("generate GitHub App JWT: %w", err)
	}

	appClient := githubapp.NewClient(opts.HTTPClient)
	token, err := appClient.CreateInstallationToken(ctx, cfg.GitHub.InstallationID, appJWT)
	if err != nil {
		return fmt.Errorf("create installation access token: %w", err)
	}

	issueClient := githubissue.NewClient(opts.HTTPClient, token)
	issue, err := issueClient.GetIssue(ctx, opts.Owner, opts.Repo, opts.Issue)
	if err != nil {
		return fmt.Errorf("get issue: %w", err)
	}

	comments, err := issueClient.ListComments(ctx, opts.Owner, opts.Repo, opts.Issue)
	if err != nil {
		return fmt.Errorf("list comments: %w", err)
	}

	for _, comment := range comments {
		if strings.Contains(comment.Body, cfg.Bot.Marker) {
			fmt.Fprintf(opts.Stdout, "bot comment already exists on issue #%d; skipping\n", opts.Issue)
			return nil
		}
	}

	body := buildCommentBody(cfg.Bot.Marker, issue)
	if dryRun {
		fmt.Fprintln(opts.Stdout, "dry-run: comment would be created with body:")
		fmt.Fprintln(opts.Stdout, body)
		return nil
	}

	comment, err := issueClient.CreateComment(ctx, opts.Owner, opts.Repo, opts.Issue, body)
	if err != nil {
		return fmt.Errorf("create issue comment: %w", err)
	}

	fmt.Fprintf(opts.Stdout, "created issue comment: %s\n", comment.URL)
	return nil
}

func validateOptions(opts Options) error {
	if opts.Owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if opts.Repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if opts.Issue <= 0 {
		return fmt.Errorf("--issue must be set to a positive integer")
	}
	return nil
}

func buildCommentBody(marker string, issue *githubissue.Issue) string {
	return fmt.Sprintf("%s\n안녕하세요! ggo가 이 이슈를 확인했습니다.\n\n- Issue: #%d %s\n- State: %s\n",
		marker,
		issue.Number,
		issue.Title,
		issue.State,
	)
}
