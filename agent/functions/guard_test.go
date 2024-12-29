package functions_test

import (
	"testing"

	"github/clover0/github-issue-agent/functions"
	"github/clover0/github-issue-agent/test/assert"
)

func TestGuardPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path    string
		wantErr bool
		errMsg  string
	}{
		"valid local path": {
			path:    "folder/file.txt",
			wantErr: false,
		},
		"valid path with multiple segments": {
			path:    "folder/subfolder/file.txt",
			wantErr: false,
		},
		"starts with parent directory": {
			path:    "../file.txt",
			wantErr: true,
		},
		"contains parent directory": {
			path:    "folder/../file.txt",
			wantErr: false,
		},
		"contains tilde": {
			path:    "~/folder/file.txt",
			wantErr: true,
		},
		"contains double slash": {
			path:    "folder//file.txt",
			wantErr: false,
		},
		"starts with slash": {
			path:    "/folder/file.txt",
			wantErr: true,
		},
		"empty path": {
			path:    "",
			wantErr: false,
		},
		"dot path": {
			path:    ".",
			wantErr: false,
		},
		"complex invalid path": {
			path:    "../folder/~/file.txt",
			wantErr: true,
			errMsg:  "path ../folder/~/file.txt attempts to access parent directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := functions.GuardPath(tt.path)

			if !tt.wantErr {
				assert.Nil(t, err)
				return
			}

			assert.HasError(t, err)
		})
	}
}
