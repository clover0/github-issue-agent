# Usage

## Run
You decide what GitHub issue you want to resolve

Example:

- GitHub Working Repository `clover0/example-repository`
- GitHub Issue Number 123 to solve
- Base Branch `main` to create a pull request

```shell
$ issue-agent issue --github_issue_number 123 --base_branch main 
```

Repository configuration is in `issue_agent.yml` file.


With environment variables in one line. [`gh` CLI is useful](https://github.com/cli/cli#installation).
```shell
$ GITHUB_TOKEN=$(gh auth token) \
  ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} \
  issue-agent issue --github_issue_number 123 --base_branch main
```

OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable is required


## Branch

Working branch is created automatically. (`agent-` prefix)
