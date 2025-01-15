package models

import (
	"context"
	"os"

	"github.com/clover0/issue-agent/functions"
	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/step"
)

type OpenAILLMForwarder struct {
	openai OpenAI
}

func NewOpenAILLMForwarder(l logger.Logger) LLMForwarder {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic("OPENAI_API_KEY is not set")
	}

	return OpenAILLMForwarder{
		openai: NewOpenAI(l, apiKey),
	}
}

func (o OpenAILLMForwarder) StartForward(input StartCompletionInput) ([]LLMMessage, error) {
	return o.openai.StartCompletion(
		context.TODO(),
		StartCompletionInput{
			Model:           input.Model,
			SystemPrompt:    input.SystemPrompt,
			StartUserPrompt: input.StartUserPrompt,
			Functions:       functions.AllFunctions(),
		},
	)
}

func (o OpenAILLMForwarder) ForwardLLM(
	ctx context.Context,
	input StartCompletionInput,
	llmContexts []step.ReturnToLLMContext,
	history []LLMMessage,
) ([]LLMMessage, error) {
	return o.openai.ContinueCompletion(ctx, input, llmContexts, history)
}

func (o OpenAILLMForwarder) ForwardStep(ctx context.Context, history []LLMMessage) step.Step {
	return o.openai.CompletionNextStep(ctx, history)
}
