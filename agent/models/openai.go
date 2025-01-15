package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/step"
)

type OpenAI struct {
	client *openai.Client
	logger logger.Logger
}

func NewOpenAI(logger logger.Logger, apiKey string) OpenAI {
	return OpenAI{
		logger: logger,
		client: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
	}
}

func (o OpenAI) createCompletionParams(input StartCompletionInput) (openai.ChatCompletionNewParams, []LLMMessage) {
	toolFuncs := make([]openai.ChatCompletionToolParam, len(input.Functions))
	for i, f := range input.Functions {
		toolFuncs[i] = openai.ChatCompletionToolParam{
			Function: openai.F(f.ToFunctionCalling()),
			Type:     openai.F(openai.ChatCompletionToolTypeFunction),
		}
	}

	historyInitial := []LLMMessage{
		{
			Role:       LLMSystem,
			RawContent: input.SystemPrompt,
		},
		{
			Role:       LLMUser,
			RawContent: input.StartUserPrompt,
		},
	}

	return openai.ChatCompletionNewParams{
		Model: openai.F(input.Model),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(input.SystemPrompt),
			openai.UserMessage(input.StartUserPrompt),
		}),
		Temperature: openai.F(0.0),
		Tools:       openai.F(toolFuncs),
	}, historyInitial
}

func (o OpenAI) StartCompletion(ctx context.Context, input StartCompletionInput) ([]LLMMessage, error) {
	var history []LLMMessage
	params, historyInitial := o.createCompletionParams(input)
	history = append(history, historyInitial...)

	o.logger.Info(logger.Green(fmt.Sprintf("model: %s, sending message...\n", input.Model)))
	o.logger.Debug("system prompt:\n%s\n", input.SystemPrompt)
	o.logger.Debug("user prompt:\n%s\n", input.StartUserPrompt)
	chat, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	msg := chat.Choices[0]
	lastMsg := LLMMessage{
		Role:              LLMAssistant,
		RawContent:        msg.Message.Content,
		FinishReason:      convertToFinishReason(msg.FinishReason),
		ReturnedToolCalls: convertToToolCalls(msg.Message.ToolCalls),
		RawMessageStruct:  msg.Message,
	}
	history = append(history, lastMsg)

	o.logger.Debug(fmt.Sprintf("prompt token: %d, completion token: %d\n",
		chat.Usage.PromptTokens, chat.Usage.CompletionTokens,
	))

	o.logger.Info(logger.Yellow("returned messages:\n"))
	o.debugShowChoice(history)

	return history, nil
}

func (o OpenAI) ContinueCompletion(
	ctx context.Context,
	input StartCompletionInput,
	llmContexts []step.ReturnToLLMContext,
	history []LLMMessage,
) ([]LLMMessage, error) {
	params, _ := o.createCompletionParams(input)

	// build from history
	params.Messages.Value = []openai.ChatCompletionMessageParamUnion{}
	for _, h := range history {
		switch h.Role {
		case LLMAssistant:
			if h.RawMessageStruct == nil {
				return nil, errors.New("rawMessageStruct should not be nil. But it is nil")
			}

			m, ok := h.RawMessageStruct.(openai.ChatCompletionMessage)
			if !ok {
				return nil, errors.New("RawMessageStruct can't convert ChatCompletionMessage")
			}

			params.Messages.Value = append(params.Messages.Value, m)
		case LLMUser:
			params.Messages.Value = append(params.Messages.Value, openai.UserMessage(h.RawContent))
		case LLMSystem:
			params.Messages.Value = append(params.Messages.Value, openai.SystemMessage(h.RawContent))
		case LLMTool:
			params.Messages.Value = append(params.Messages.Value,
				openai.ToolMessage(h.RespondToolCall.ToolCallerID, h.RawContent),
			)
		}
	}

	// new message
	var newMsg LLMMessage
	for _, v := range llmContexts {
		if v.ToolCallerID != "" {
			// tool message
			params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(v.ToolCallerID, v.Content))
			newMsg = LLMMessage{
				Role:       LLMTool,
				RawContent: v.Content,
				RespondToolCall: ToolCall{
					ToolCallerID: v.ToolCallerID,
					ToolName:     v.ToolName,
				},
			}
		} else {
			// user message
			params.Messages.Value = append(params.Messages.Value, openai.UserMessage(v.Content))
			newMsg = LLMMessage{
				Role:       LLMUser,
				RawContent: v.Content,
			}
		}
		history = append(history, newMsg)
	}

	o.debugShowSendingMsg(params)
	chat, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("continue completion error: %w", err)
	}

	msg := chat.Choices[0]
	lastMsg := LLMMessage{
		Role:              LLMAssistant,
		RawContent:        msg.Message.Content,
		FinishReason:      convertToFinishReason(msg.FinishReason),
		ReturnedToolCalls: convertToToolCalls(msg.Message.ToolCalls),
		RawMessageStruct:  msg.Message,
	}
	history = append(history, lastMsg)

	o.logger.Info(logger.Yellow("returned messages:\n"))
	o.debugShowChoice(history)

	return history, nil
}

func convertToFinishReason(finishReason openai.ChatCompletionChoicesFinishReason) MessageFinishReason {
	switch finishReason {
	case openai.ChatCompletionChoicesFinishReasonLength:
		return FinishLengthOver
	case openai.ChatCompletionChoicesFinishReasonStop:
		return FinishStop
	case openai.ChatCompletionChoicesFinishReasonToolCalls:
		return FinishToolCalls
	default:
		panic(fmt.Sprintf("convertToFinishReason: unknown finish reason: %s", finishReason))
	}
}

func convertToToolCalls(toolCalls []openai.ChatCompletionMessageToolCall) []ToolCall {
	var res []ToolCall
	for _, v := range toolCalls {
		res = append(res, ToolCall{
			ToolCallerID: v.ID,
			ToolName:     v.Function.Name,
			Argument:     v.Function.Arguments,
		})
	}
	return res
}

func (o OpenAI) CompletionNextStep(_ context.Context, history []LLMMessage) step.Step {
	// last message
	lastMsg := history[len(history)-1]

	switch lastMsg.FinishReason {
	case FinishStop:
		return step.NewWaitingInstructionStep(lastMsg.RawContent)
	case FinishToolCalls:
		var input []step.FunctionsInput
		for _, v := range lastMsg.ReturnedToolCalls {
			input = append(input, step.FunctionsInput{
				FuncName:     v.ToolName,
				FunctionArgs: v.Argument,
				ToolCallerID: v.ToolCallerID,
			})
		}
		return step.NewExecStep(input)
	case FinishLengthOver:
		return step.NewUnrecoverableStep(fmt.Errorf("chat completion length error"))
	}

	return step.NewUnknownStep()
}

func (o OpenAI) debugShowSendingMsg(param openai.ChatCompletionNewParams) {
	if len(param.Messages.Value) > 0 {
		o.logger.Info(logger.Green(fmt.Sprintf("model: %s, sending messages:\n", param.Model.String())))
		// TODO: show all messages. But now, show only the last message
		o.logger.Debug(fmt.Sprintf("%s\n", param.Messages.Value[len(param.Messages.Value)-1]))
	}
}

func (o OpenAI) debugShowChoice(history []LLMMessage) {
	last := history[len(history)-1]
	o.logger.Debug(fmt.Sprintf("finish_reason: %s, role: %s, message.content: %s\n",
		last.FinishReason, last.Role, last.RawContent,
	))
	o.logger.Debug("tools:\n")
	for _, t := range last.ReturnedToolCalls {
		o.logger.Debug(fmt.Sprintf("id: %s, function_name:%s, function_args:%s\n",
			t.ToolCallerID, t.ToolName, t.Argument))
	}
}
