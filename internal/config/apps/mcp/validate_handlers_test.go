package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/go-polyscript/engines/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArguments(t *testing.T) {
	t.Run("nil arguments", func(t *testing.T) {
		result, err := extractArguments(nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("map[string]any arguments", func(t *testing.T) {
		input := map[string]any{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		}

		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("json.RawMessage arguments", func(t *testing.T) {
		input := json.RawMessage(`{"name":"John","age":30,"active":true}`)

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"name":   "John",
			"age":    float64(30), // JSON numbers become float64
			"active": true,
		}
		assert.Equal(t, expected, result)
	})

	t.Run("[]byte arguments", func(t *testing.T) {
		input := []byte(`{"expression":"2+2","type":"math"}`)

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"expression": "2+2",
			"type":       "math",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("struct that needs marshaling", func(t *testing.T) {
		type testStruct struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		input := testStruct{Name: "test", Count: 42}

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"name":  "test",
			"count": float64(42), // JSON numbers become float64
		}
		assert.Equal(t, expected, result)
	})

	t.Run("invalid JSON in RawMessage", func(t *testing.T) {
		input := json.RawMessage(`{"invalid":}`)

		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to unmarshal arguments from JSON")
	})

	t.Run("invalid JSON in byte slice", func(t *testing.T) {
		input := []byte(`{"broken": json}`)

		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to unmarshal arguments from JSON")
	})

	t.Run("unmarshalable object", func(t *testing.T) {
		// Create a value that can't be marshaled to JSON
		input := make(chan int)

		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to marshal arguments to JSON")
	})

	t.Run("string that parses as JSON", func(t *testing.T) {
		input := `{"message":"hello","count":5}`

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"message": "hello",
			"count":   float64(5),
		}
		assert.Equal(t, expected, result)
	})
}

func TestBuiltinToolHandlerExecution(t *testing.T) {
	ctx := context.Background()

	t.Run("echo handler execution", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		// Test handler execution
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: map[string]any{
					"message": "Hello, World!",
				},
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Hello, World!")
	})

	t.Run("calculation handler execution - valid", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		// Test handler execution with valid expression
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: map[string]any{
					"expression": "2 + 2",
				},
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Content, 1)
		assert.False(t, result.IsError)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "2 + 2")
		assert.Contains(t, textContent.Text, "not implemented")
	})

	t.Run("calculation handler execution - missing expression", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		_, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)

		// Test handler execution without expression
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: map[string]any{},
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "expression parameter required")
	})

	t.Run("calculation handler execution - invalid arguments", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		_, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)

		// Test handler execution with invalid arguments
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: make(chan int), // This will cause extractArguments to fail
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Error extracting arguments")
	})

	t.Run("file read handler execution - valid", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/tmp",
			},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		// Test handler execution with valid path
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: map[string]any{
					"path": "test.txt",
				},
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Content, 1)
		assert.False(t, result.IsError)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "/tmp/test.txt")
		assert.Contains(t, textContent.Text, "not implemented")
	})

	t.Run("file read handler execution - missing path", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/tmp",
			},
		}

		_, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)

		// Test handler execution without path
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: map[string]any{},
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "path parameter required")
	})

	t.Run("file read handler execution - invalid arguments", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/workspace",
			},
		}

		_, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)

		// Test handler execution with invalid arguments
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParams{
				Arguments: make(chan int), // This will cause extractArguments to fail
			},
		}

		result, err := mcpHandler(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Error extracting arguments")
	})
}

func TestScriptToolHandlerCreateMCPTool(t *testing.T) {
	t.Run("successful script tool creation", func(t *testing.T) {
		mockPlatformEval := &mocks.Evaluator{}
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: mockPlatformEval,
		}

		mockEval.On("GetCompiledEvaluator").Return(mockPlatformEval, nil)

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
			StaticData: &staticdata.StaticData{
				Data: map[string]any{
					"version": "1.0.0",
				},
			},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		require.NotNil(t, tool)
		require.NotNil(t, mcpHandler)

		// Verify the tool was created correctly
		assert.Empty(t, tool.Name)        // Will be set by caller
		assert.Empty(t, tool.Description) // Will be set by caller

		mockEval.AssertExpectations(t)
	})

	t.Run("evaluator compilation error", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &mocks.Evaluator{},
		}
		mockEval.On("GetCompiledEvaluator").Return(nil, assert.AnError)

		handler := &ScriptToolHandler{
			Evaluator:  mockEval,
			StaticData: &staticdata.StaticData{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "failed to get compiled evaluator")

		mockEval.AssertExpectations(t)
	})

	t.Run("nil compiled evaluator", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &mocks.Evaluator{},
		}
		mockEval.On("GetCompiledEvaluator").Return(nil, nil)

		handler := &ScriptToolHandler{
			Evaluator:  mockEval,
			StaticData: &staticdata.StaticData{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "compiled evaluator is nil")

		mockEval.AssertExpectations(t)
	})
}
