package agithub

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/clover0/issue-agent/config"
	"github.com/clover0/issue-agent/logger"
)

func CloneRepository(lo logger.Logger, conf config.Config) error {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		lo.Error("GITHUB_TOKEN is not set")
		return fmt.Errorf("GITHUB_TOKEN is not set\n")
	}
	lo.Info("cloning repository...\n")
	cmd := exec.Command("git", "clone", "--depth", "1",
		fmt.Sprintf("https://oauth2:%s@github.com/%s/%s.git", token, conf.Agent.GitHub.Owner, conf.Agent.GitHub.Repository),
		conf.WorkDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return fmt.Errorf("failed to clone repository: %w\n", err)
	}

	lo.Info("cloned repository successfully\n")
	return nil
}
