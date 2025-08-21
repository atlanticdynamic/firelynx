package echo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoApp_Type(t *testing.T) {
	echo := New("test-echo")
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
			echo:    New("test-echo"),
			wantErr: false,
		},
		{
			name:    "empty ID",
			echo:    &EchoApp{ID: "", Response: "Hello"},
			wantErr: true,
			errMsg:  "missing required field: echo app ID",
		},
		{
			name:    "empty response",
			echo:    &EchoApp{ID: "test", Response: ""},
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
	echo := New("test-echo")
	assert.Contains(t, echo.String(), "Echo App (response:")
}

func TestEchoApp_ToTree(t *testing.T) {
	echo := New("test-echo")
	tree := echo.ToTree()

	// Get the underlying tree for inspection
	treeObj := tree.Tree()

	// Since we can't easily inspect the rendered tree, we'll just verify it's not nil
	assert.NotNil(t, treeObj)
}

func TestEchoApp_Interpolation(t *testing.T) {
	// Set up test environment variable
	require.NoError(t, os.Setenv("TEST_MESSAGE", "Hello from environment"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("TEST_MESSAGE"))
	})

	tests := []struct {
		name             string
		echo             *EchoApp
		expectedID       string
		expectedResponse string
		wantErr          bool
	}{
		{
			name: "response interpolation works",
			echo: &EchoApp{
				ID:       "app-${TEST_MESSAGE}",   // Should NOT be interpolated (env_interpolation:"no")
				Response: "Echo: ${TEST_MESSAGE}", // Should be interpolated (env_interpolation:"yes")
			},
			expectedID:       "app-${TEST_MESSAGE}",          // Not interpolated
			expectedResponse: "Echo: Hello from environment", // Interpolated
			wantErr:          false,
		},
		{
			name: "missing env var with default in response",
			echo: &EchoApp{
				ID:       "test-id",
				Response: "Hello ${MISSING_VAR:World}!", // Should use default
			},
			expectedID:       "test-id",
			expectedResponse: "Hello World!",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.echo.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.echo.ID, "ID should not be interpolated")
				assert.Equal(t, tt.expectedResponse, tt.echo.Response, "Response should be interpolated")
			}
		})
	}
}
