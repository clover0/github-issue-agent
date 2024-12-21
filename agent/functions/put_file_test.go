package functions_test

import (
	"os"
	"testing"

	"github/clover0/github-issue-agent/functions"
	"github/clover0/github-issue-agent/store"
	"github.com/stretchr/testify/assert"
)

func TestPutFile(t *testing.T) {
	t.Run("successfully creates a file", func(t *testing.T) {
		input := functions.PutFileInput{
			OutputPath:  "testdata/testfile.txt",
			ContentText: "Hello, World!",
		}

		file, err := functions.PutFile(input)
		assert.NoError(t, err)
		assert.Equal(t, input.OutputPath, file.Path)
		assert.Equal(t, input.ContentText+"\n", file.Content)

		// Clean up
		os.Remove(input.OutputPath)
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		input := functions.PutFileInput{
			OutputPath:  "",
			ContentText: "Hello, World!",
		}

		_, err := functions.PutFile(input)
		assert.Error(t, err)
	})

	t.Run("writes content correctly", func(t *testing.T) {
		input := functions.PutFileInput{
			OutputPath:  "testdata/testfile2.txt",
			ContentText: "Test Content",
		}

		file, err := functions.PutFile(input)
		assert.NoError(t, err)
		assert.Equal(t, input.ContentText+"\n", file.Content)

		// Clean up
		os.Remove(input.OutputPath)
	})
}
