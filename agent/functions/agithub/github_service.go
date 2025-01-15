package agithub

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v66/github"

	"github.com/clover0/issue-agent/logger"
)

type GitHubService struct {
	owner      string
	repository string
	client     *github.Client
	logger     logger.Logger
}

func NewGitHubService(
	owner string,
	repository string,
	client *github.Client,
	logger logger.Logger,
) GitHubService {
	return GitHubService{
		owner:      owner,
		repository: repository,
		client:     client,
		logger:     logger,
	}
}

func (s GitHubService) GetPullRequestDiff(prNumber string) (string, error) {
	number, err := strconv.Atoi(prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to convert pull request number to int: %w", err)
	}

	c := context.Background()
	diff, _, err := s.client.PullRequests.GetRaw(c, s.owner, s.repository, number, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", fmt.Errorf("failed to get pull request diff: %w", err)
	}

	return diff, nil
}
