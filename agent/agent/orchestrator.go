package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/clover0/issue-agent/config"
	"github.com/clover0/issue-agent/functions"
	"github.com/clover0/issue-agent/functions/agithub"
	"github.com/clover0/issue-agent/loader"
	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/models"
	libprompt "github.com/clover0/issue-agent/prompt"
	"github.com/clover0/issue-agent/store"
	"github.com/clover0/issue-agent/util"
)

// OrchestrateAgents orchestrates agents
// Currently, the processing is based on the issue command
// TODO: refactor many arguments
// TODO: no dependent on issue command
func OrchestrateAgents(
	ctx context.Context,
	lo logger.Logger,
	conf config.Config,
	loaderr loader.Loader,
	baseBranch string,
	issue loader.Issue,
	gh *github.Client,
) error {
	llmForwarder := models.SelectForwarder(lo, conf.Agent.Model)

	promptPath := conf.Agent.PromptPath
	if len(promptPath) > 0 {
		// In container, the prompt file is mounted to `config.PromptFilePath`
		promptPath = config.PromptFilePath
	}
	promptTemplate, err := libprompt.LoadPrompt(promptPath)
	if err != nil {
		lo.Error("failed to load prompt template: %s\n", err)
		return err
	}

	functions.InitializeFunctions(
		*conf.Agent.GitHub.NoSubmit,
		agithub.NewGitHubService(conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository, gh, lo),
		conf.Agent.AllowFunctions,
	)
	lo.Info("allowed functions: %s\n", strings.Join(util.Map(
		functions.AllFunctions(),
		func(e functions.Function) string { return e.Name.String() },
	), ","))

	submitServiceCaller := agithub.NewSubmitFileGitHubService(conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository, gh, lo).
		Caller(ctx, functions.SubmitFilesServiceInput{
			BaseBranch: baseBranch,
			GitEmail:   conf.Agent.Git.UserEmail,
			GitName:    conf.Agent.Git.UserName,
		})

	dataStore := store.NewStore()

	parameter := Parameter{
		MaxSteps: conf.Agent.MaxSteps,
		Model:    conf.Agent.Model,
	}

	prompt, err := libprompt.BuildRequirementPrompt(promptTemplate, conf.CommunicationLanguage, issue)
	if err != nil {
		lo.Error("failed build requirement prompt: %s\n", err)
		return err
	}
	requirementAgent, err := RunRequirementAgent(prompt, submitServiceCaller, parameter, lo, &dataStore, llmForwarder)
	if err != nil {
		lo.Error("requirement agent failed: %s\n", err)
		return err
	}

	instruction := requirementAgent.History()[len(requirementAgent.History())-1].RawContent
	prompt, err = libprompt.BuildDeveloperPrompt(promptTemplate, conf.CommunicationLanguage, loaderr, issue.Path, instruction)
	if err != nil {
		lo.Error("failed build developer prompt: %s\n", err)
		return err
	}
	developerAgent, err := RunDeveloperAgent(prompt, submitServiceCaller, parameter, lo, &dataStore, llmForwarder)
	if err != nil {
		lo.Error("developer agent failed: %s\n", err)
		return err
	}

	if *conf.Agent.SkipReviewAgents {
		lo.Info("Skip review agents\n")
		lo.Info("Agents finished successfully!\n")
		return nil
	}

	if s := dataStore.GetSubmission(store.LastSubmissionKey); s == nil {
		lo.Error("submission is not found\n")
		return err
	}
	submittedPRNumber := dataStore.GetSubmission(store.LastSubmissionKey).PullRequestNumber

	prompt, err = libprompt.BuildReviewManagerPrompt(promptTemplate, conf.CommunicationLanguage, issue, util.Map(developerAgent.ChangedFiles(), func(f store.File) string { return f.Path }))
	if err != nil {
		lo.Error("failed to build review manager prompt: %s\n", err)
		return err
	}
	reviewManager, err := ReviewManagerAgent(
		prompt,
		parameter,
		developerAgent.ChangedFiles(),
		submitServiceCaller,
		lo, &dataStore, llmForwarder)
	if err != nil {
		lo.Error("reviewManagerAgent failed: %s\n", err)
		return err
	}
	output := reviewManager.History()[len(reviewManager.History())-1].RawContent
	lo.Info("ReviewManagerAgent output: %s\n", output)
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
		lo.Error("failed to unmarshal output: %s\n", err)
		return err
	}

	for _, p := range prompts {
		lo.Info("Run %s\n", p.AgentName)
		prpt, err := libprompt.BuildReviewerPrompt(promptTemplate, conf.CommunicationLanguage, submittedPRNumber, p.Prompt)
		if err != nil {
			lo.Error("failed to build reviewer prompt: %s\n", err)
			return err
		}

		reviewer, err := RunReviewAgent(
			p.AgentName,
			prpt,
			parameter, submitServiceCaller, lo, &dataStore, llmForwarder)
		if err != nil {
			lo.Error("%s failed: %s\n", p.AgentName, err)
			return err
		}
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
			lo.Error("failed to unmarshal output: %s\n", err)
			return err
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
				// TODO: escape JSON in Suggestion string
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

	lo.Info("Agents finished successfully!\n")

	return nil
}

func RunRequirementAgent(
	prompt libprompt.Prompt,
	submitServiceCaller functions.SubmitFilesCallerType,
	parameter Parameter,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder models.LLMForwarder,
) (Agent, error) {
	ag := NewAgent(
		parameter,
		"requirementAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("requirement agent failed: %s\n", err)
		return Agent{}, err
	}

	return ag, nil
}

func RunDeveloperAgent(
	prompt libprompt.Prompt,
	submitServiceCaller functions.SubmitFilesCallerType,
	parameter Parameter,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder models.LLMForwarder,
) (Agent, error) {
	ag := NewAgent(
		parameter,
		"developerAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("agent failed: %s\n", err)
		return Agent{}, err
	}

	return ag, nil
}

func ReviewManagerAgent(
	prompt libprompt.Prompt,
	parameter Parameter,
	changedFiles []store.File,
	submitServiceCaller functions.SubmitFilesCallerType,
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder models.LLMForwarder,
) (Agent, error) {
	var changedFilePath []string
	for _, f := range changedFiles {
		changedFilePath = append(changedFilePath, f.Path)
	}
	ag := NewAgent(
		parameter,
		"reviewManagerAgent",
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("reviewManagerAgent failed: %s\n", err)
		return Agent{}, err
	}

	return ag, nil
}

func RunReviewAgent(
	name string,
	prompt libprompt.Prompt,
	parameter Parameter,
	submitServiceCaller functions.SubmitFilesCallerType, // TODO
	lo logger.Logger,
	dataStore *store.Store,
	llmForwarder models.LLMForwarder,
) (Agent, error) {
	ag := NewAgent(
		parameter,
		name,
		lo,
		submitServiceCaller,
		prompt,
		llmForwarder,
		dataStore,
	)

	if _, err := ag.Work(); err != nil {
		lo.Error("%s failed: %s\n", name, err)
		return Agent{}, err
	}

	return ag, nil
}
