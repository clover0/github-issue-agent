package config

import (
	_ "embed"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed default_config.yaml
var defaultConfig []byte

func DefaultConfig() []byte {
	return defaultConfig
}

type Config struct {
	WorkDir string `yaml:"workdir"`
	Agent   struct {
		PromptTemplate string `yaml:"prompt_template"`
		Model          string `yaml:"model"`
		MaxSteps       int    `yaml:"max_steps"`
		Git            struct {
			UserName  string `yaml:"user_name"`
			UserEmail string `yaml:"user_email"`
		} `yaml:"git"`
		GitHub struct {
			NoSubmit        bool   `yaml:"no_submit"`
			CloneRepository bool   `yaml:"clone_repository"`
			Owner           string `yaml:"owner"`
			Repository      string `yaml:"repository"`
			BaseBranch      string `yaml:"base_branch"`
		}
	} `yaml:"agent"`
	AllowFunctions []string `yaml:"allow_functions"`
}

func Load(path string) (Config, error) {
	var cnfg Config

	var data []byte
	if path == "" {
		data = defaultConfig
	} else {
		file, err := os.Open(path)
		if err != nil {
			return cnfg, err
		}
		data, err = io.ReadAll(file)
		if err != nil {
			return cnfg, err
		}
	}

	if err := yaml.Unmarshal(data, &cnfg); err != nil {
		return cnfg, err
	}

	return cnfg, nil
}
