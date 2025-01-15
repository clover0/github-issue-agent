# Setup

Copy [default_config.yml](https://github.com/clover0/issue-agent/blob/main/agent/config/default_config.yml) to your repository root as `issue_agent.yml`
and edit the file.


Or you can use the following minimum YAML to create a config file.


Configuration parameter example as follows. See [default_config.yml](https://github.com/clover0/issue-agent/blob/main/agent/config/default_config.yml) for more details.

```yaml
# Minimum Example
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
    repository: "example-repository"
```

Set up the environment variables.


```shell

GITHUB_TOKEN=your github token

# If you use OpenAI models
OPENAI_API_KEY=your OpenAI API Key

# If you use Anthropic models
ANTHROPIC_API_KEY=your Anthropic API Key
```
