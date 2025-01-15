package functions

import (
	"io"
	"log"
	"os"

	"github.com/clover0/issue-agent/store"
)

const FuncOpenFile = "open_file"

func InitOpenFileFunction() Function {
	f := Function{
		Name:        FuncOpenFile,
		Description: "Open the file full content",
		Func:        OpenFile,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The path of the file to open",
				},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	}

	functionsMap[FuncOpenFile] = f

	return f
}

type OpenFileInput struct {
	Path string
}

func OpenFile(input OpenFileInput) (store.File, error) {
	if err := guardPath(input.Path); err != nil {
		return store.File{}, err
	}

	file, err := os.Open(input.Path)
	if err != nil {
		return store.File{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	return store.File{
		Path:    input.Path,
		Content: string(data),
	}, nil
}
