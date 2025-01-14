package cli

import (
	"flag"
	"fmt"
	"github.com/go-playground/validator/v10"
)

type IssueInputs struct {
	Common            *CommonInput
	GithubIssueNumber string
	BaseBranch        string `validate:"required"`
	FromFile          string
}

func ParseIssueInput(flags []string) (IssueInputs, error) {
	cliIn := IssueInputs{
		Common: &CommonInput{},
	}

	cmd := flag.NewFlagSet("issue", flag.ExitOnError)

	addCommonFlags(cmd, cliIn.Common)

	cmd.StringVar(&cliIn.GithubIssueNumber, "github_issue_number", "", "GitHubLoader issue number")
	cmd.StringVar(&cliIn.BaseBranch, "base_branch", "", "Base Branch for pull request")
	cmd.StringVar(&cliIn.FromFile, "from_file", "", "Issue content from file path")

	if err := cmd.Parse(flags); err != nil {
		return IssueInputs{}, fmt.Errorf("failed to parse input: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(cliIn); err != nil {
		errs := err.(validator.ValidationErrors)
		return IssueInputs{}, fmt.Errorf("validation failed: %w", errs)
	}

	if cliIn.GithubIssueNumber == "" && cliIn.FromFile == "" {
		return IssueInputs{}, fmt.Errorf("github_issue_number or from_file is required")
	}

	return cliIn, nil
}
