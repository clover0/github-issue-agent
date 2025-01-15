package models

import (
	"context"

	"github.com/clover0/issue-agent/functions"
	"github.com/clover0/issue-agent/step"
)

// TODO: make no OpenAI dependency

type StartCompletionInput struct {
	Model           string
	SystemPrompt    string
	StartUserPrompt string
	Functions       []functions.Function
}

type LLMForwarder interface {
	StartForward(input StartCompletionInput) ([]LLMMessage, error)
	ForwardLLM(
		ctx context.Context,
		input StartCompletionInput,
		llmContexts []step.ReturnToLLMContext,
		history []LLMMessage,
	) ([]LLMMessage, error)
	ForwardStep(ctx context.Context, history []LLMMessage) step.Step
}

type LLMMessage struct {
	Role         MessageRole
	RawContent   string
	FinishReason MessageFinishReason

	// user to llm
	RespondToolCall ToolCall

	// llm to user
	ReturnedToolCalls []ToolCall

	// returned raw message struct from LLM API
	RawMessageStruct any
}

type ToolCall struct {
	ToolCallerID string
	ToolName     string
	Argument     string
}

type MessageRole string

const (
	LLMAssistant MessageRole = "assistant"
	LLMUser                  = "user"
	LLMSystem                = "system"
	LLMTool                  = "tool"
)

type MessageFinishReason string

const (
	FinishStop       MessageFinishReason = "stop"
	FinishToolCalls                      = "toolCalls"
	FinishLengthOver                     = "lengthOver"
)
