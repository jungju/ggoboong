package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jungju/ggoboong/internal/envfile"
	"github.com/jungju/ggoboong/internal/runner"
	"github.com/jungju/ggoboong/internal/setup"
)

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
		SilenceErrors: true,
		SilenceUsage:  true,
	}

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
	issuesCmd.Flags().StringVar(&issuesOpts.ConfigPath, "config", "", "Path to YAML config (default: ./ggo.yaml, then ~/.ggo/ggo.yaml)")

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

	rootCmd.AddCommand(runCmd, issuesCmd, loginCmd)
	return rootCmd
}
