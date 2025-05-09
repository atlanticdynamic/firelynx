package echo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEcho_Type(t *testing.T) {
	echo := NewEcho("test-response")
	assert.Equal(t, "echo", echo.Type())
}

func TestEcho_Validate(t *testing.T) {
	tests := []struct {
		name    string
		echo    *Echo
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid echo app",
			echo:    NewEcho("test-response"),
			wantErr: false,
		},
		{
			name:    "empty response",
			echo:    NewEcho(""),
			wantErr: true,
			errMsg:  "missing required field: echo app response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.echo.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEcho_String(t *testing.T) {
	echo := NewEcho("test-response")
	assert.Equal(t, "Echo App (response: test-response)", echo.String())
}

func TestEcho_ToTree(t *testing.T) {
	echo := NewEcho("test-response")
	tree := echo.ToTree()

	// Get the underlying tree for inspection
	treeObj := tree.Tree()

	// Since we can't easily inspect the rendered tree, we'll just verify it's not nil
	assert.NotNil(t, treeObj)
}
