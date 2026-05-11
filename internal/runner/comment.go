package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jungju/ggoboong/internal/config"
	"github.com/jungju/ggoboong/internal/githubissue"
)

type CommentOptions struct {
	Owner               string
	Repo                string
	Issue               int
	BodyFile            string
	SkipIfLastCommenter string
	ConfigPath          string
	DryRun              bool
	Stdout              io.Writer
	HTTPClient          *http.Client
}

func Comment(ctx context.Context, opts CommentOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if err := validateCommentOptions(opts); err != nil {
		return err
	}

	body, err := readCommentBody(opts.BodyFile)
	if err != nil {
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

	issueClient, err := newIssueClient(ctx, cfg, opts.HTTPClient)
	if err != nil {
		return err
	}

	if opts.SkipIfLastCommenter != "" {
		comments, err := issueClient.ListComments(ctx, opts.Owner, opts.Repo, opts.Issue)
		if err != nil {
			return fmt.Errorf("list comments: %w", err)
		}
		if len(comments) > 0 {
			lastComment := comments[len(comments)-1]
			if githubissue.LoginMatches(lastComment.User.Login, opts.SkipIfLastCommenter) {
				fmt.Fprintf(opts.Stdout, "last comment is already by %s on issue #%d; skipping\n", lastComment.User.Login, opts.Issue)
				return nil
			}
		}
	}

	if dryRun {
		fmt.Fprintln(opts.Stdout, "dry-run: comment would be created with body:")
		fmt.Fprint(opts.Stdout, body)
		return nil
	}

	comment, err := issueClient.CreateComment(ctx, opts.Owner, opts.Repo, opts.Issue, body)
	if err != nil {
		return fmt.Errorf("create issue comment: %w", err)
	}

	fmt.Fprintf(opts.Stdout, "created issue comment: %s\n", comment.URL)
	return nil
}

func validateCommentOptions(opts CommentOptions) error {
	if opts.Owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if opts.Repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if opts.Issue <= 0 {
		return fmt.Errorf("--issue must be set to a positive integer")
	}
	if strings.TrimSpace(opts.BodyFile) == "" {
		return fmt.Errorf("--body-file is required")
	}
	return nil
}

func readCommentBody(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --body-file: %w", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return "", fmt.Errorf("--body-file must not be empty")
	}
	return string(body), nil
}
