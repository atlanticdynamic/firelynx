//go:build integration

package starlark_test

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed starlark_tool.star
var starlarkToolScript []byte

func TestStarlarkToolExample(t *testing.T) {
	t.Parallel()

	// Create temp directory and write embedded script to disk
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "starlark_tool.star")
	err := os.WriteFile(scriptPath, starlarkToolScript, 0o644)
	require.NoError(t, err, "Should be able to write script to temp file")

	tests := []struct {
		name         string
		staticData   map[string]any
		runtimeArgs  map[string]any
		expectedJSON string
		expectError  bool
	}{
		{
			name: "echo_operation",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "echo",
				"input":     "hello world",
			},
			expectedJSON: `{"content":"hello world","isError":false}`,
		},
		{
			name: "count_chars_operation",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "count_chars",
				"input":     "hello",
			},
			expectedJSON: `{"content":5,"isError":false}`,
		},
		{
			name: "split_words_operation",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "split_words",
				"input":     "hello world test",
			},
			expectedJSON: `{"content":["hello","world","test"],"isError":false}`,
		},
		{
			name: "missing_operation",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"input": "test",
				// No operation provided
			},
			expectedJSON: `{"content":"Operation is required","isError":true}`,
		},
		{
			name: "unsupported_operation",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "unknown",
				"input":     "test",
			},
			expectedJSON: `{"content":"Unsupported operation: unknown","isError":true}`,
		},
		{
			name: "count_chars_empty_input",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "count_chars",
				"input":     "",
			},
			expectedJSON: `{"content":0,"isError":false}`,
		},
		{
			name: "split_words_single_word",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "split_words",
				"input":     "single",
			},
			expectedJSON: `{"content":["single"],"isError":false}`,
		},
		{
			name: "split_words_empty_input",
			staticData: map[string]any{
				"config": map[string]any{
					"version": "1.0",
				},
			},
			runtimeArgs: map[string]any{
				"operation": "split_words",
				"input":     "",
			},
			expectedJSON: `{"content":[],"isError":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script loader from disk
			scriptLoader, err := loader.NewFromDisk(scriptPath)
			require.NoError(t, err, "Should be able to create script loader")

			// Create Starlark evaluator
			starlarkEval, err := starlark.FromStarlarkLoader(nil, scriptLoader)
			require.NoError(t, err, "Should be able to create Starlark evaluator")

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
			result, err := starlarkEval.Eval(enrichedCtx)
			require.NoError(t, err)

			// Get the result as JSON
			resultValue := result.Interface()
			actualJSON, err := json.Marshal(resultValue)
			require.NoError(t, err, "Result should be JSON serializable")

			// Compare JSON responses
			var actualMap map[string]any
			err = json.Unmarshal(actualJSON, &actualMap)
			require.NoError(t, err)

			var expectedMap map[string]any
			err = json.Unmarshal([]byte(tt.expectedJSON), &expectedMap)
			require.NoError(t, err)

			assert.Equal(t, expectedMap, actualMap, "Script response should match expected JSON")

			t.Logf("Script response: %s", string(actualJSON))
		})
	}
}

func TestStarlarkToolDataAccess(t *testing.T) {
	// Test that the script can access nested data from static configuration
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "starlark_tool.star")
	err := os.WriteFile(scriptPath, starlarkToolScript, 0o644)
	require.NoError(t, err, "Should be able to write script to temp file")

	// Provide static data with nested config
	staticData := map[string]any{
		"config": map[string]any{
			"max_length": 100,
			"enabled":    true,
		},
		"version": "2.0",
	}

	runtimeArgs := map[string]any{
		"operation": "echo",
		"input":     "test data access",
	}

	// Create Starlark evaluator
	scriptLoader, err := loader.NewFromDisk(scriptPath)
	require.NoError(t, err, "Should be able to create script loader")

	starlarkEval, err := starlark.FromStarlarkLoader(nil, scriptLoader)
	require.NoError(t, err, "Should be able to create Starlark evaluator")

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

	result, err := starlarkEval.Eval(enrichedCtx)
	require.NoError(t, err)

	// Should work successfully - the script accesses config under data namespace
	resultValue := result.Interface()
	actualJSON, err := json.Marshal(resultValue)
	require.NoError(t, err)

	expectedJSON := `{"content":"test data access","isError":false}`
	var actualMap, expectedMap map[string]any
	err = json.Unmarshal(actualJSON, &actualMap)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(expectedJSON), &expectedMap)
	require.NoError(t, err)

	assert.Equal(t, expectedMap, actualMap, "Should access nested data correctly")
	t.Logf("Data access test result: %s", string(actualJSON))
}
