# GitHub Issue Agent
Agent that, given an assignment, produces work products in the git repository.

[Documentation](https://clover0.github.io/issue-agent/)

## Models
The following models are supported.

- OpenAI models
  - gpt-4o
  - gpt-4o-mini
- Anthropic models
  - claude-3-5-sonnet


## Work Product
Only GitHub pull requests are supported.


## Installation
### Homebrew
```shell
brew install clover0/issue-agent/issue-agent
```

### GitHub Releases
Download the binary from [GitHub Releases](https://github.com/clover0/issue-agent/releases)

## Getting Started
### Setup

Copy [default_config.yml](agent/config/default_config.yml) to your repository root as `issue_agent.yml`.

Next, edit the config file as needed.

Configuration parameter example as follows. See [default_config.yml](agent/config/default_config.yml) for more details.

```yaml
# Example
# issue_agent.yml

communication_language: "Japanese"
agent:
  model: "claude-3-5-sonnet-20241022"
  max_steps: 70
  git:
    user_name: "username"
    user_email: "email@example.com"
  github:
    owner: "clover0"
    repository: "github-issue-agent"
```

Set up the environment variables.

```shell
GITHUB_TOKEN=your github token

# If you use OpenAI models
OPENAI_API_KEY=your OpenAI API Key

# If you use Anthropic models
ANTHROPIC_API_KEY=your Anthropic API Key
````


### Run
Human decides what GitHub issue they want to resolve

e.g)
- GitHub Working Repository `clover0/example-repository`
- GitHub Issue Number 123 to solve
- Base Branch `main` to create a pull request

```shell
$ issue-agent issue --github_issue_number 123 \
                    --base_branch main 
````

With environment variables in one line. [`gh` CLI is useful](https://github.com/cli/cli#installation).
```shell
$ GITHUB_TOKEN=$(gh auth token) \
  ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} \
  issue-agent issue --github_issue_number 123 \
                    --base_branch main
```

- Working branch is created automatically. (`agent-` prefix)
- OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable is required
