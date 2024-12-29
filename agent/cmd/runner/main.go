package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"

	"github/clover0/github-issue-agent/agent"
	"github/clover0/github-issue-agent/config"
	"github/clover0/github-issue-agent/config/cli"
	"github/clover0/github-issue-agent/functions"
	"github/clover0/github-issue-agent/functions/agithub"
	"github/clover0/github-issue-agent/loader"
	"github/clover0/github-issue-agent/logger"
	"github/clover0/github-issue-agent/models"
	libprompt "github/clover0/github-issue-agent/prompt"
	"github/clover0/github-issue-agent/store"
	"github/clover0/github-issue-agent/util"
)

func newGitHub() *github.Client {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		panic("GITHUB_TOKEN is not set")
	}
	return github.NewClient(nil).WithAuthToken(token)
}

func main() {
	// TODO:
	//lo := logger.NewDefaultLogger()
	lo := logger.NewPrinter()

	cliIn, err := cli.ParseIssueInput()
	if err != nil {
		lo.Error("failed to parse input: %s", err)
		os.Exit(1)
	}

	conf, err := config.Load(cliIn.Common.Config)
	if err != nil {
		lo.Error("failed to load config: %s", err)
		os.Exit(1)
	}

	if conf.Agent.GitHub.CloneRepository {
		if err := agithub.CloneRepository(lo, conf); err != nil {
			lo.Error("failed to clone repository")
			os.Exit(1)
		}
	}

	// TODO: no dependency with changing directory
	if err := os.Chdir(conf.WorkDir); err != nil {
		lo.Error("failed to change directory: %s", err)
		os.Exit(1)
	}

	llmForwarder := models.SelectForwarder(lo, conf.Agent.Model)

	promptTemplate, err := libprompt.LoadPromptTemplateFromYAML(conf.Agent.PromptTemplate)
	if err != nil {
		lo.Error("failed to load prompt template: %s", err)
		os.Exit(1)
	}

	ctx := context.Background()

	gh := newGitHub()

	functions.InitializeFunctions(
		conf.Agent.GitHub.NoSubmit,
		agithub.NewGitHubService(conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository, gh, lo),
		conf.AllowFunctions,
	)
	lo.Info("allowed functions: %s", conf.AllowFunctions)

	var issLoader loader.Loader
	var issue loader.Issue
	if len(cliIn.FromFile) > 0 {
		lo.Info("load issue from file")
		issLoader = loader.NewFileLoader()
		if issue, err = issLoader.LoadIssue(ctx, cliIn.FromFile); err != nil {
			lo.Error("failed to load issue from file: %s", err)
			os.Exit(1)
		}
	} else {
		lo.Info("load issue from GitHub")
		issLoader = loader.NewGitHubLoader(gh, conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository)
		if issue, err = issLoader.LoadIssue(ctx, cliIn.GithubIssueNumber); err != nil {
			lo.Error("failed to load issue from GitHub: %s", err)
			os.Exit(1)
		}
	}

	submitServiceCaller := agithub.NewSubmitFileGitHubService(conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository, gh, lo).
		Caller(ctx, functions.SubmitFilesServiceInput{
			BaseBranch: cliIn.BaseBranch,
			GitEmail:   conf.Agent.Git.UserEmail,
			GitName:    conf.Agent.Git.UserName,
		})

	dataStore := store.NewStore()

	parameter := agent.Parameter{
		MaxSteps: conf.Agent.MaxSteps,
		Model:    conf.Agent.Model,
	}

	prompt, err := libprompt.BuildRequirementPrompt(promptTemplate, issue)
	if err != nil {
		lo.Error("failed build requirement prompt: %s", err)
		os.Exit(1)
	}
	requirementAgent := RunRequirementAgent(prompt, submitServiceCaller, parameter, lo, &dataStore, llmForwarder)

	instruction := requirementAgent.History()[len(requirementAgent.History())-1].RawContent
	prompt, err = libprompt.BuildDeveloperPrompt(promptTemplate, issLoader, issue.Path, instruction)
	if err != nil {
		lo.Error("failed build developer prompt: %s", err)
		os.Exit(1)
	}
	developer2Agent := RunDeveloperAgent(prompt, submitServiceCaller, parameter, lo, &dataStore, llmForwarder)

	if s := dataStore.GetSubmission(store.LastSubmissionKey); s == nil {
		lo.Error("submission is not found")
		os.Exit(1)
	}
	submittedPRNumber := dataStore.GetSubmission(store.LastSubmissionKey).PullRequestNumber

	prompt, err = libprompt.BuildReviewManagerPrompt(promptTemplate, issue, util.Map(developer2Agent.ChangedFiles(), func(f store.File) string { return f.Path }))
	if err != nil {
		lo.Error("failed to build review manager prompt: %s", err)
		os.Exit(1)
	}
	reviewManager := ReviewManagerAgent(
		prompt,
		parameter,
		developer2Agent.ChangedFiles(),
		submitServiceCaller,
		lo, &dataStore, llmForwarder)
	output := reviewManager.History()[len(reviewManager.History())-1].RawContent
	lo.Info("ReviewManagerAgent output: %s", output)
	type agentPrompt struct {
		AgentName string `json:"agent_name"`
		Prompt    string `json:"prompt"`
	}

	// TODO: refactor
	// parse json output for revwier agents
	// expected output:
	//   text text text...
	//   [{"agent_name": "agent1", "prompt": "prompt1"}, ...]
	//   test...
	var prompts []agentPrompt
	jsonStart := strings.Index(output, "[")   // find JSON start
	jsonEnd := strings.LastIndex(output, "]") // find JSON end
	outBuff := bytes.NewBufferString(output[jsonStart : jsonEnd+1])
	if err := json.Unmarshal(outBuff.Bytes(), &prompts); err != nil {
		lo.Error("failed to unmarshal output: %s", err)
		os.Exit(1)
	}

	for _, p := range prompts {
		lo.Info("Run %s\n", p.AgentName)
		prpt, err := libprompt.BuildReviewerPrompt(promptTemplate, issue, submittedPRNumber, p.Prompt)
		if err != nil {
			lo.Error("failed to build reviewer prompt: %s", err)
			os.Exit(1)
		}

		reviewer := RunReviewAgent(
			p.AgentName,
			prpt,
			parameter, submitServiceCaller, lo, &dataStore, llmForwarder)
		output := reviewer.History()[len(reviewer.History())-1].RawContent

		// parse JSON output
		// TODO: validate
		var reviews []struct {
			ReviewFilePath  string `json:"review_file_path"`
			ReviewStartLine int    `json:"review_start_line"`
			ReviewEndLine   int    `json:"review_end_line"`
			ReviewComment   string `json:"review_comment"`
			Suggestion      string `json:"suggestion"`
		}
		jsonStart := strings.Index(output, "[")   // find JSON start
		jsonEnd := strings.LastIndex(output, "]") // find JSON end
		outBuff := bytes.NewBufferString(output[jsonStart : jsonEnd+1])
		if err := json.Unmarshal(outBuff.Bytes(), &reviews); err != nil {
			lo.Error("failed to unmarshal output: %s", err)
			os.Exit(1)
		}

		// TODO: move to agithub package
		var comments []*github.DraftReviewComment
		for _, r := range reviews {
			startLine := github.Int(r.ReviewStartLine)
			if *startLine == 0 {
				*startLine = 1
			}
			if r.ReviewStartLine == r.ReviewEndLine {
				startLine = nil
			}
			body := fmt.Sprintf("from %s\n", p.AgentName) +
				r.ReviewComment
			if r.Suggestion != "" {
				body += "\n\n```suggestion\n" + r.Suggestion + "\n```\n"
			}
			comments = append(comments, &github.DraftReviewComment{
				Path:      github.String(r.ReviewFilePath),
				Body:      github.String(body),
				StartLine: startLine,
				Line:      github.Int(r.ReviewEndLine),
				Side:      github.String("RIGHT"),
			})
		}

		if _, _, err := gh.PullRequests.CreateReview(context.Background(),
			conf.Agent.GitHub.Owner,
			conf.Agent.GitHub.Repository,
			submittedPRNumber,
			&github.PullRequestReviewRequest{
				Event:    github.String("COMMENT"),
				Comments: comments,
			},
		); err != nil {
			lo.Error("failed to create pull request review: %s. but agent continue to work\n", err)
		}
		lo.Info("Finish %s\n", p.AgentName)
	}

	lo.Info("Agents finished successfully!")
}

func RunRequirementAgent(
	prompt libprompt.Prompt,
	submitServiceCaller functions.SubmitFilesCallerType,
	parameter agent.Parameter,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder agent.LLMForwarder,
) agent.Agent {
	ag := agent.NewAgent(
		parameter,
		"requirementAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("requirement agent failed: %s", err)
		os.Exit(1)
	}

	return ag
}

func RunDeveloperAgent(
	prompt libprompt.Prompt,
	submitServiceCaller functions.SubmitFilesCallerType,
	parameter agent.Parameter,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder agent.LLMForwarder,
) agent.Agent {
	ag := agent.NewAgent(
		parameter,
		"developerAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("ag failed: %s", err)
		os.Exit(1)
	}

	return ag
}

func ReviewManagerAgent(
	prompt libprompt.Prompt,
	parameter agent.Parameter,
	changedFiles []store.File,
	submitServiceCaller functions.SubmitFilesCallerType,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder agent.LLMForwarder,
) agent.Agent {
	var changedFilePath []string
	for _, f := range changedFiles {
		changedFilePath = append(changedFilePath, f.Path)
	}
	ag := agent.NewAgent(
		parameter,
		"reviewManagerAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("reviewManagerAgent failed: %s", err)
		os.Exit(1)
	}

	return ag
}

func RunReviewAgent(
	name string,
	prompt libprompt.Prompt,
	parameter agent.Parameter,
	submitServiceCaller functions.SubmitFilesCallerType, // TODO
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder agent.LLMForwarder,
) agent.Agent {
	ag := agent.NewAgent(
		parameter,
		name,
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("%s failed: %s", name, err)
		os.Exit(1)
	}

	return ag
}
