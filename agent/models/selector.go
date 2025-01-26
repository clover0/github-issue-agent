package models

import (
	"fmt"
	"strings"

	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/util"
)

func SelectForwarder(lo logger.Logger, model string) (LLMForwarder, error) {
	if util.IsAWSBedrockModel(model) {
		return NewBedrockLLMForwarder(lo), nil
	}
	if strings.HasPrefix(model, "gpt") {
		return NewOpenAILLMForwarder(lo), nil
	}

	if strings.HasPrefix(model, "claude") {
		return NewAnthropicLLMForwarder(lo), nil
	}

	if model == "" {
		return nil, fmt.Errorf("model is not specified")
	}

	return nil, fmt.Errorf("SelectForwarder: model %s is not supported", model)
}
