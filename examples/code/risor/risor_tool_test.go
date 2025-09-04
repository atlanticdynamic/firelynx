//go:build integration

package risor_test

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed risor_tool.risor
var risorToolScript []byte

func TestRisorToolExample(t *testing.T) {
	t.Parallel()

	// Create temp directory and write embedded script to disk
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "risor_tool.risor")
	err := os.WriteFile(scriptPath, risorToolScript, 0o644)
	require.NoError(t, err, "Should be able to write script to temp file")

	tests := []struct {
		name         string
		staticData   map[string]any
		runtimeArgs  map[string]any
		expectedJSON string
		expectError  bool
	}{
		{
			name: "default_operation_fallback",
			staticData: map[string]any{
				"default_operation": "reverse",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"input": "hello",
				// No operation provided - should use default
			},
			expectedJSON: `{"isError":false,"content":"olleh"}`,
		},
		{
			name: "runtime_overrides_default",
			staticData: map[string]any{
				"default_operation": "reverse",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "echo",
				"input":     "hello world",
			},
			expectedJSON: `{"isError":false,"content":"hello world"}`,
		},
		{
			name: "input_length_validation",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  5,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "echo",
				"input":     "this is too long",
			},
			expectedJSON: `{"isError":true,"content":"Input exceeds maximum length of 5"}`,
		},
		{
			name: "uppercase_disabled_by_config",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  100,
				"allow_uppercase":   false,
			},
			runtimeArgs: map[string]any{
				"operation": "uppercase",
				"input":     "hello",
			},
			expectedJSON: `{"isError":true,"content":"Uppercase operation disabled by configuration"}`,
		},
		{
			name: "uppercase_enabled_by_config",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "uppercase",
				"input":     "hello",
			},
			expectedJSON: `{"isError":false,"content":"HELLO"}`,
		},
		{
			name: "reverse_operation",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "reverse",
				"input":     "hello world",
			},
			expectedJSON: `{"isError":false,"content":"dlrow olleh"}`,
		},
		{
			name: "lowercase_operation",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "lowercase",
				"input":     "HELLO WORLD",
			},
			expectedJSON: `{"isError":false,"content":"hello world"}`,
		},
		{
			name: "unsupported_operation",
			staticData: map[string]any{
				"default_operation": "echo",
				"max_input_length":  100,
				"allow_uppercase":   true,
			},
			runtimeArgs: map[string]any{
				"operation": "unsupported",
				"input":     "test",
			},
			expectedJSON: `{"isError":true,"content":"Unsupported operation: unsupported"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script loader from disk
			scriptLoader, err := loader.NewFromDisk(scriptPath)
			require.NoError(t, err, "Should be able to create script loader")

			// Create Risor evaluator
			risorEval, err := risor.FromRisorLoader(nil, scriptLoader)
			require.NoError(t, err, "Should be able to create Risor evaluator")

			// Create script data that matches the namespace pattern:
			// {"data": {static_config}, "args": {tool_arguments}}
			scriptData := map[string]any{
				"data": tt.staticData,
				"args": tt.runtimeArgs,
			}

			// Create context provider and add script data
			ctx := t.Context()
			contextProvider := data.NewContextProvider(constants.EvalData)
			enrichedCtx, err := contextProvider.AddDataToContext(ctx, scriptData)
			require.NoError(t, err)

			// Execute the script
			result, err := risorEval.Eval(enrichedCtx)
			require.NoError(t, err)

			// Get the result as JSON
			resultValue := result.Interface()
			actualJSON, err := json.Marshal(resultValue)
			require.NoError(t, err, "Result should be JSON serializable")

			// Compare JSON responses
			assert.JSONEq(t, tt.expectedJSON, string(actualJSON), "Script response should match expected JSON")

			t.Logf("Script response: %s", string(actualJSON))
		})
	}
}

func TestRisorToolHardcodedFallbacks(t *testing.T) {
	// Test that hardcoded fallbacks work when static data doesn't provide values
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "risor_tool.risor")
	err := os.WriteFile(scriptPath, risorToolScript, 0o644)
	require.NoError(t, err, "Should be able to write script to temp file")

	// Provide minimal static data - missing some expected fields
	staticData := map[string]any{
		// Missing: default_operation, max_input_length, allow_uppercase
		"some_other_field": "value",
	}

	runtimeArgs := map[string]any{
		"operation": "echo",
		"input":     "test hardcoded fallbacks",
	}

	// Create script loader from disk
	scriptLoader, err := loader.NewFromDisk(scriptPath)
	require.NoError(t, err, "Should be able to create script loader")

	// Create Risor evaluator
	risorEval, err := risor.FromRisorLoader(nil, scriptLoader)
	require.NoError(t, err, "Should be able to create Risor evaluator")

	// Create script data with namespace structure
	scriptData := map[string]any{
		"data": staticData,
		"args": runtimeArgs,
	}

	// Execute script
	ctx := t.Context()
	contextProvider := data.NewContextProvider(constants.EvalData)
	enrichedCtx, err := contextProvider.AddDataToContext(ctx, scriptData)
	require.NoError(t, err)

	result, err := risorEval.Eval(enrichedCtx)
	require.NoError(t, err)

	// Should work with hardcoded fallbacks:
	// default_operation: "echo" (hardcoded fallback)
	// max_input_length: 1000 (hardcoded fallback)
	// allow_uppercase: true (hardcoded fallback)
	resultValue := result.Interface()
	actualJSON, err := json.Marshal(resultValue)
	require.NoError(t, err)

	expectedJSON := `{"isError":false,"content":"test hardcoded fallbacks"}`
	assert.JSONEq(t, expectedJSON, string(actualJSON), "Should work with hardcoded fallbacks")
	t.Logf("Hardcoded fallbacks test result: %s", string(actualJSON))
}
