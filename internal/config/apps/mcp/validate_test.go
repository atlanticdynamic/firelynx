package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	evalMocks "github.com/robbyt/go-polyscript/engines/mocks"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEvaluatorAdapter adapts evalMocks.Evaluator to implement evaluators.Evaluator interface
type mockEvaluatorAdapter struct {
	mock.Mock
	PlatformEvaluator *evalMocks.Evaluator
}

func (m *mockEvaluatorAdapter) Type() evaluators.EvaluatorType {
	args := m.Called()
	return args.Get(0).(evaluators.EvaluatorType)
}

func (m *mockEvaluatorAdapter) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockEvaluatorAdapter) GetCompiledEvaluator() (platform.Evaluator, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(platform.Evaluator), args.Error(1)
}

func (m *mockEvaluatorAdapter) GetTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// mockDataProvider mocks the data provider interface for script tool tests
type mockDataProvider struct {
	mock.Mock
}

func (m *mockDataProvider) GetData(ctx context.Context) (map[string]any, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *mockDataProvider) AddDataToContext(ctx context.Context, d ...map[string]any) (context.Context, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(context.Context), args.Error(1)
}

// mockToolHandler for testing unknown handler types
type mockToolHandler struct{}

func (m *mockToolHandler) CreateMCPTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
	return nil, nil, nil
}

func (m *mockToolHandler) Validate() error {
	return nil
}

func (m *mockToolHandler) Type() string {
	return "mock"
}

func TestApp_Validate(t *testing.T) {
	t.Run("valid minimal app", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer, "compiled server should be created")
	})

	t.Run("missing server name", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerVersion = "1.0.0"

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingServerName)
	})

	t.Run("missing server version", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingServerVersion)
	})

	t.Run("invalid transport", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Transport = &Transport{
			SSEEnabled: true,
			SSEPath:    "", // Missing path when SSE enabled
		}

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidTransport)
	})

	t.Run("invalid tool", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name: "", // Missing name
			},
		}

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidTool)
	})

	t.Run("invalid middleware", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Middlewares = []*Middleware{
			{
				Type: 999, // Invalid type
			},
		}

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidMiddleware)
	})

	t.Run("valid app with tools", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "echo",
				Description: "Echo tool",
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})

	t.Run("duplicate tool names", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "duplicate",
				Description: "First tool",
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
			{
				Name:        "duplicate",
				Description: "Second tool",
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinCalculation,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicateToolName)
	})

	t.Run("duplicate prompt names", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Prompts = []*Prompt{
			{
				Name:        "duplicate",
				Description: "First prompt",
			},
			{
				Name:        "duplicate",
				Description: "Second prompt",
			},
		}

		err := app.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicatePromptName)
	})

	t.Run("valid app with prompts", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Prompts = []*Prompt{
			{
				Name:        "test_prompt",
				Description: "Test prompt",
				Arguments: []*PromptArgument{
					{
						Name:        "input",
						Description: "Input parameter",
						Required:    true,
					},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})
}

func TestTransport_Validate(t *testing.T) {
	t.Run("valid transport with SSE disabled", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: false,
			SSEPath:    "",
		}

		err := transport.Validate()
		require.NoError(t, err)
	})

	t.Run("SSE enabled should fail validation", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: true,
			SSEPath:    "/events",
		}

		err := transport.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE transport is not yet implemented for MCP apps")
	})

	t.Run("SSE enabled but missing path should fail validation", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: true,
			SSEPath:    "",
		}

		err := transport.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE transport is not yet implemented for MCP apps")
	})
}

func TestTool_Validate(t *testing.T) {
	t.Run("valid tool with builtin handler", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
				Config:      map[string]string{},
			},
		}

		err := tool.Validate()
		require.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		tool := &Tool{
			Description: "Echo tool",
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingToolName)
	})

	t.Run("description is optional", func(t *testing.T) {
		tool := &Tool{
			Name: "echo",
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.NoError(t, err, "description should be optional, not required")
	})

	t.Run("missing handler", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			Handler:     nil,
		}

		err := tool.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingToolHandler)
	})

	t.Run("valid input schema", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			InputSchema: `{"type": "object", "properties": {"message": {"type": "string"}}}`,
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid input schema - bad JSON", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			InputSchema: `{"type": "object", "properties": {"message": {"type": "string"}`,
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidJSONSchema)
	})

	t.Run("invalid input schema - invalid type", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			InputSchema: `{"type": "invalid_type"}`,
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidJSONSchema)
	})

	t.Run("valid output schema", func(t *testing.T) {
		tool := &Tool{
			Name:         "echo",
			Description:  "Echo tool",
			OutputSchema: `{"type": "string"}`,
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid output schema", func(t *testing.T) {
		tool := &Tool{
			Name:         "echo",
			Description:  "Echo tool",
			OutputSchema: `{"type": "invalid"}`,
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidJSONSchema)
	})
}

func TestScriptToolHandler_Validate(t *testing.T) {
	t.Run("missing evaluator", func(t *testing.T) {
		handler := &ScriptToolHandler{
			StaticData: nil,
			Evaluator:  nil,
		}

		err := handler.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingEvaluator)
	})

	t.Run("valid with static data", func(t *testing.T) {
		validStaticData := &staticdata.StaticData{
			Data: map[string]any{
				"key": "value",
			},
		}

		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &evalMocks.Evaluator{},
		}
		mockEval.On("Validate").Return(nil)

		handler := &ScriptToolHandler{
			StaticData: validStaticData,
			Evaluator:  mockEval,
		}

		err := handler.Validate()
		// Mock evaluator returns no error, so validation should succeed
		require.NoError(t, err)
	})
}

func TestBuiltinToolHandler_Validate(t *testing.T) {
	t.Run("valid echo handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		require.NoError(t, err)
	})

	t.Run("valid calculation handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		require.NoError(t, err)
	})

	t.Run("valid file read handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/workspace",
			},
		}

		err := handler.Validate()
		require.NoError(t, err)
	})

	t.Run("file read handler missing base directory", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingBaseDirectory)
	})

	t.Run("unknown builtin type", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinType(999),
			Config:      map[string]string{},
		}

		err := handler.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrUnknownBuiltinType)
	})
}

func TestMiddleware_Validate(t *testing.T) {
	t.Run("valid rate limiting middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareRateLimiting,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		require.NoError(t, err)
	})

	t.Run("valid logging middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareLogging,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		require.NoError(t, err)
	})

	t.Run("valid authentication middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareAuthentication,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		require.NoError(t, err)
	})

	t.Run("unknown middleware type", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareType(999),
			Config: map[string]string{},
		}

		err := middleware.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrUnknownMiddlewareType)
	})
}

func TestBuiltinToolHandler_CreateMCPTool(t *testing.T) {
	t.Run("create echo tool", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		assert.NotNil(t, tool)
		assert.NotNil(t, mcpHandler)
		assert.Empty(t, tool.Name)        // Will be set by caller
		assert.Empty(t, tool.Description) // Will be set by caller
	})

	t.Run("create calculation tool", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		assert.NotNil(t, tool)
		assert.NotNil(t, mcpHandler)
	})

	t.Run("create file read tool", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/workspace",
			},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.NoError(t, err)
		assert.NotNil(t, tool)
		assert.NotNil(t, mcpHandler)
	})

	t.Run("unknown builtin type", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinType(999),
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		require.ErrorIs(t, err, ErrUnknownBuiltinType)
	})
}

func TestScriptToolHandler_CreateMCPTool(t *testing.T) {
	t.Run("script tool with mock evaluator error", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &evalMocks.Evaluator{},
		}
		mockEval.On("GetCompiledEvaluator").Return(nil, nil)

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "compiled evaluator is nil")
	})

	t.Run("script tool with nil evaluator", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: nil,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "script tool handler requires an evaluator")
	})
}

func TestPrompt_Validate(t *testing.T) {
	t.Run("valid prompt", func(t *testing.T) {
		prompt := &Prompt{
			Name:        "test_prompt",
			Description: "Test prompt",
			Title:       "Test Prompt",
			Arguments: []*PromptArgument{
				{
					Name:        "input",
					Description: "Input parameter",
					Required:    true,
				},
			},
		}

		err := prompt.Validate()
		require.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		prompt := &Prompt{
			Description: "Test prompt",
		}

		err := prompt.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingPromptName)
	})

	t.Run("duplicate argument names", func(t *testing.T) {
		prompt := &Prompt{
			Name:        "test_prompt",
			Description: "Test prompt",
			Arguments: []*PromptArgument{
				{
					Name:        "duplicate",
					Description: "First argument",
				},
				{
					Name:        "duplicate",
					Description: "Second argument",
				},
			},
		}

		err := prompt.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicatePromptArgName)
	})

	t.Run("invalid argument", func(t *testing.T) {
		prompt := &Prompt{
			Name:        "test_prompt",
			Description: "Test prompt",
			Arguments: []*PromptArgument{
				{
					Name: "", // Missing name
				},
			},
		}

		err := prompt.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "argument 0")
		assert.Contains(t, err.Error(), "prompt argument name is required")
	})
}

func TestPromptArgument_Validate(t *testing.T) {
	t.Run("valid argument", func(t *testing.T) {
		arg := &PromptArgument{
			Name:        "input",
			Title:       "Input",
			Description: "Input parameter",
			Required:    true,
		}

		err := arg.Validate()
		require.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		arg := &PromptArgument{
			Description: "Input parameter",
		}

		err := arg.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingPromptArgumentName)
	})

	t.Run("name only is valid", func(t *testing.T) {
		arg := &PromptArgument{
			Name: "input",
		}

		err := arg.Validate()
		require.NoError(t, err)
	})
}

func TestParseJSONSchema(t *testing.T) {
	t.Parallel()

	t.Run("valid schema with all fields", func(t *testing.T) {
		schemaString := `{
			"type": "object",
			"description": "Test schema",
			"properties": {
				"name": {
					"type": "string",
					"description": "Name field"
				},
				"age": {
					"type": "number",
					"description": "Age field"
				}
			},
			"required": ["name"]
		}`

		schema, err := parseJSONSchema(schemaString)
		require.NoError(t, err)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "Test schema", schema.Description)
		assert.Len(t, schema.Properties, 2)
		assert.Equal(t, "string", schema.Properties["name"].Type)
		assert.Equal(t, "Name field", schema.Properties["name"].Description)
		assert.Equal(t, "number", schema.Properties["age"].Type)
		assert.Equal(t, []string{"name"}, schema.Required)
	})

	t.Run("minimal valid schema", func(t *testing.T) {
		schemaString := `{"type": "string"}`

		schema, err := parseJSONSchema(schemaString)
		require.NoError(t, err)
		assert.Equal(t, "string", schema.Type)
		assert.Empty(t, schema.Description)
		assert.Nil(t, schema.Properties)
		assert.Nil(t, schema.Required)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		schemaString := `{"type": "string"`

		schema, err := parseJSONSchema(schemaString)
		require.Error(t, err)
		assert.Nil(t, schema)
		assert.Contains(t, err.Error(), "failed to parse JSON schema")
	})

	t.Run("schema with invalid property structure", func(t *testing.T) {
		schemaString := `{
			"type": "object",
			"properties": {
				"invalid": "not an object"
			}
		}`

		schema, err := parseJSONSchema(schemaString)
		require.NoError(t, err)
		assert.Equal(t, "object", schema.Type)
		assert.Empty(t, schema.Properties)
	})
}

func TestConvertAnnotationsToMCPSDK(t *testing.T) {
	t.Parallel()

	t.Run("nil annotations", func(t *testing.T) {
		result := convertAnnotationsToMCPSDK(nil)
		assert.Nil(t, result)
	})

	t.Run("complete annotations", func(t *testing.T) {
		destructive := true
		openWorld := false
		annotations := &ToolAnnotations{
			Title:           "Test Tool",
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &destructive,
			OpenWorldHint:   &openWorld,
		}

		result := convertAnnotationsToMCPSDK(annotations)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Tool", result.Title)
		assert.True(t, result.ReadOnlyHint)
		assert.True(t, result.IdempotentHint)
		assert.Equal(t, &destructive, result.DestructiveHint)
		assert.Equal(t, &openWorld, result.OpenWorldHint)
	})

	t.Run("minimal annotations", func(t *testing.T) {
		annotations := &ToolAnnotations{
			ReadOnlyHint:   false,
			IdempotentHint: false,
		}

		result := convertAnnotationsToMCPSDK(annotations)
		assert.NotNil(t, result)
		assert.Empty(t, result.Title)
		assert.False(t, result.ReadOnlyHint)
		assert.False(t, result.IdempotentHint)
		assert.Nil(t, result.DestructiveHint)
		assert.Nil(t, result.OpenWorldHint)
	})
}

func TestScriptToolHandler_convertToMCPContent(t *testing.T) {
	handler := &ScriptToolHandler{}

	t.Run("map with error field", func(t *testing.T) {
		mockResult := &evalMocks.EvaluatorResponse{}
		mockResult.On("Interface").Return(map[string]any{
			"error": "test error message",
		})

		result, err := handler.convertToMCPContent(mockResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.JSONEq(t, `{"error":"test error message"}`, result.Content[0].(*mcpsdk.TextContent).Text)
		mockResult.AssertExpectations(t)
	})

	t.Run("map without error field", func(t *testing.T) {
		mockResult := &evalMocks.EvaluatorResponse{}
		mockResult.On("Interface").Return(map[string]any{
			"status": "success",
			"data":   "test data",
		})

		result, err := handler.convertToMCPContent(mockResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Contains(t, result.Content[0].(*mcpsdk.TextContent).Text, "success")
		assert.Contains(t, result.Content[0].(*mcpsdk.TextContent).Text, "test data")
		mockResult.AssertExpectations(t)
	})

	t.Run("string value", func(t *testing.T) {
		mockResult := &evalMocks.EvaluatorResponse{}
		mockResult.On("Interface").Return("test string result")

		result, err := handler.convertToMCPContent(mockResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "test string result", result.Content[0].(*mcpsdk.TextContent).Text)
		mockResult.AssertExpectations(t)
	})

	t.Run("byte slice value", func(t *testing.T) {
		mockResult := &evalMocks.EvaluatorResponse{}
		mockResult.On("Interface").Return([]byte("test bytes"))

		result, err := handler.convertToMCPContent(mockResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "test bytes", result.Content[0].(*mcpsdk.TextContent).Text)
		mockResult.AssertExpectations(t)
	})

	t.Run("other type value", func(t *testing.T) {
		mockResult := &evalMocks.EvaluatorResponse{}
		mockResult.On("Interface").Return(42)

		result, err := handler.convertToMCPContent(mockResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "42", result.Content[0].(*mcpsdk.TextContent).Text)
		mockResult.AssertExpectations(t)
	})
}

func TestScriptToolHandler_prepareScriptContext(t *testing.T) {
	t.Parallel()

	t.Run("with static data", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &evalMocks.Evaluator{},
		}

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any{
			"config": "test",
		}, nil)

		arguments := map[string]any{
			"input": "value",
		}

		scriptContext, err := handler.prepareScriptContext(t.Context(), provider, arguments)
		require.NoError(t, err)

		// Script execution context should have static data under "data" namespace
		dataSection, ok := scriptContext["data"].(map[string]any)
		require.True(t, ok, "script context should have 'data' section for static data access")
		assert.Equal(t, "test", dataSection["config"], "static data should be accessible in script context")

		// Script execution context should have tool arguments under "args" namespace
		assert.Equal(t, arguments, scriptContext["args"], "tool arguments should be accessible in script context")

		mockEval.AssertExpectations(t)
		provider.AssertExpectations(t)
	})

	t.Run("empty static data", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &evalMocks.Evaluator{},
		}

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any{}, nil)

		arguments := map[string]any{
			"input": "value",
		}

		scriptContext, err := handler.prepareScriptContext(t.Context(), provider, arguments)
		require.NoError(t, err)

		// Script execution context should have empty static data under "data" namespace
		dataSection, ok := scriptContext["data"].(map[string]any)
		require.True(t, ok, "script context should have 'data' section for static data access")
		assert.Empty(t, dataSection, "data section should be empty when no static data provided")

		// Script execution context should have tool arguments under "args" namespace
		assert.Equal(t, arguments, scriptContext["args"], "tool arguments should be accessible in script context")

		mockEval.AssertExpectations(t)
		provider.AssertExpectations(t)
	})

	t.Run("provider error", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{
			PlatformEvaluator: &evalMocks.Evaluator{},
		}

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any(nil), assert.AnError)

		scriptContext, err := handler.prepareScriptContext(t.Context(), provider, nil)
		require.Error(t, err)
		assert.Nil(t, scriptContext)
		assert.Contains(t, err.Error(), "failed to get tool static data")

		mockEval.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestScriptToolHandler_executeScriptTool(t *testing.T) {
	t.Run("successful execution with risor", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: &evaluators.RisorEvaluator{
				Code: `func process() {
					args := ctx.get("args", {})
					input := args.get("input", "")
					return {"result": "success", "input": input}
				}
				process()`,
				Timeout: 5 * time.Second,
			},
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any{
			"config": "test",
		}, nil)

		arguments := map[string]any{
			"input": "test value",
		}

		eval, err := handler.Evaluator.GetCompiledEvaluator()
		require.NoError(t, err)

		result, err := handler.executeScriptTool(t.Context(), eval, provider, arguments)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		provider.AssertExpectations(t)
	})

	t.Run("script execution timeout", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: &evaluators.RisorEvaluator{
				Code: `func process() {
					// Simulate long-running operation
					for i := 0; i < 1000000; i++ {
						// Busy wait to consume time
					}
					return "should timeout"
				}
				process()`,
				Timeout: 1 * time.Millisecond, // Very short timeout
			},
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any{}, nil)

		eval, err := handler.Evaluator.GetCompiledEvaluator()
		require.NoError(t, err)

		result, err := handler.executeScriptTool(t.Context(), eval, provider, map[string]any{})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "timeout")

		provider.AssertExpectations(t)
	})

	t.Run("script compilation error", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: &evaluators.RisorEvaluator{
				Code:    `undefined_function()`, // This should cause a compilation error
				Timeout: 5 * time.Second,
			},
		}

		eval, err := handler.Evaluator.GetCompiledEvaluator()
		require.Error(t, err)
		assert.Nil(t, eval)
		assert.Contains(t, err.Error(), "undefined variable \"undefined_function\"")
	})

	t.Run("static data provider error", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: &evaluators.RisorEvaluator{
				Code: `func process() {
					return {"result": "success"}
				}
				process()`,
				Timeout: 5 * time.Second,
			},
		}

		provider := &mockDataProvider{}
		provider.On("GetData", mock.Anything).Return(map[string]any(nil), assert.AnError)

		eval, err := handler.Evaluator.GetCompiledEvaluator()
		require.NoError(t, err)

		result, err := handler.executeScriptTool(t.Context(), eval, provider, map[string]any{})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to prepare script context")

		provider.AssertExpectations(t)
	})
}

// Additional tests for compileMCPServer edge cases to improve coverage
func TestApp_ValidateCompileMCPServerEdgeCases(t *testing.T) {
	t.Run("tool with Title field", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "test-tool",
				Description: "Test tool",
				Title:       "Custom Title",
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})

	t.Run("tool with custom InputSchema", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "test-tool",
				Description: "Test tool",
				InputSchema: `{"type":"object","properties":{"input":{"type":"string"}},"required":["input"]}`,
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})

	t.Run("tool with custom OutputSchema", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:         "test-tool",
				Description:  "Test tool",
				OutputSchema: `{"type":"object","properties":{"result":{"type":"string"}}}`,
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})

	t.Run("tool with Annotations", func(t *testing.T) {
		app := NewApp("test-id")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "test-tool",
				Description: "Test tool",
				Annotations: &ToolAnnotations{
					Title:           "Annotated Tool",
					ReadOnlyHint:    true,
					IdempotentHint:  true,
					DestructiveHint: &[]bool{false}[0],
					OpenWorldHint:   &[]bool{true}[0],
				},
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.NoError(t, err)
		assert.NotNil(t, app.compiledServer)
	})

	t.Run("invalid custom InputSchema", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "test-tool",
				Description: "Test tool",
				InputSchema: `{"type":"invalid-type"}`, // Invalid JSON schema
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON Schema type: invalid-type")
	})

	t.Run("invalid custom OutputSchema", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:         "test-tool",
				Description:  "Test tool",
				OutputSchema: `{"type":"invalid-type"}`, // Invalid JSON schema
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON Schema type: invalid-type")
	})

	t.Run("unparseable custom InputSchema JSON", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:        "test-tool",
				Description: "Test tool",
				InputSchema: `{"type":"object",}`, // Invalid JSON
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON: invalid character")
	})

	t.Run("unparseable custom OutputSchema JSON", func(t *testing.T) {
		app := NewApp("")
		app.ServerName = "Test Server"
		app.ServerVersion = "1.0.0"
		app.Tools = []*Tool{
			{
				Name:         "test-tool",
				Description:  "Test tool",
				OutputSchema: `{"type":"object",}`, // Invalid JSON
				Handler: &BuiltinToolHandler{
					BuiltinType: BuiltinEcho,
					Config:      map[string]string{},
				},
			},
		}

		err := app.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON: invalid character")
	})
}

// Test validateJSONSchema edge cases
func TestValidateJSONSchemaEdgeCases(t *testing.T) {
	t.Run("invalid JSON Schema type", func(t *testing.T) {
		schema := `{"type":"invalid-type"}`
		err := validateJSONSchema(schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON Schema type")
	})

	t.Run("non-string type field", func(t *testing.T) {
		schema := `{"type":123}`
		err := validateJSONSchema(schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JSON Schema 'type' must be a string")
	})

	t.Run("invalid properties structure", func(t *testing.T) {
		schema := `{"type":"object","properties":"not-an-object"}`
		err := validateJSONSchema(schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JSON Schema 'properties' must be an object")
	})

	t.Run("invalid required array", func(t *testing.T) {
		schema := `{"type":"object","required":"not-an-array"}`
		err := validateJSONSchema(schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JSON Schema 'required' must be an array")
	})

	t.Run("invalid required array element", func(t *testing.T) {
		schema := `{"type":"object","required":[123]}`
		err := validateJSONSchema(schema)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JSON Schema 'required' array element 0 must be a string")
	})
}

// Test getDefaultInputSchema edge cases
func TestGetDefaultInputSchemaEdgeCases(t *testing.T) {
	t.Run("builtin calculation handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
		}
		schema, err := getDefaultInputSchema(handler)
		require.NoError(t, err)
		assert.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "string", schema.Properties["expression"].Type)
		assert.Equal(t, "Mathematical expression to evaluate", schema.Properties["expression"].Description)
		assert.Contains(t, schema.Required, "expression")
	})

	t.Run("builtin file read handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
		}
		schema, err := getDefaultInputSchema(handler)
		require.NoError(t, err)
		assert.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "string", schema.Properties["path"].Type)
		assert.Equal(t, "Path to the file to read", schema.Properties["path"].Description)
		assert.Contains(t, schema.Required, "path")
	})

	t.Run("script tool handler", func(t *testing.T) {
		handler := &ScriptToolHandler{}
		schema, err := getDefaultInputSchema(handler)
		require.NoError(t, err)
		assert.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "Tool input parameters", schema.Description)
	})

	t.Run("unknown handler type", func(t *testing.T) {
		// Create a mock handler that doesn't match any known types
		handler := &mockToolHandler{}
		schema, err := getDefaultInputSchema(handler)
		require.NoError(t, err)
		assert.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "Tool input parameters", schema.Description)
	})
}

func TestTransportValidateInterpolationError(t *testing.T) {
	t.Run("interpolation error case", func(t *testing.T) {
		// Create a transport with a missing environment variable
		// This will cause interpolation.InterpolateStruct to fail
		transport := &Transport{
			SSEPath: "${NONEXISTENT_MCP_VAR}", // Non-existent env var
		}

		err := transport.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transport interpolation failed")
	})
}

func TestScriptToolHandlerCreateMCPToolErrors(t *testing.T) {
	t.Run("GetCompiledEvaluator returns error", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{}
		mockEval.On("GetCompiledEvaluator").Return(nil, assert.AnError)

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "failed to get compiled evaluator")
		mockEval.AssertExpectations(t)
	})

	t.Run("GetCompiledEvaluator returns nil", func(t *testing.T) {
		mockEval := &mockEvaluatorAdapter{}
		mockEval.On("GetCompiledEvaluator").Return(nil, nil)

		handler := &ScriptToolHandler{
			Evaluator: mockEval,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "compiled evaluator is nil")
		mockEval.AssertExpectations(t)
	})
}

func TestPromptValidateErrors(t *testing.T) {
	t.Run("interpolation error", func(t *testing.T) {
		prompt := &Prompt{
			Name:        "test",
			Description: "${NONEXISTENT_PROMPT_VAR}", // Non-existent env var
			Arguments:   []*PromptArgument{{Name: "test", Description: "Test arg"}},
		}

		err := prompt.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interpolation failed")
	})
}
