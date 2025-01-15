package step

import (
	"fmt"

	"github.com/clover0/issue-agent/functions"
)

// TODO: Step owned Agent

type DoType string
type JSONString string

func (j JSONString) String() string {
	return string(j)
}

const (
	Unknown            = DoType("unknown")
	Unrecoverable      = DoType("unrecoverable")
	Exec               = DoType("exec")
	WaitingInstruction = DoType("waiting_instruction")
	ReturnToLLM        = DoType("return_to_llm")
	Submit             = DoType("submit")
)

type Step struct {
	Do                  DoType
	ReturnToLLMContexts []ReturnToLLMContext
	FunctionContexts    []FunctionContext
	UnrecoverableErr    error

	// TODO: data modeling
	LastOutput string
}

type ReturnToLLMContext struct {
	ToolCallerID string // TODO: non OpenAI dependency
	ToolName     string
	Content      string
}

type FunctionContext struct {
	Function     functions.Function
	FunctionArgs JSONString
	ToolCallerID string // TODO: non OpenAI dependency
}

type FunctionsInput struct {
	FuncName     string
	FunctionArgs string
	ToolCallerID string // TODO: non OpenAI dependency
}

func NewExecStep(fnsInput []FunctionsInput) Step {
	var contexts []FunctionContext
	for _, v := range fnsInput {
		f, err := functions.FunctionByName(v.FuncName)
		if err != nil {
			return NewUnrecoverableStep(fmt.Errorf("function not found %s: %w", v.FuncName, err))
		}
		contexts = append(contexts, FunctionContext{
			Function:     f,
			FunctionArgs: JSONString(v.FunctionArgs),
			ToolCallerID: v.ToolCallerID,
		})
	}

	return Step{
		Do:               Exec,
		FunctionContexts: contexts,
	}
}

func NewWaitingInstructionStep(output string) Step {
	return Step{
		Do:         WaitingInstruction,
		LastOutput: output,
	}
}

func NewUnknownStep() Step {
	return Step{Do: Unknown}
}

func NewUnrecoverableStep(err error) Step {
	return Step{
		Do:               Unrecoverable,
		UnrecoverableErr: err,
	}
}

type ReturnToLLMInput struct {
	ToolCallerID string // TODO: non OpenAI dependency
	ToolName     string
	Content      string
}

func NewReturnToLLMStep(input []ReturnToLLMInput) Step {
	var contexts []ReturnToLLMContext
	for _, v := range input {
		contexts = append(contexts, ReturnToLLMContext{
			ToolCallerID: v.ToolCallerID,
			ToolName:     v.ToolName,
			Content:      v.Content,
		})
	}
	return Step{
		Do:                  ReturnToLLM,
		ReturnToLLMContexts: contexts,
	}
}
