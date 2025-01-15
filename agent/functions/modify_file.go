package functions

import (
	"fmt"
	"os"

	"github.com/clover0/issue-agent/store"
)

const FuncModifyFile = "modify_file"

func InitModifyFileFunction() Function {
	f := Function{
		Name: FuncModifyFile,
		Description: `Modify the file at output_path with the contents of content_text.
Modified file must be full content including modified content`,
		Func: ModifyFile,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"output_path": map[string]interface{}{
					"type":        "string",
					"description": "Path of the file to be modified to the new content",
				},
				"content_text": map[string]interface{}{
					"type":        "string",
					"description": "The new content of the file",
				},
			},
			"required":             []string{"output_path", "content_text"},
			"additionalProperties": false,
		},
	}

	functionsMap[FuncModifyFile] = f

	return f
}

type ModifyFileInput struct {
	OutputPath  string `json:"output_path"`
	ContentText string `json:"content_text"`
}

func ModifyFile(input ModifyFileInput) (store.File, error) {
	if err := guardPath(input.OutputPath); err != nil {
		return store.File{}, err
	}

	var file store.File
	f, err := os.Create(input.OutputPath)
	if err != nil {
		return file, fmt.Errorf("modify %s: %w", input.OutputPath, err)
	}
	defer f.Close()

	// EOF should be a newline
	if len(input.ContentText) != 0 && input.ContentText[len(input.ContentText)-1] != '\n' {
		input.ContentText += "\n"
	}

	if _, err := f.WriteString(input.ContentText); err != nil {
		return file, err
	}

	return store.File{
		Path:    input.OutputPath,
		Content: input.ContentText,
	}, nil
}
