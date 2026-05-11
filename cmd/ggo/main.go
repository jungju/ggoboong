package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/jungju/ggoboong/internal/envfile"
	"github.com/jungju/ggoboong/internal/runner"
	"github.com/jungju/ggoboong/internal/setup"
)

var version = "dev"

func main() {
	if err := envfile.LoadDefault(); err != nil {
		fmt.Fprintf(os.Stderr, "error: load .env: %v\n", err)
		os.Exit(1)
	}

	if err := newRootCommand().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "ggo",
		Short:         "GitHub App based issue comment bot",
		Version:       currentVersion(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.SetVersionTemplate("ggo {{.Version}}\n")

	var opts runner.Options
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Create a reply comment on a GitHub issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Stdout = cmd.OutOrStdout()
			return runner.Run(cmd.Context(), opts)
		},
	}

	runCmd.Flags().StringVar(&opts.Owner, "owner", "", "GitHub repository owner")
	runCmd.Flags().StringVar(&opts.Repo, "repo", "", "GitHub repository name")
	runCmd.Flags().IntVar(&opts.Issue, "issue", 0, "GitHub issue number")
	runCmd.Flags().StringVar(&opts.ConfigPath, "config", "", "Path to YAML config (default: ./ggo.yaml, then ~/.ggo/ggo.yaml)")
	runCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Print the comment body without creating it")

	var issuesOpts runner.IssuesOptions
	issuesCmd := &cobra.Command{
		Use:   "issues",
		Short: "Find GitHub issues by labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			issuesOpts.Stdout = cmd.OutOrStdout()
			return runner.Issues(cmd.Context(), issuesOpts)
		},
	}
	issuesCmd.Flags().StringVar(&issuesOpts.Owner, "owner", "", "GitHub repository owner")
	issuesCmd.Flags().StringVar(&issuesOpts.Repo, "repo", "", "GitHub repository name")
	issuesCmd.Flags().StringVar(&issuesOpts.State, "state", "open", "Issue state: open, closed, all")
	issuesCmd.Flags().StringArrayVar(&issuesOpts.Labels, "label", nil, "Only include issues with this label; repeat or comma-separate")
	issuesCmd.Flags().StringArrayVar(&issuesOpts.Labels, "tag", nil, "Alias for --label")
	issuesCmd.Flags().StringArrayVar(&issuesOpts.WithoutLabels, "without-label", nil, "Exclude issues with this label; repeat or comma-separate")
	issuesCmd.Flags().StringArrayVar(&issuesOpts.WithoutLabels, "without-tag", nil, "Alias for --without-label")
	issuesCmd.Flags().IntVar(&issuesOpts.Limit, "limit", 0, "Maximum number of issues to print; 0 means all")
	issuesCmd.Flags().BoolVar(&issuesOpts.JSON, "json", false, "Print issues as JSON")
	issuesCmd.Flags().BoolVar(&issuesOpts.IncludeComments, "include-comments", false, "Include issue comments in JSON output")
	issuesCmd.Flags().BoolVar(&issuesOpts.IncludeComments, "comments", false, "Alias for --include-comments")
	issuesCmd.Flags().StringVar(&issuesOpts.LastCommenterNot, "last-commenter-not", "", "Only include issues whose last comment is not by this GitHub user")
	issuesCmd.Flags().StringVar(&issuesOpts.LastCommentContains, "last-comment-contains", "", "Only include issues whose last comment contains this text")
	issuesCmd.Flags().StringVar(&issuesOpts.LastCommentMatches, "last-comment-matches", "", "Only include issues whose last comment matches this regular expression")
	issuesCmd.Flags().StringVar(&issuesOpts.UpdatedAfter, "updated-after", "", "Only include issues updated after RFC3339 time or YYYY-MM-DD")
	issuesCmd.Flags().StringVar(&issuesOpts.UpdatedAfter, "since", "", "Alias for --updated-after")
	issuesCmd.Flags().StringVar(&issuesOpts.ConfigPath, "config", "", "Path to YAML config (default: ./ggo.yaml, then ~/.ggo/ggo.yaml)")

	var commentOpts runner.CommentOptions
	commentCmd := &cobra.Command{
		Use:   "comment",
		Short: "Create a custom comment on a GitHub issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			commentOpts.Stdout = cmd.OutOrStdout()
			return runner.Comment(cmd.Context(), commentOpts)
		},
	}
	commentCmd.Flags().StringVar(&commentOpts.Owner, "owner", "", "GitHub repository owner")
	commentCmd.Flags().StringVar(&commentOpts.Repo, "repo", "", "GitHub repository name")
	commentCmd.Flags().IntVar(&commentOpts.Issue, "issue", 0, "GitHub issue number")
	commentCmd.Flags().StringVar(&commentOpts.BodyFile, "body-file", "", "Path to a Markdown file containing the comment body")
	commentCmd.Flags().StringVar(&commentOpts.SkipIfLastCommenter, "skip-if-last-commenter", "", "Skip when the last comment is by this GitHub user")
	commentCmd.Flags().StringVar(&commentOpts.ConfigPath, "config", "", "Path to YAML config (default: ./ggo.yaml, then ~/.ggo/ggo.yaml)")
	commentCmd.Flags().BoolVar(&commentOpts.DryRun, "dry-run", false, "Print the comment body without creating it")

	var setupOpts setup.Options
	loginCmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"init", "configure"},
		Short:   "Install GitHub App credentials under ~/.ggo",
		RunE: func(cmd *cobra.Command, args []string) error {
			setupOpts.Stdout = cmd.OutOrStdout()
			return setup.Login(setupOpts)
		},
	}
	loginCmd.Flags().Int64Var(&setupOpts.InstallationID, "installation-id", 0, "GitHub App installation ID")
	loginCmd.Flags().StringVar(&setupOpts.PrivateKeyPath, "private-key", "", "Path to GitHub App private key PEM")
	loginCmd.Flags().BoolVar(&setupOpts.DryRun, "dry-run-default", false, "Set bot.dry_run in ~/.ggo/ggo.yaml")
	loginCmd.Flags().BoolVar(&setupOpts.Force, "force", false, "Overwrite existing ~/.ggo files")

	rootCmd.AddCommand(runCmd, issuesCmd, commentCmd, loginCmd)
	return rootCmd
}

func currentVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return version
}
