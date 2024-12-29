package functions

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/openai/openai-go"
	libstore "github/clover0/github-issue-agent/store"
)

func InitializeFunctions(
	noSubmit bool,
	repoService RepositoryService,
	allowFunctions []string,
) {
	if allowFunction(allowFunctions, FuncOpenFile) {
		InitOpenFileFunction()
	}
	if allowFunction(allowFunctions, FuncListFiles) {
		InitListFilesFunction()
	}
	if allowFunction(allowFunctions, FuncPutFile) {
		InitPutFileFunction()
	}
	if allowFunction(allowFunctions, FuncModifyFile) {
		InitModifyFileFunction()
	}
	// TODO:
	if !noSubmit && allowFunction(allowFunctions, FuncModifyFile) {
		InitSubmitFilesGitHubFunction()
	}
	if allowFunction(allowFunctions, FuncGetWebSearchResult) {
		InitGetWebSearchResult()
	}
	if allowFunction(allowFunctions, FuncGetWebPageFromURL) {
		InitFuncGetWebPageFromURLFunction()
	}
	if allowFunction(allowFunctions, FuncGetPullRequestDiff) {
		InitGetPullRequestFunction(repoService)
	}
}

func allowFunction(allowFunctions []string, name string) bool {
	return slices.Contains(allowFunctions, name)
}

type FuncName string

func (f FuncName) String() string {
	return string(f)
}

type Function struct {
	Name        FuncName
	Description string
	Func        any
	FuncType    reflect.Type
	Parameters  map[string]interface{}
}

var functionsMap = map[string]Function{}

// TODO: no dependent on openai-go
func (f Function) ToFunctionCalling() openai.FunctionDefinitionParam {
	return openai.FunctionDefinitionParam{
		Name:        openai.F(f.Name.String()),
		Description: openai.F(f.Name.String()),
		Parameters:  openai.F(openai.FunctionParameters(f.Parameters)),
	}
}

func FunctionByName(name string) (Function, error) {
	if f, ok := functionsMap[name]; ok {
		return f, nil
	}

	return Function{}, errors.New(fmt.Sprintf("%s does not exist in functions", name))
}

// AllFunctions returns all functions
// WARNING: Called initialization functions before calling this function
func AllFunctions() []Function {
	var fns []Function
	for _, f := range functionsMap {
		fns = append(fns, f)
	}
	return fns
}

func marshalFuncArgs(args string, input any) error {
	return json.Unmarshal([]byte(args), &input)
}

const defaultSuccessReturning = "The process was successful!"

type optionalArg struct {
	SubmitFilesFunction SubmitFilesCallerType
}

type FunctionOption func(o *optionalArg)

func SetSubmitFiles(fn SubmitFilesCallerType) FunctionOption {
	return func(o *optionalArg) {
		o.SubmitFilesFunction = fn
	}
}

func ExecFunction(store *libstore.Store, funcName FuncName, argsJson string, optArg ...FunctionOption) (string, error) {
	option := &optionalArg{}
	for _, o := range optArg {
		o(option)
	}
	switch funcName {
	case FuncOpenFile:
		// TODO: logger from context
		fmt.Println("functions: do open_file")
		input := OpenFileInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		file, err := OpenFile(input)
		if err != nil {
			return "", err
		}
		return file.Content, nil

	case FuncListFiles:
		fmt.Println("functions: do list_files")
		input := ListFilesInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		files, err := ListFiles(input)
		if err != nil {
			return "", err
		}
		return strings.Join(files, "\n"), nil

	case FuncPutFile:
		fmt.Println("functions: do put_file")
		input := PutFileInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		file, err := PutFile(input)
		if err != nil {
			return "", err
		}
		StoreFileAfterPutFile(store, file)
		return defaultSuccessReturning, nil

	case FuncModifyFile:
		fmt.Println("functions: do modify_file")
		input := ModifyFileInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		file, err := ModifyFile(input)
		if err != nil {
			return "", err
		}
		StoreFileAfterModifyFile(store, file)
		return defaultSuccessReturning, nil

	case FuncSubmitFiles:
		fmt.Println("functions: do submit changes")
		input := SubmitFilesInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		out, err := SubmitFiles(option.SubmitFilesFunction, input)
		if err != nil {
			return "", err
		}

		// NOTE: we would like to use any key, but for ease of implementation, we keep this as a simple implementation.
		SubmitFilesAfter(store, libstore.LastSubmissionKey, out)

		return defaultSuccessReturning, nil

	case FuncGetWebSearchResult:
		fmt.Println("functions: do get_latest_version_search_result")
		input := GetWebSearchResultInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}

		r, err := GetWebSearchResult(input)
		if err != nil {
			return "", err
		}
		return r, nil

	case FuncGetWebPageFromURL:
		fmt.Println("functions: do get_web_page_from_url")
		input := GetWebPageFromURLInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}

		r, err := GetWebPageFromURL(input)
		if err != nil {
			return "", err
		}
		return r, nil

	case FuncGetPullRequestDiff:
		fmt.Println("functions: do get_pull_request_diff")
		input := GetPullRequestDiffInput{}
		if err := marshalFuncArgs(argsJson, &input); err != nil {
			return "", fmt.Errorf("failed to unmarshal args: %w", err)
		}
		fn, ok := functionsMap[FuncGetPullRequestDiff].Func.(GetPullRequestDiffType)
		if !ok {
			return "", fmt.Errorf("cat not call %s function", FuncGetPullRequestDiff)
		}
		return fn(input)

	}

	return "", errors.New("function not found")
}
