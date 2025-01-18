package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/step"
)

type BedrockLLMForwarder struct {
	Bedrock BedrockClient
}

func NewBedrockLLMForwarder(l logger.Logger) LLMForwarder {
	bed, err := NewBedrock(l)
	if err != nil {
		l.Error("failed to create bedrock client: %v", err)
		panic(err)
	}
	return BedrockLLMForwarder{
		Bedrock: bed,
	}
}

func (a BedrockLLMForwarder) StartForward(input StartCompletionInput) ([]LLMMessage, error) {
	var history []LLMMessage
	params, initialHistory := a.createParams(input)

	history = append(history, initialHistory...)

	a.Bedrock.logger.Info(logger.Green(fmt.Sprintf("model: %s, sending message...\n", input.Model)))
	a.Bedrock.logger.Debug("system prompt:\n%s\n", input.SystemPrompt)
	a.Bedrock.logger.Debug("user prompt:\n%s\n", input.StartUserPrompt)
	resp, err := a.Bedrock.Messages.Create(context.TODO(), params)
	if err != nil {
		return nil, err
	}

	var toolCalls []ToolCall
	var text string
	for _, cont := range resp.Content {
		// discard text
		if cont.Type == "text" {
			text = cont.Text
			continue
		}
		if cont.Type == "tool_use" {
			j, err := json.Marshal(cont.Input)
			if err != nil {
				return nil, err
			}
			toolCalls = append(toolCalls, ToolCall{
				ToolCallerID: cont.ID,
				ToolName:     cont.Name,
				Argument:     string(j),
			})
		}
	}
	history = append(history, LLMMessage{
		Role:              LLMAssistant,
		FinishReason:      convertAnthoropicStopReasonToReason(resp.StopReason),
		RawContent:        text,
		ReturnedToolCalls: toolCalls,
	})

	a.Bedrock.logger.Info(logger.Yellow("returned messages:\n"))
	a.showDebugMessage(history[len(history)-1])

	return history, nil
}

func (a BedrockLLMForwarder) ForwardLLM(
	_ context.Context,
	input StartCompletionInput,
	llmContexts []step.ReturnToLLMContext,
	history []LLMMessage,
) ([]LLMMessage, error) {
	params, _ := a.createParams(input)

	// reset message
	params["messages"] = make([]J, 0)

	// build message from history
	for _, h := range history {
		switch h.Role {
		case LLMAssistant:
			if len(h.ReturnedToolCalls) > 0 {
				content := make([]J, 0)
				for _, v := range h.ReturnedToolCalls {
					var input map[string]any
					if err := json.Unmarshal([]byte(v.Argument), &input); err != nil {
						return nil, fmt.Errorf("failed to unmarshal tool argument: %w", err)
					}
					content = append(content, J{
						"type": "tool_use",
						"id":   v.ToolCallerID,
						"name": v.ToolName,
						// json marshal?
						"input": input,
					})
				}
				params["messages"] = append(params["messages"].([]J), J{
					"role":    "assistant",
					"content": content,
				})
			} else {
				params["messages"] = append(params["messages"].([]J), J{
					"role":    "assistant",
					"content": h.RawContent,
				})
			}
		case LLMUser:
			params["messages"] = append(params["messages"].([]J), J{
				"role":    "user",
				"content": h.RawContent,
			})

		// multiple contents in 1 message
		case LLMTool:
			// 本来は複数のLLM Messageを1つのmessageにまとめる必要がある
			params["messages"] = append(params["messages"].([]J), J{
				"role": "user",
				"content": []J{
					{
						"type":        "tool_result",
						"tool_use_id": h.RespondToolCall.ToolCallerID,
						"content":     h.RawContent,
					},
				},
			})
		default:
			return nil, fmt.Errorf("unknown role: %s", h.Role)
		}
	}

	// new message
	var newMsg LLMMessage
	content := make([]J, len(llmContexts))
	for i, v := range llmContexts {
		if v.ToolCallerID != "" {
			content[i] = J{
				"type":        "tool_result",
				"tool_use_id": v.ToolCallerID,
				"content":     v.Content,
			}

			newMsg = LLMMessage{
				Role:       LLMTool,
				RawContent: v.Content,
				RespondToolCall: ToolCall{
					ToolCallerID: v.ToolCallerID,
					ToolName:     v.ToolName,
				},
			}
		} else {
			params["messages"] = append(params["messages"].([]J), J{
				"role":    "user",
				"content": v.Content,
			})
			newMsg = LLMMessage{
				Role:       LLMUser,
				RawContent: v.Content,
			}
		}
		history = append(history, newMsg)
	}
	params["messages"] = append(params["messages"].([]J), J{
		"role":    "user",
		"content": content,
	})

	a.Bedrock.logger.Info(logger.Green(fmt.Sprintf("model: %s, sending message...\n", input.Model)))
	a.Bedrock.logger.Debug("%s\n", newMsg.RawContent)

	resp, err := a.Bedrock.Messages.Create(context.TODO(), params)
	if err != nil {
		return nil, err
	}

	// TODO: refactor with StartForward
	var toolCalls []ToolCall
	var text string
	for _, cont := range resp.Content {
		// assumption of only 1 text per content
		if cont.Type == "text" {
			text = cont.Text
			continue
		}
		if cont.Type == "tool_use" {
			j, err := json.Marshal(cont.Input)
			if err != nil {
				return nil, err
			}
			toolCalls = append(toolCalls, ToolCall{
				ToolCallerID: cont.ID,
				ToolName:     cont.Name,
				Argument:     string(j),
			})
		}
	}
	history = append(history, LLMMessage{
		Role:              LLMAssistant,
		FinishReason:      convertAnthoropicStopReasonToReason(resp.StopReason),
		RawContent:        text,
		ReturnedToolCalls: toolCalls,
	})

	a.Bedrock.logger.Info(logger.Yellow("returned messages:\n"))
	a.showDebugMessage(history[len(history)-1])

	return history, nil
}

// TODO: refactor with openai forwarder
func (a BedrockLLMForwarder) ForwardStep(_ context.Context, history []LLMMessage) step.Step {
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

func (a BedrockLLMForwarder) createParams(input StartCompletionInput) (J, []LLMMessage) {
	tools := make([]J, len(input.Functions))

	for i, f := range input.Functions {
		tools[i] = J{
			"name":         f.Name,
			"description":  f.Description,
			"input_schema": f.Parameters,
		}
	}

	body := J{
		"anthropic_version": "bedrock-2023-05-31",

		"system": input.SystemPrompt,
		"messages": []J{
			{"role": "user", "content": input.StartUserPrompt},
		},
		"temperature": 0.0,
		"tool_choice": J{
			"type": "auto",
			//"disable_parallel_tool_use": true,
		},
		"tools":      tools,
		"max_tokens": 8192, // TODO: max_tokens
	}

	return body, []LLMMessage{
		{
			Role:       LLMUser,
			RawContent: input.StartUserPrompt,
		},
	}
}

// TODO: refactor with openai debugging
func (a BedrockLLMForwarder) showDebugMessage(m LLMMessage) {
	a.Bedrock.logger.Debug(fmt.Sprintf("finish_reason: %s, role: %s, message.content: %s\n",
		m.FinishReason, m.Role, m.RawContent,
	))
	a.Bedrock.logger.Debug("tools:\n")
	for _, t := range m.ReturnedToolCalls {
		a.Bedrock.logger.Debug(fmt.Sprintf("id: %s, function_name:%s, function_args:%s\n",
			t.ToolCallerID, t.ToolName, t.Argument))
	}
}
