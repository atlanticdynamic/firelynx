package mcp

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/stretchr/testify/assert"
)

func TestApp_Validate(t *testing.T) {
	t.Run("valid minimal app", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
			Tools:         []*Tool{},
			Resources:     []*Resource{},
			Prompts:       []*Prompt{},
			Middlewares:   []*Middleware{},
		}

		err := app.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, app.compiledServer, "compiled server should be created")
	})

	t.Run("missing server name", func(t *testing.T) {
		app := &App{
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
		}

		err := app.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server name is required")
	})

	t.Run("missing server version", func(t *testing.T) {
		app := &App{
			ServerName: "Test Server",
			Transport:  &Transport{},
		}

		err := app.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server version is required")
	})

	t.Run("invalid transport", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport: &Transport{
				SSEEnabled: true,
				SSEPath:    "", // Missing path when SSE enabled
			},
		}

		err := app.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transport configuration")
	})

	t.Run("invalid tool", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
			Tools: []*Tool{
				{
					Name: "", // Missing name
				},
			},
		}

		err := app.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tool configuration")
	})

	t.Run("invalid middleware", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
			Middlewares: []*Middleware{
				{
					Type: 999, // Invalid type
				},
			},
		}

		err := app.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid middleware configuration")
	})

	t.Run("valid app with tools", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
			Tools: []*Tool{
				{
					Name:        "echo",
					Description: "Echo tool",
					Handler: &BuiltinToolHandler{
						BuiltinType: BuiltinEcho,
						Config:      map[string]string{},
					},
				},
			},
		}

		err := app.Validate()
		assert.NoError(t, err)
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
		assert.NoError(t, err)
	})

	t.Run("valid transport with SSE enabled", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: true,
			SSEPath:    "/events",
		}

		err := transport.Validate()
		assert.NoError(t, err)
	})

	t.Run("SSE enabled but missing path", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: true,
			SSEPath:    "",
		}

		err := transport.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sse_path is required when SSE is enabled")
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
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		tool := &Tool{
			Description: "Echo tool",
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool name is required")
	})

	t.Run("missing description", func(t *testing.T) {
		tool := &Tool{
			Name: "echo",
			Handler: &BuiltinToolHandler{
				BuiltinType: BuiltinEcho,
			},
		}

		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool description is required")
	})

	t.Run("missing handler", func(t *testing.T) {
		tool := &Tool{
			Name:        "echo",
			Description: "Echo tool",
			Handler:     nil,
		}

		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool handler is required")
	})
}

func TestScriptToolHandler_Validate(t *testing.T) {
	t.Run("missing evaluator", func(t *testing.T) {
		handler := &ScriptToolHandler{
			StaticData: nil,
			Evaluator:  nil,
		}

		err := handler.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "evaluator is required")
	})

	t.Run("valid with static data", func(t *testing.T) {
		validStaticData := &staticdata.StaticData{
			Data: map[string]any{
				"key": "value",
			},
		}

		handler := &ScriptToolHandler{
			StaticData: validStaticData,
			Evaluator:  &mockEvaluator{}, // We need a mock since the interface isn't implemented
		}

		err := handler.Validate()
		// Mock evaluator returns no error, so validation should succeed
		assert.NoError(t, err)
	})
}

func TestBuiltinToolHandler_Validate(t *testing.T) {
	t.Run("valid echo handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid calculation handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid file read handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config: map[string]string{
				"base_directory": "/workspace",
			},
		}

		err := handler.Validate()
		assert.NoError(t, err)
	})

	t.Run("file read handler missing base directory", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config:      map[string]string{},
		}

		err := handler.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base_directory is required")
	})

	t.Run("unknown builtin type", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinType(999),
			Config:      map[string]string{},
		}

		err := handler.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown builtin tool type")
	})
}

func TestMiddleware_Validate(t *testing.T) {
	t.Run("valid rate limiting middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareRateLimiting,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid logging middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareLogging,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid authentication middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareAuthentication,
			Config: map[string]string{},
		}

		err := middleware.Validate()
		assert.NoError(t, err)
	})

	t.Run("unknown middleware type", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareType(999),
			Config: map[string]string{},
		}

		err := middleware.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown middleware type")
	})
}

func TestBuiltinToolHandler_CreateMCPTool(t *testing.T) {
	t.Run("create echo tool", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.NotNil(t, mcpHandler)
		assert.Equal(t, "", tool.Name)        // Will be set by caller
		assert.Equal(t, "", tool.Description) // Will be set by caller
	})

	t.Run("create calculation tool", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.NoError(t, err)
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
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.NotNil(t, mcpHandler)
	})

	t.Run("unknown builtin type", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinType(999),
			Config:      map[string]string{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "unknown builtin tool type")
	})
}

func TestScriptToolHandler_CreateMCPTool(t *testing.T) {
	t.Run("script tool with mock evaluator error", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: &mockEvaluator{},
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "compiled evaluator is nil")
	})

	t.Run("script tool with nil evaluator", func(t *testing.T) {
		handler := &ScriptToolHandler{
			Evaluator: nil,
		}

		tool, mcpHandler, err := handler.CreateMCPTool()
		assert.Error(t, err)
		assert.Nil(t, tool)
		assert.Nil(t, mcpHandler)
		assert.Contains(t, err.Error(), "script tool handler requires an evaluator")
	})
}

// Mock evaluator for testing
type mockEvaluator struct{}

func (m *mockEvaluator) Type() evaluators.EvaluatorType {
	return evaluators.EvaluatorTypeRisor
}

func (m *mockEvaluator) Validate() error {
	return nil
}

func (m *mockEvaluator) GetCompiledEvaluator() (platform.Evaluator, error) {
	return nil, nil // Return nil to test error handling
}

func (m *mockEvaluator) GetTimeout() time.Duration {
	return time.Minute
}
