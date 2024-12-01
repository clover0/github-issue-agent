package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/go-github/v66/github"

	"github/clover0/github-issue-agent/agent"
	"github/clover0/github-issue-agent/config/cli"
	"github/clover0/github-issue-agent/functions"
	"github/clover0/github-issue-agent/functions/agithub"
	"github/clover0/github-issue-agent/loader"
	"github/clover0/github-issue-agent/logger"
	"github/clover0/github-issue-agent/models"
	libprompt "github/clover0/github-issue-agent/prompt"
	"github/clover0/github-issue-agent/store"
)

func newGitHub() *github.Client {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		panic("GITHUB_TOKEN is not set")
	}
	return github.NewClient(nil).WithAuthToken(token)
}

func main() {
	//lo := logger.NewDefaultLogger()
	lo := logger.NewPrinter()

	cliIn, err := cli.ParseInput()
	if err != nil {
		lo.Error("failed to parse input: %s", err)
		os.Exit(1)
	}

	if cliIn.CloneRepository {
		token, ok := os.LookupEnv("GITHUB_TOKEN")
		if !ok {
			lo.Error("GITHUB_TOKEN is not set")
			os.Exit(1)
		}
		cmd := exec.Command("git", "clone", "--depth", "1",
			fmt.Sprintf("https://oauth2:%s@github.com/%s/%s.git", token, cliIn.RepositoryOwner, cliIn.Repository),
			cliIn.AgentWorkDir,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			lo.Error(string(output))
			lo.Error("failed to clone repository: %s", err)
			os.Exit(1)
		}
		lo.Info("cloned repository successfully")
	}

	// TODO: no dependency with changing directory
	if err := os.Chdir(cliIn.AgentWorkDir); err != nil {
		lo.Error("failed to change directory: %s", err)
		os.Exit(1)
	}

	promptTemplate, err := libprompt.LoadPromptTemplateFromYAML(cliIn.Template)
	if err != nil {
		lo.Error("failed to load prompt template: %s", err)
		os.Exit(1)
	}

	gh := newGitHub()

	issLoader := loader.NewGitHubLoader(gh, cliIn.RepositoryOwner, cliIn.Repository)

	dataStore := store.NewStore()

	startAgent := RunDeveloperAgent(promptTemplate, issLoader, cliIn, lo, gh, &dataStore)

	RunSecurityAgent(promptTemplate, startAgent.ChangedFiles(), cliIn, lo, gh, &dataStore)

	lo.Info("Agents finished successfully!")
}

func RunDeveloperAgent(
	promptTemplate libprompt.PromptTemplate,
	issLoader loader.Loader,
	cliIn cli.Inputs,
	lo logger.Logger,
	gh *github.Client,
	dataStore *store.Store,
) agent.Agent {
	ctx := context.Background()

	prompt, err := libprompt.BuildDeveloperPrompt(promptTemplate, issLoader, cliIn.GithubIssueNumber)
	if err != nil {
		lo.Error("failed buld prompt: %s", err)
		os.Exit(1)
	}

	ag := agent.NewAgent(
		agent.Parameter{
			MaxSteps: cliIn.MaxSteps,
			Model:    cliIn.Model,
		},
		"main",
		lo,
		agithub.NewSubmitFileGitHubService(cliIn.RepositoryOwner, cliIn.Repository, gh, lo).
			Caller(ctx, functions.SubmitFilesServiceInput{
				BaseBranch: cliIn.BaseBranch,
				GitEmail:   cliIn.GitEmail,
				GitName:    cliIn.GitName,
			}),
		prompt,
		models.NewOpenAILLMForwarder(lo),
		dataStore,
	)

	_, err = ag.Work()
	if err != nil {
		lo.Error("ag failed: %s", err)
		os.Exit(1)
	}

	return ag
}

func RunSecurityAgent(
	promptTemplate libprompt.PromptTemplate,
	changedFiles []store.File,
	cliIn cli.Inputs,
	lo logger.Logger,
	gh *github.Client,
	dataStore *store.Store,
) agent.Agent {
	ctx := context.Background()
	var changedFilePath []string
	for _, f := range changedFiles {
		changedFilePath = append(changedFilePath, f.Path)
	}

	securityPrompt, err := libprompt.BuildSecurityPrompt(promptTemplate, changedFilePath)
	if err != nil {
		lo.Error("failed to build security prompt: %s", err)
		os.Exit(1)
	}
	ag := agent.NewAgent(
		agent.Parameter{
			MaxSteps: cliIn.MaxSteps,
			Model:    cliIn.Model,
		},
		"securityAgent",
		lo,
		agithub.NewSubmitFileGitHubService(cliIn.RepositoryOwner, cliIn.Repository, gh, lo).
			Caller(ctx, functions.SubmitFilesServiceInput{
				BaseBranch: cliIn.BaseBranch,
				GitEmail:   cliIn.GitEmail,
				GitName:    cliIn.GitName,
			}),
		securityPrompt,
		models.NewOpenAILLMForwarder(lo),
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("securityAgent failed: %s", err)
		os.Exit(1)
	}

	return ag
}
