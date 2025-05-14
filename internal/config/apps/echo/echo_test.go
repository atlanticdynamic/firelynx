package echo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoApp_Type(t *testing.T) {
	echo := New()
	assert.Equal(t, "echo", echo.Type())
}

func TestEchoApp_Validate(t *testing.T) {
	tests := []struct {
		name    string
		echo    *EchoApp
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid echo app",
			echo:    New(),
			wantErr: false,
		},
		{
			name:    "empty response",
			echo:    &EchoApp{Response: ""},
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

func TestEchoApp_String(t *testing.T) {
	echo := New()
	assert.Contains(t, echo.String(), "Echo App (response:")
}

func TestEchoApp_ToTree(t *testing.T) {
	echo := New()
	tree := echo.ToTree()

	// Get the underlying tree for inspection
	treeObj := tree.Tree()

	// Since we can't easily inspect the rendered tree, we'll just verify it's not nil
	assert.NotNil(t, treeObj)
}
