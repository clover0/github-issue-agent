package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v66/github"

	"github.com/clover0/issue-agent/agent"
	"github.com/clover0/issue-agent/config"
	"github.com/clover0/issue-agent/functions/agithub"
	"github.com/clover0/issue-agent/loader"
	"github.com/clover0/issue-agent/logger"
)

func IssueCommand(flags []string) error {
	cliIn, err := ParseIssueInput(flags)
	if err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	conf, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	lo := logger.NewPrinter(conf.LogLevel)

	if *conf.Agent.GitHub.CloneRepository {
		if err := agithub.CloneRepository(lo, conf); err != nil {
			lo.Error("failed to clone repository")
			return err
		}
	}

	// TODO: no dependency with changing directory
	if err := os.Chdir(conf.WorkDir); err != nil {
		lo.Error("failed to change directory: %s\n", err)
		return err
	}

	gh := newGitHub()

	ctx := context.Background()
	var issLoader loader.Loader
	var issue loader.Issue
	if len(cliIn.FromFile) > 0 {
		lo.Info("load issue from file\n")
		issLoader = loader.NewFileLoader()
		if issue, err = issLoader.LoadIssue(ctx, cliIn.FromFile); err != nil {
			lo.Error("failed to load issue from file: %s\n", err)
			return err
		}
	} else {
		lo.Info("load issue from GitHub\n")
		issLoader = loader.NewGitHubLoader(gh, conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository)
		if issue, err = issLoader.LoadIssue(ctx, cliIn.GithubIssueNumber); err != nil {
			lo.Error("failed to load issue from GitHub: %s\n", err)
			return err
		}
	}

	return agent.OrchestrateAgents(ctx, lo, conf, issLoader, cliIn.BaseBranch, issue, gh)
}

func newGitHub() *github.Client {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		panic("GITHUB_TOKEN is not set")
	}
	return github.NewClient(nil).WithAuthToken(token)
}
