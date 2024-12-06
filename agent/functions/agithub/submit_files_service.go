package agithub

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"

	"github/clover0/github-issue-agent/functions"
	"github/clover0/github-issue-agent/functions/agit"
	"github/clover0/github-issue-agent/logger"
)

type SubmitFileGitHubService struct {
	owner      string
	repository string
	client     *github.Client
	logger     logger.Logger
}

func NewSubmitFileGitHubService(
	owner string,
	repository string,
	client *github.Client,
	logger logger.Logger,
) functions.SubmitFilesService {
	return SubmitFileGitHubService{
		owner:      owner,
		repository: repository,
		client:     client,
		logger:     logger,
	}
}

func (s SubmitFileGitHubService) Caller(
	ctx context.Context,
	callerInput functions.SubmitFilesServiceInput,
) functions.SubmitFilesCallerType {
	return func(input functions.SubmitFilesInput) error {
		var out string
		var err error

		// TODO: validation before this caller
		if callerInput.GitEmail == "" {
			return fmt.Errorf("submit file service: git email is not set")
		}
		if callerInput.GitName == "" {
			return fmt.Errorf("submit file service: git  name is not set")
		}

		output, err := agit.GitConfigLocal(s.logger, "user.email", callerInput.GitEmail)
		if err != nil {
			s.logger.Error(output)
			return fmt.Errorf("submit file service: git config email error: %w", err)
		}

		output, err = agit.GitConfigLocal(s.logger, "user.name", callerInput.GitName)
		if err != nil {
			s.logger.Error(output)
			return fmt.Errorf("submit file service: git config email error: %w", err)
		}

		out, err = agit.GitStatus(s.logger)
		if err != nil {
			return fmt.Errorf("submit file service: git stastus error: %w", err)
		}

		newBranch := agit.MakeBranchName()

		out, err = agit.GitSwitchCreate(s.logger, newBranch)
		if err != nil {
			return fmt.Errorf("submit file service: git switch error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git swicth create: %s", out))

		out, err = agit.GitAddAll(s.logger)
		if err != nil {
			return fmt.Errorf("submit file service: git add error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git add all: %s\n", out))

		out, err = agit.GitCommit(s.logger, input.CommitMessage)
		if err != nil {
			return fmt.Errorf("submit file service: git commit error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git commit: %s\n", out))

		out, err = agit.GitPushBranch(s.logger, newBranch)
		if err != nil {
			s.logger.Error(out)
			return fmt.Errorf("submit file service: git push branch error: %w", err)
		}

		s.logger.Debug(fmt.Sprintf("submit file service: create PR parameter: %s", callerInput))
		pr, _, err := s.client.PullRequests.Create(ctx, s.owner, s.repository, &github.NewPullRequest{
			Title: &input.CommitMessage,
			Head:  &newBranch,
			Base:  &callerInput.BaseBranch,
			Body:  &input.PullRequestContent,
		})
		if err != nil {
			return fmt.Errorf("submit file service: create PR: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("created PR: %s", pr.URL))

		return nil
	}
}

func (s SubmitFileGitHubService) Commit() {
}
