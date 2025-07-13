package mcp

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		app, err := FromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, app)
	})

	t.Run("minimal valid proto", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    stringPtr("Test Server"),
			ServerVersion: stringPtr("1.0.0"),
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "Test Server", app.ServerName)
		assert.Equal(t, "1.0.0", app.ServerVersion)
		assert.NotNil(t, app.Transport)
		assert.Empty(t, app.Tools)
		assert.Empty(t, app.Resources)
		assert.Empty(t, app.Prompts)
		assert.Empty(t, app.Middlewares)
	})

	t.Run("proto with transport", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    stringPtr("Test Server"),
			ServerVersion: stringPtr("1.0.0"),
			Transport: &pbApps.McpTransport{
				SseEnabled: boolPtr(true),
				SsePath:    stringPtr("/events"),
			},
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, app.Transport)
		assert.True(t, app.Transport.SSEEnabled)
		assert.Equal(t, "/events", app.Transport.SSEPath)
	})

	t.Run("proto with builtin tool", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    stringPtr("Test Server"),
			ServerVersion: stringPtr("1.0.0"),
			Tools: []*pbApps.McpTool{
				{
					Name:        stringPtr("echo"),
					Description: stringPtr("Echo tool"),
					Handler: &pbApps.McpTool_Builtin{
						Builtin: &pbApps.McpBuiltinHandler{
							Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_ECHO),
							Config: map[string]string{
								"key": "value",
							},
						},
					},
				},
			},
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.Len(t, app.Tools, 1)
		assert.Equal(t, "echo", app.Tools[0].Name)
		assert.Equal(t, "Echo tool", app.Tools[0].Description)

		handler, ok := app.Tools[0].Handler.(*BuiltinToolHandler)
		assert.True(t, ok)
		assert.Equal(t, BuiltinEcho, handler.BuiltinType)
		assert.Equal(t, "value", handler.Config["key"])
	})

	t.Run("proto with middleware", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    stringPtr("Test Server"),
			ServerVersion: stringPtr("1.0.0"),
			Middlewares: []*pbApps.McpMiddleware{
				{
					Type: mcpMiddlewareTypePtr(pbApps.McpMiddleware_RATE_LIMITING),
					Config: map[string]string{
						"rate": "100",
					},
				},
			},
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.Len(t, app.Middlewares, 1)
		assert.Equal(t, MiddlewareRateLimiting, app.Middlewares[0].Type)
		assert.Equal(t, "100", app.Middlewares[0].Config["rate"])
	})
}

func TestToProto(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		var app *App
		proto := app.ToProto()
		assert.Nil(t, proto)
	})

	t.Run("minimal valid app", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &Transport{},
			Tools:         []*Tool{},
			Resources:     []*Resource{},
			Prompts:       []*Prompt{},
			Middlewares:   []*Middleware{},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Equal(t, "Test Server", *proto.ServerName)
		assert.Equal(t, "1.0.0", *proto.ServerVersion)
		assert.NotNil(t, proto.Transport)
		assert.Empty(t, proto.Tools)
		assert.Empty(t, proto.Resources)
		assert.Empty(t, proto.Prompts)
		assert.Empty(t, proto.Middlewares)
	})

	t.Run("app with SSE transport", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport: &Transport{
				SSEEnabled: true,
				SSEPath:    "/events",
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)
		assert.NotNil(t, proto.Transport)
		assert.True(t, *proto.Transport.SseEnabled)
		assert.Equal(t, "/events", *proto.Transport.SsePath)
	})

	t.Run("app with builtin tool", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Tools: []*Tool{
				{
					Name:        "echo",
					Description: "Echo tool",
					Handler: &BuiltinToolHandler{
						BuiltinType: BuiltinEcho,
						Config: map[string]string{
							"key": "value",
						},
					},
				},
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Len(t, proto.Tools, 1)
		assert.Equal(t, "echo", *proto.Tools[0].Name)
		assert.Equal(t, "Echo tool", *proto.Tools[0].Description)

		builtinHandler := proto.Tools[0].Handler.(*pbApps.McpTool_Builtin)
		assert.Equal(t, pbApps.McpBuiltinHandler_ECHO, *builtinHandler.Builtin.Type)
		assert.Equal(t, "value", builtinHandler.Builtin.Config["key"])
	})
}

func TestTransportFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		transport, err := transportFromProto(nil)
		assert.NoError(t, err)
		assert.NotNil(t, transport)
		assert.False(t, transport.SSEEnabled)
		assert.Empty(t, transport.SSEPath)
	})

	t.Run("SSE enabled", func(t *testing.T) {
		proto := &pbApps.McpTransport{
			SseEnabled: boolPtr(true),
			SsePath:    stringPtr("/events"),
		}

		transport, err := transportFromProto(proto)
		assert.NoError(t, err)
		assert.True(t, transport.SSEEnabled)
		assert.Equal(t, "/events", transport.SSEPath)
	})

	t.Run("SSE disabled", func(t *testing.T) {
		proto := &pbApps.McpTransport{
			SseEnabled: boolPtr(false),
		}

		transport, err := transportFromProto(proto)
		assert.NoError(t, err)
		assert.False(t, transport.SSEEnabled)
		assert.Empty(t, transport.SSEPath)
	})
}

func TestTransportToProto(t *testing.T) {
	t.Run("nil transport", func(t *testing.T) {
		var transport *Transport
		proto := transport.toProto()
		assert.Nil(t, proto)
	})

	t.Run("SSE enabled", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: true,
			SSEPath:    "/events",
		}

		proto := transport.toProto()
		assert.NotNil(t, proto)
		assert.True(t, *proto.SseEnabled)
		assert.Equal(t, "/events", *proto.SsePath)
	})

	t.Run("SSE disabled", func(t *testing.T) {
		transport := &Transport{
			SSEEnabled: false,
		}

		proto := transport.toProto()
		assert.NotNil(t, proto)
		assert.False(t, *proto.SseEnabled)
		assert.Nil(t, proto.SsePath)
	})
}

func TestToolFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		tool, err := toolFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, tool)
	})

	t.Run("builtin tool", func(t *testing.T) {
		proto := &pbApps.McpTool{
			Name:        stringPtr("echo"),
			Description: stringPtr("Echo tool"),
			Handler: &pbApps.McpTool_Builtin{
				Builtin: &pbApps.McpBuiltinHandler{
					Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_ECHO),
					Config: map[string]string{
						"key": "value",
					},
				},
			},
		}

		tool, err := toolFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.Equal(t, "echo", tool.Name)
		assert.Equal(t, "Echo tool", tool.Description)

		handler, ok := tool.Handler.(*BuiltinToolHandler)
		assert.True(t, ok)
		assert.Equal(t, BuiltinEcho, handler.BuiltinType)
		assert.Equal(t, "value", handler.Config["key"])
	})

	t.Run("script tool", func(t *testing.T) {
		proto := &pbApps.McpTool{
			Name:        stringPtr("script"),
			Description: stringPtr("Script tool"),
			Handler: &pbApps.McpTool_Script{
				Script: &pbApps.McpScriptHandler{
					StaticData: &pbData.StaticData{
						Data: map[string]*structpb.Value{
							"key": structpb.NewStringValue("value"),
						},
					},
				},
			},
		}

		tool, err := toolFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.Equal(t, "script", tool.Name)
		assert.Equal(t, "Script tool", tool.Description)

		handler, ok := tool.Handler.(*ScriptToolHandler)
		assert.True(t, ok)
		assert.NotNil(t, handler.StaticData)
	})

	t.Run("no handler", func(t *testing.T) {
		proto := &pbApps.McpTool{
			Name:        stringPtr("empty"),
			Description: stringPtr("Empty tool"),
		}

		tool, err := toolFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.Equal(t, "empty", tool.Name)
		assert.Nil(t, tool.Handler)
	})
}

//nolint:dupl // Different function being tested despite similar test structure
func TestBuiltinHandlerFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		handler, err := builtinHandlerFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, handler)
	})

	t.Run("echo handler", func(t *testing.T) {
		proto := &pbApps.McpBuiltinHandler{
			Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_ECHO),
			Config: map[string]string{
				"key": "value",
			},
		}

		handler, err := builtinHandlerFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, handler)
		assert.Equal(t, BuiltinEcho, handler.BuiltinType)
		assert.Equal(t, "value", handler.Config["key"])
	})

	t.Run("calculation handler", func(t *testing.T) {
		proto := &pbApps.McpBuiltinHandler{
			Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_CALCULATION),
		}

		handler, err := builtinHandlerFromProto(proto)
		assert.NoError(t, err)
		assert.Equal(t, BuiltinCalculation, handler.BuiltinType)
	})

	t.Run("file read handler", func(t *testing.T) {
		proto := &pbApps.McpBuiltinHandler{
			Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_FILE_READ),
		}

		handler, err := builtinHandlerFromProto(proto)
		assert.NoError(t, err)
		assert.Equal(t, BuiltinFileRead, handler.BuiltinType)
	})
}

func TestBuiltinHandlerToProto(t *testing.T) {
	t.Run("nil handler", func(t *testing.T) {
		var handler *BuiltinToolHandler
		proto := handler.toProto()
		assert.Nil(t, proto)
	})

	t.Run("echo handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinEcho,
			Config: map[string]string{
				"key": "value",
			},
		}

		proto := handler.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpBuiltinHandler_ECHO, *proto.Type)
		assert.Equal(t, "value", proto.Config["key"])
	})

	t.Run("calculation handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinCalculation,
			Config:      map[string]string{},
		}

		proto := handler.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpBuiltinHandler_CALCULATION, *proto.Type)
	})

	t.Run("file read handler", func(t *testing.T) {
		handler := &BuiltinToolHandler{
			BuiltinType: BuiltinFileRead,
			Config:      map[string]string{},
		}

		proto := handler.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpBuiltinHandler_FILE_READ, *proto.Type)
	})
}

//nolint:dupl // Different function being tested despite similar test structure
func TestMiddlewareFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		middleware, err := middlewareFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, middleware)
	})

	t.Run("rate limiting middleware", func(t *testing.T) {
		proto := &pbApps.McpMiddleware{
			Type: mcpMiddlewareTypePtr(pbApps.McpMiddleware_RATE_LIMITING),
			Config: map[string]string{
				"rate": "100",
			},
		}

		middleware, err := middlewareFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, middleware)
		assert.Equal(t, MiddlewareRateLimiting, middleware.Type)
		assert.Equal(t, "100", middleware.Config["rate"])
	})

	t.Run("logging middleware", func(t *testing.T) {
		proto := &pbApps.McpMiddleware{
			Type: mcpMiddlewareTypePtr(pbApps.McpMiddleware_MCP_LOGGING),
		}

		middleware, err := middlewareFromProto(proto)
		assert.NoError(t, err)
		assert.Equal(t, MiddlewareLogging, middleware.Type)
	})

	t.Run("authentication middleware", func(t *testing.T) {
		proto := &pbApps.McpMiddleware{
			Type: mcpMiddlewareTypePtr(pbApps.McpMiddleware_MCP_AUTHENTICATION),
		}

		middleware, err := middlewareFromProto(proto)
		assert.NoError(t, err)
		assert.Equal(t, MiddlewareAuthentication, middleware.Type)
	})
}

func TestResourceFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		resource, err := resourceFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, resource)
	})

	t.Run("complete resource", func(t *testing.T) {
		proto := &pbApps.McpResource{
			Uri:         stringPtr("test://resource"),
			Name:        stringPtr("Test Resource"),
			Description: stringPtr("A test resource"),
			MimeType:    stringPtr("text/plain"),
		}

		resource, err := resourceFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "test://resource", resource.URI)
		assert.Equal(t, "Test Resource", resource.Name)
		assert.Equal(t, "A test resource", resource.Description)
		assert.Equal(t, "text/plain", resource.MIMEType)
	})
}

func TestPromptFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		prompt, err := promptFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, prompt)
	})

	t.Run("complete prompt", func(t *testing.T) {
		proto := &pbApps.McpPrompt{
			Name:        stringPtr("Test Prompt"),
			Description: stringPtr("A test prompt"),
		}

		prompt, err := promptFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, prompt)
		assert.Equal(t, "Test Prompt", prompt.Name)
		assert.Equal(t, "A test prompt", prompt.Description)
	})
}

// Helper functions for creating protobuf pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func mcpBuiltinTypePtr(t pbApps.McpBuiltinHandler_Type) *pbApps.McpBuiltinHandler_Type {
	return &t
}

func mcpMiddlewareTypePtr(t pbApps.McpMiddleware_Type) *pbApps.McpMiddleware_Type {
	return &t
}
