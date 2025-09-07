package mcp

import (
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
			"age":    int64(30), // JSON numbers now preserved as int64
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
			"count": int64(42), // JSON numbers now preserved as int64
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
			"count":   int64(5),
		}
		assert.Equal(t, expected, result)
	})

	// Edge cases for empty/whitespace JSON
	t.Run("empty byte slice", func(t *testing.T) {
		result, err := extractArguments([]byte{})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("whitespace only JSON", func(t *testing.T) {
		result, err := extractArguments([]byte("   \n\t   "))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("empty JSON object string", func(t *testing.T) {
		result, err := extractArguments("{}")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("null JSON value", func(t *testing.T) {
		result, err := extractArguments(json.RawMessage("null"))
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	// Non-object JSON types (should error)
	t.Run("JSON array", func(t *testing.T) {
		input := json.RawMessage(`[1, 2, 3]`)
		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "arguments must be a JSON object")
	})

	t.Run("JSON string primitive", func(t *testing.T) {
		input := json.RawMessage(`"hello world"`)
		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "arguments must be a JSON object")
	})

	t.Run("JSON number primitive", func(t *testing.T) {
		input := json.RawMessage(`42`)
		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "arguments must be a JSON object")
	})

	t.Run("JSON boolean primitive", func(t *testing.T) {
		input := json.RawMessage(`true`)
		result, err := extractArguments(input)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "arguments must be a JSON object")
	})

	// Number precision tests
	t.Run("large integer precision", func(t *testing.T) {
		// JavaScript MAX_SAFE_INTEGER is 9007199254740991
		input := json.RawMessage(`{"id": 9007199254740991, "small": 42}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.Equal(t, int64(9007199254740991), result["id"])
		assert.Equal(t, int64(42), result["small"])
	})

	t.Run("float precision", func(t *testing.T) {
		input := json.RawMessage(`{"pi": 3.14159265359, "e": 2.71828182846}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.InDelta(t, 3.14159265359, result["pi"], 0.00000000001)
		assert.InDelta(t, 2.71828182846, result["e"], 0.00000000001)
	})

	t.Run("very large number as string", func(t *testing.T) {
		// Number too large for int64 or float64
		input := json.RawMessage(`{"huge": 99999999999999999999999999999}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		// Should preserve as string when too large
		assert.IsType(t, "", result["huge"])
		assert.Equal(t, "99999999999999999999999999999", result["huge"])
	})

	// Nested and complex structures
	t.Run("deeply nested object", func(t *testing.T) {
		input := json.RawMessage(`{"a": {"b": {"c": {"d": "deep"}}}}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.NotNil(t, result["a"])
		nestedA := result["a"].(map[string]any)
		nestedB := nestedA["b"].(map[string]any)
		nestedC := nestedB["c"].(map[string]any)
		assert.Equal(t, "deep", nestedC["d"])
	})

	t.Run("mixed types in object", func(t *testing.T) {
		input := json.RawMessage(`{
			"string": "text",
			"number": 123,
			"float": 45.67,
			"bool": true,
			"null": null,
			"array": [1, 2, 3],
			"object": {"nested": "value"}
		}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.Equal(t, "text", result["string"])
		assert.Equal(t, int64(123), result["number"])
		assert.InDelta(t, 45.67, result["float"], 0.001)
		assert.Equal(t, true, result["bool"])
		assert.Nil(t, result["null"])
		assert.NotNil(t, result["array"])
		assert.NotNil(t, result["object"])

		// Check nested object
		nestedObj := result["object"].(map[string]any)
		assert.Equal(t, "value", nestedObj["nested"])
	})

	// Unicode and special characters
	t.Run("unicode in JSON", func(t *testing.T) {
		input := json.RawMessage(`{"emoji": "🎉", "chinese": "你好", "arabic": "مرحبا"}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.Equal(t, "🎉", result["emoji"])
		assert.Equal(t, "你好", result["chinese"])
		assert.Equal(t, "مرحبا", result["arabic"])
	})

	t.Run("escaped characters in JSON", func(t *testing.T) {
		input := json.RawMessage(`{"quote": "He said \"hello\"", "newline": "line1\nline2"}`)
		result, err := extractArguments(input)
		require.NoError(t, err)
		assert.Equal(t, `He said "hello"`, result["quote"])
		assert.Equal(t, "line1\nline2", result["newline"])
	})

	// Test existing case that should now return int64 instead of float64
	t.Run("json.RawMessage arguments - updated for int64", func(t *testing.T) {
		input := json.RawMessage(`{"name":"John","age":30,"active":true}`)

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"name":   "John",
			"age":    int64(30), // Now returns int64 instead of float64
			"active": true,
		}
		assert.Equal(t, expected, result)
	})

	t.Run("struct that needs marshaling - updated for int64", func(t *testing.T) {
		type testStruct struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		input := testStruct{Name: "test", Count: 42}

		result, err := extractArguments(input)
		require.NoError(t, err)

		expected := map[string]any{
			"name":  "test",
			"count": int64(42), // Now returns int64 instead of float64
		}
		assert.Equal(t, expected, result)
	})
}

func TestBuiltinToolHandlerExecution(t *testing.T) {
	ctx := t.Context()

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
		args, err := json.Marshal(map[string]any{
			"message": "Hello, World!",
		})
		require.NoError(t, err)
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
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
		args, err := json.Marshal(map[string]any{
			"expression": "2 + 2",
		})
		require.NoError(t, err)
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
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
		args, err := json.Marshal(map[string]any{})
		require.NoError(t, err)
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
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
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage([]byte("invalid json")), // This will cause extractArguments to fail
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
		args, err := json.Marshal(map[string]any{
			"path": "test.txt",
		})
		require.NoError(t, err)
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
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
		args, err := json.Marshal(map[string]any{})
		require.NoError(t, err)
		req := &mcpsdk.CallToolRequest{
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
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
			Params: &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage([]byte("invalid json")), // This will cause extractArguments to fail
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

func TestConvertJSONNumber(t *testing.T) {
	t.Run("small integer to int64", func(t *testing.T) {
		num := json.Number("42")
		result := convertJSONNumber(num)
		assert.Equal(t, int64(42), result)
	})

	t.Run("large safe integer to int64", func(t *testing.T) {
		num := json.Number("9007199254740991") // JavaScript MAX_SAFE_INTEGER
		result := convertJSONNumber(num)
		assert.Equal(t, int64(9007199254740991), result)
	})

	t.Run("float to float64", func(t *testing.T) {
		num := json.Number("3.14159")
		result := convertJSONNumber(num)
		assert.InEpsilon(t, 3.14159, result, 0.0001)
	})

	t.Run("very large integer to string", func(t *testing.T) {
		num := json.Number("99999999999999999999999999999")
		result := convertJSONNumber(num)
		assert.Equal(t, "99999999999999999999999999999", result)
	})

	t.Run("scientific notation to string", func(t *testing.T) {
		num := json.Number("1.23456789012345678901234567890e+50")
		result := convertJSONNumber(num)
		assert.Equal(t, "1.23456789012345678901234567890e+50", result)
	})

	t.Run("negative integer to int64", func(t *testing.T) {
		num := json.Number("-123")
		result := convertJSONNumber(num)
		assert.Equal(t, int64(-123), result)
	})

	t.Run("zero to int64", func(t *testing.T) {
		num := json.Number("0")
		result := convertJSONNumber(num)
		assert.Equal(t, int64(0), result)
	})
}

func TestConvertJSONNumbers(t *testing.T) {
	t.Run("simple map with numbers", func(t *testing.T) {
		input := map[string]any{
			"int":   json.Number("42"),
			"float": json.Number("3.14"),
			"big":   json.Number("99999999999999999999999999999"),
			"text":  "hello",
		}

		convertJSONNumbers(input)

		assert.Equal(t, int64(42), input["int"])
		assert.InEpsilon(t, 3.14, input["float"], 0.0001)
		assert.Equal(t, "99999999999999999999999999999", input["big"])
		assert.Equal(t, "hello", input["text"])
	})

	t.Run("nested map with numbers", func(t *testing.T) {
		input := map[string]any{
			"outer": map[string]any{
				"inner": map[string]any{
					"number": json.Number("123"),
				},
			},
		}

		convertJSONNumbers(input)

		outerMap := input["outer"].(map[string]any)
		innerMap := outerMap["inner"].(map[string]any)
		assert.Equal(t, int64(123), innerMap["number"])
	})

	t.Run("array with numbers", func(t *testing.T) {
		input := map[string]any{
			"numbers": []any{
				json.Number("1"),
				json.Number("2.5"),
				json.Number("99999999999999999999999999999"),
				"text",
			},
		}

		convertJSONNumbers(input)

		numbers := input["numbers"].([]any)
		assert.Equal(t, int64(1), numbers[0])
		assert.InEpsilon(t, 2.5, numbers[1], 0.0001)
		assert.Equal(t, "99999999999999999999999999999", numbers[2])
		assert.Equal(t, "text", numbers[3])
	})

	t.Run("nested arrays", func(t *testing.T) {
		input := []any{
			[]any{
				json.Number("42"),
				json.Number("3.14"),
			},
		}

		convertJSONNumbers(input)

		outerArray := input[0].([]any)
		assert.Equal(t, int64(42), outerArray[0])
		assert.InEpsilon(t, 3.14, outerArray[1], 0.0001)
	})

	t.Run("no numbers to convert", func(t *testing.T) {
		input := map[string]any{
			"text":  "hello",
			"bool":  true,
			"null":  nil,
			"array": []any{"a", "b", "c"},
		}
		original := map[string]any{
			"text":  "hello",
			"bool":  true,
			"null":  nil,
			"array": []any{"a", "b", "c"},
		}

		convertJSONNumbers(input)

		// Should be unchanged
		assert.Equal(t, original, input)
	})
}
