package agit

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/clover0/issue-agent/logger"
)

const branchPrefix = "agent-"

func MakeBranchName() string {
	return fmt.Sprintf("%s%d", branchPrefix, time.Now().UnixNano())
}

func GitConfigLocal(lo logger.Logger, key, value string) (string, error) {
	cmd := exec.Command("git", "config", "--local", key, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}

func GitStatus(lo logger.Logger) (string, error) {
	cmd := exec.Command("git", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}

func GitSwitchCreate(lo logger.Logger, branch string) (string, error) {
	cmd := exec.Command("git", "switch", "-c", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}

func GitAddAll(lo logger.Logger) (string, error) {
	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}

func GitCommit(lo logger.Logger, commit string, detail string) (string, error) {
	cmd := exec.Command("git", "commit", "-m", commit+"\n\n"+detail)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}

func GitPushBranch(lo logger.Logger, branch string) (string, error) {
	cmd := exec.Command("git", "push", "--set-upstream", "origin", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lo.Error(string(output))
		return "", err
	}
	lo.Info(string(output))

	return string(output), err
}
