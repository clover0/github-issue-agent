package agithub

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"

	"github.com/clover0/issue-agent/functions"
	"github.com/clover0/issue-agent/functions/agit"
	"github.com/clover0/issue-agent/logger"
)

// TODO: move to GitHub service
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

// TODO: move to GitHub service
func (s SubmitFileGitHubService) Caller(
	ctx context.Context,
	callerInput functions.SubmitFilesServiceInput,
) functions.SubmitFilesCallerType {
	errorf := func(format string, a ...any) error {
		return fmt.Errorf("submit file service: "+format, a...)
	}

	return func(input functions.SubmitFilesInput) (submitFileOut functions.SubmitFilesOutput, _ error) {
		var out string
		var err error

		// TODO: validation before this caller
		if callerInput.GitEmail == "" {
			return submitFileOut, errorf("git email is not set")
		}
		if callerInput.GitName == "" {
			return submitFileOut, errorf("git  name is not set")
		}

		output, err := agit.GitConfigLocal(s.logger, "user.email", callerInput.GitEmail)
		if err != nil {
			s.logger.Error(output)
			return submitFileOut, errorf("git config email error: %w", err)
		}

		output, err = agit.GitConfigLocal(s.logger, "user.name", callerInput.GitName)
		if err != nil {
			s.logger.Error(output)
			return submitFileOut, errorf("git config email error: %w", err)
		}

		out, err = agit.GitStatus(s.logger)
		if err != nil {
			return submitFileOut, errorf("git status error: %w", err)
		}

		newBranch := agit.MakeBranchName()

		out, err = agit.GitSwitchCreate(s.logger, newBranch)
		if err != nil {
			return submitFileOut, errorf("git switch error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git swicth create: %s\n", out))

		out, err = agit.GitAddAll(s.logger)
		if err != nil {
			return submitFileOut, errorf("git add error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git add all: %s\n", out))

		out, err = agit.GitCommit(s.logger, input.CommitMessageShort, input.CommitMessageDetail)
		if err != nil {
			return submitFileOut, errorf("git commit error: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("git commit: %s\n", out))

		out, err = agit.GitPushBranch(s.logger, newBranch)
		if err != nil {
			s.logger.Error(out)
			return submitFileOut, errorf("git push branch error: %w", err)
		}

		s.logger.Debug(fmt.Sprintf("created PR parameter: name=%s, email=%s, base-branch=%s\n",
			callerInput.GitName, callerInput.GitEmail, callerInput.BaseBranch))
		pr, _, err := s.client.PullRequests.Create(ctx, s.owner, s.repository, &github.NewPullRequest{
			Title: &input.CommitMessageShort,
			Head:  &newBranch,
			Base:  &callerInput.BaseBranch,
			Body:  &input.PullRequestContent,
		})
		if err != nil {
			return submitFileOut, errorf("failed to create PR: %w", err)
		}
		s.logger.Debug(fmt.Sprintf("created PR: %v\n", pr.URL))

		return functions.SubmitFilesOutput{
			Branch:            newBranch,
			PullRequestNumber: *pr.Number,
		}, nil
	}
}
