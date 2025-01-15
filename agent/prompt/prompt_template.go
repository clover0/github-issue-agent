package prompt

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/clover0/issue-agent/config/template"
)

type PromptTemplate struct {
	Agents []struct {
		Name           string `yaml:"name"`
		SystemTemplate string `yaml:"system_prompt"`
		UserTemplate   string `yaml:"user_prompt"`
	}
}

func LoadPrompt(filePath string) (PromptTemplate, error) {
	var pt PromptTemplate

	var data []byte
	if filePath == "" {
		data = template.DefaultTemplate()
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			return pt, err
		}
		defer file.Close()

		data, err = io.ReadAll(file)
		if err != nil {
			return pt, err
		}
	}

	err := yaml.Unmarshal(data, &pt)
	if err != nil {
		return pt, err
	}

	return pt, nil
}
