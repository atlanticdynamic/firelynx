//go:build integration

package mcp

import (
	_ "embed"
	"testing"
	"time"

	mcpapp "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/echo_args.risor
var echoArgsRisorScript string

//go:embed testdata/static_data_access.risor
var staticDataAccessRisorScript string

//go:embed testdata/args_access.risor
var argsAccessRisorScript string

//go:embed testdata/error_handling.risor
var errorHandlingRisorScript string

//go:embed testdata/analyzer.star
var analyzerStarlarkScript string

// TestScriptBridgeDirectly tests the script tool bridge without HTTP layer
func TestScriptBridgeDirectly(t *testing.T) {
	ctx := t.Context()

	t.Run("risor echo args tool", func(t *testing.T) {
		// Create Risor evaluator using embedded script
		risorEval := &evaluators.RisorEvaluator{
			Code:    echoArgsRisorScript,
			Timeout: 5 * time.Second,
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		// Create static data
		staticData := &staticdata.StaticData{
			Data: map[string]any{
				"max_result": 1000,
				"precision":  "high",
			},
		}

		// Create script tool handler
		handler := &mcpapp.ScriptToolHandler{
			Evaluator:  risorEval,
			StaticData: staticData,
		}

		// Create MCP tool
		mcpTool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, mcpTool)
		require.NotNil(t, mcpHandler)

		// Test tool execution
		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{
				"expression": "10 + 5",
			},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check that we got content back
		assert.Greater(t, len(result.Content), 0)

		// Check content contains the echoed expression (simplified test)
		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "10 + 5", "Script should echo back the expression from arguments")
	})

	t.Run("risor tool with error", func(t *testing.T) {
		risorEval := &evaluators.RisorEvaluator{
			Code:    string(calculatorRisorScript),
			Timeout: 5 * time.Second,
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		handler := &mcpapp.ScriptToolHandler{
			Evaluator: risorEval,
		}

		_, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)

		// Test with division by zero
		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{
				"expression": "10 / 0",
			},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		assert.NoError(t, err, "Handler should not return Go error for script runtime errors")
		assert.NotNil(t, result, "Result should be returned")
		assert.True(t, result.IsError, "Result should indicate error")

		// Verify error message is in content
		assert.Greater(t, len(result.Content), 0, "Error result should have content")
		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		assert.True(t, ok, "Error content should be text")
		assert.Contains(t, textContent.Text, "Division by zero", "Error message should be in content")
	})

	t.Run("starlark data analyzer tool", func(t *testing.T) {
		// Create Starlark evaluator using embedded script
		starlarkEval := &evaluators.StarlarkEvaluator{
			Code:    analyzerStarlarkScript,
			Timeout: 5 * time.Second,
		}

		err := starlarkEval.Validate()
		require.NoError(t, err)

		// Create static data
		staticData := &staticdata.StaticData{
			Data: map[string]any{
				"max_depth": 10,
			},
		}

		handler := &mcpapp.ScriptToolHandler{
			Evaluator:  starlarkEval,
			StaticData: staticData,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		assert.Empty(t, tool.Name, "Tool name should be empty as it's set by the caller during tool registration")
		assert.Empty(t, tool.Description, "Tool description should be empty as it's set by the caller during tool registration")

		// Test with JSON data
		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{
				"data": map[string]any{
					"name":  "John",
					"age":   30,
					"items": []any{"a", "b", "c"},
				},
				"type": "structure",
			},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Greater(t, len(result.Content), 0)
		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Object with")
	})

	t.Run("static data access", func(t *testing.T) {
		// Test that static data is properly accessible in scripts
		risorEval := &evaluators.RisorEvaluator{
			Code:    staticDataAccessRisorScript,
			Timeout: 5 * time.Second,
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		staticData := &staticdata.StaticData{
			Data: map[string]any{
				"max_value": 500,
			},
		}

		handler := &mcpapp.ScriptToolHandler{
			Evaluator:  risorEval,
			StaticData: staticData,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		assert.Empty(t, tool.Name, "Tool name should be empty as it's set by the caller during tool registration")
		assert.Empty(t, tool.Description, "Tool description should be empty as it's set by the caller during tool registration")

		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "500")
	})

	t.Run("args access", func(t *testing.T) {
		// Test that MCP arguments are properly accessible in scripts
		risorEval := &evaluators.RisorEvaluator{
			Code:    argsAccessRisorScript,
			Timeout: 5 * time.Second,
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		handler := &mcpapp.ScriptToolHandler{
			Evaluator: risorEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		assert.Empty(t, tool.Name, "Tool name should be empty as it's set by the caller during tool registration")
		assert.Empty(t, tool.Description, "Tool description should be empty as it's set by the caller during tool registration")

		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{
				"name":  "Alice",
				"count": 42,
			},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Alice")
		assert.Contains(t, textContent.Text, "42")
	})
}

func TestScriptBridgeErrorHandling(t *testing.T) {
	ctx := t.Context()

	t.Run("script returns error object", func(t *testing.T) {
		risorEval := &evaluators.RisorEvaluator{
			Code:    errorHandlingRisorScript,
			Timeout: 5 * time.Second,
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		handler := &mcpapp.ScriptToolHandler{
			Evaluator: risorEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		assert.Empty(t, tool.Name, "Tool name should be empty as it's set by the caller during tool registration")
		assert.Empty(t, tool.Description, "Tool description should be empty as it's set by the caller during tool registration")

		// Test error case
		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{
				"error": true,
			},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		assert.NoError(t, err, "Handler should not return Go error for script runtime errors")
		assert.NotNil(t, result, "Result should be returned")
		assert.True(t, result.IsError, "Result should indicate error")

		// Verify error message is in content
		assert.Greater(t, len(result.Content), 0, "Error result should have content")
		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		assert.True(t, ok, "Error content should be text")
		assert.Contains(t, textContent.Text, "Something went wrong", "Error message should be in content")

		// Test success case
		params.Arguments.(map[string]any)["error"] = false
		req = &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err = mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("script execution timeout", func(t *testing.T) {
		risorEval := &evaluators.RisorEvaluator{
			Code: `
				// Simulate long-running operation
				for i := 0; i < 1000000; i++ {
					// Busy wait
				}
				{"text": "Done"}
			`,
			Timeout: 1 * time.Millisecond, // Very short timeout
		}

		err := risorEval.Validate()
		require.NoError(t, err)

		handler := &mcpapp.ScriptToolHandler{
			Evaluator: risorEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool, "CreateMCPTool should return a valid tool object")
		require.NotNil(t, mcpHandler, "CreateMCPTool should return a valid handler function")

		assert.Empty(t, tool.Name, "Tool name should be empty as it's set by the caller during tool registration")
		assert.Empty(t, tool.Description, "Tool description should be empty as it's set by the caller during tool registration")

		params := &mcpsdk.CallToolParams{
			Arguments: map[string]any{},
		}

		req := &mcpsdk.CallToolRequest{
			Params: params,
		}
		result, err := mcpHandler(ctx, req)
		assert.Error(t, err, "Script execution should timeout with very short timeout")
		assert.Nil(t, result, "Result should be nil when script execution times out")
		assert.Contains(t, err.Error(), "timeout", "Error message should indicate timeout occurred")
	})

	t.Run("nil evaluator", func(t *testing.T) {
		handler := &mcpapp.ScriptToolHandler{
			Evaluator: nil,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "requires an evaluator")
	})
}
