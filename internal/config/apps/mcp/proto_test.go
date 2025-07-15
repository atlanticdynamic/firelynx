package mcp

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		app, err := FromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, app)
	})

	t.Run("minimal valid proto", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
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
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
			Transport: &pbApps.McpTransport{
				SseEnabled: proto.Bool(true),
				SsePath:    proto.String("/events"),
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
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
			Tools: []*pbApps.McpTool{
				{
					Name:        proto.String("echo"),
					Description: proto.String("Echo tool"),
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
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
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

	t.Run("proto with resources", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
			Resources: []*pbApps.McpResource{
				{
					Uri:         proto.String("test://resource"),
					Name:        proto.String("Test Resource"),
					Description: proto.String("A test resource"),
					MimeType:    proto.String("text/plain"),
				},
			},
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.Len(t, app.Resources, 1)
		assert.Equal(t, "test://resource", app.Resources[0].URI)
		assert.Equal(t, "Test Resource", app.Resources[0].Name)
	})

	t.Run("proto with prompts", func(t *testing.T) {
		proto := &pbApps.McpApp{
			ServerName:    proto.String("Test Server"),
			ServerVersion: proto.String("1.0.0"),
			Prompts: []*pbApps.McpPrompt{
				{
					Name:        proto.String("Test Prompt"),
					Description: proto.String("A test prompt"),
				},
			},
		}

		app, err := FromProto(proto)
		assert.NoError(t, err)
		assert.Len(t, app.Prompts, 1)
		assert.Equal(t, "Test Prompt", app.Prompts[0].Name)
		assert.Equal(t, "A test prompt", app.Prompts[0].Description)
	})
}

func TestToProto(t *testing.T) {
	t.Parallel()

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

	t.Run("app with script tool", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Tools: []*Tool{
				{
					Name:        "script",
					Description: "Script tool",
					Handler: &ScriptToolHandler{
						StaticData: &staticdata.StaticData{
							Data: map[string]any{
								"key": "value",
							},
						},
					},
				},
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Len(t, proto.Tools, 1)
		assert.Equal(t, "script", *proto.Tools[0].Name)

		scriptHandler := proto.Tools[0].Handler.(*pbApps.McpTool_Script)
		assert.NotNil(t, scriptHandler.Script)
		assert.NotNil(t, scriptHandler.Script.StaticData)
	})

	t.Run("app with resources", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Resources: []*Resource{
				{
					URI:         "test://resource",
					Name:        "Test Resource",
					Description: "A test resource",
					MIMEType:    "text/plain",
				},
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Len(t, proto.Resources, 1)
		assert.Equal(t, "test://resource", *proto.Resources[0].Uri)
		assert.Equal(t, "Test Resource", *proto.Resources[0].Name)
	})

	t.Run("app with prompts", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Prompts: []*Prompt{
				{
					Name:        "Test Prompt",
					Description: "A test prompt",
				},
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Len(t, proto.Prompts, 1)
		assert.Equal(t, "Test Prompt", *proto.Prompts[0].Name)
		assert.Equal(t, "A test prompt", *proto.Prompts[0].Description)
	})

	t.Run("app with middlewares", func(t *testing.T) {
		app := &App{
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Middlewares: []*Middleware{
				{
					Type: MiddlewareRateLimiting,
					Config: map[string]string{
						"rate": "100",
					},
				},
			},
		}

		result := app.ToProto()
		require.IsType(t, (*pbApps.McpApp)(nil), result)
		proto := result.(*pbApps.McpApp)

		assert.Len(t, proto.Middlewares, 1)
		assert.Equal(t, pbApps.McpMiddleware_RATE_LIMITING, *proto.Middlewares[0].Type)
		assert.Equal(t, "100", proto.Middlewares[0].Config["rate"])
	})
}

func TestTransportFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		transport, err := transportFromProto(nil)
		assert.NoError(t, err)
		assert.NotNil(t, transport)
		assert.False(t, transport.SSEEnabled)
		assert.Empty(t, transport.SSEPath)
	})

	t.Run("SSE enabled", func(t *testing.T) {
		proto := &pbApps.McpTransport{
			SseEnabled: proto.Bool(true),
			SsePath:    proto.String("/events"),
		}

		transport, err := transportFromProto(proto)
		assert.NoError(t, err)
		assert.True(t, transport.SSEEnabled)
		assert.Equal(t, "/events", transport.SSEPath)
	})

	t.Run("SSE disabled", func(t *testing.T) {
		proto := &pbApps.McpTransport{
			SseEnabled: proto.Bool(false),
		}

		transport, err := transportFromProto(proto)
		assert.NoError(t, err)
		assert.False(t, transport.SSEEnabled)
		assert.Empty(t, transport.SSEPath)
	})
}

func TestTransportToProto(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		tool, err := toolFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, tool)
	})

	t.Run("builtin tool", func(t *testing.T) {
		proto := &pbApps.McpTool{
			Name:        proto.String("echo"),
			Description: proto.String("Echo tool"),
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
			Name:        proto.String("script"),
			Description: proto.String("Script tool"),
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
			Name:        proto.String("empty"),
			Description: proto.String("Empty tool"),
		}

		tool, err := toolFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.Equal(t, "empty", tool.Name)
		assert.Nil(t, tool.Handler)
	})

	t.Run("tool with all fields", func(t *testing.T) {
		destructive := true
		openWorld := false
		proto := &pbApps.McpTool{
			Name:         proto.String("complete"),
			Description:  proto.String("Complete tool"),
			Title:        proto.String("Complete Tool"),
			InputSchema:  proto.String(`{"type": "object"}`),
			OutputSchema: proto.String(`{"type": "string"}`),
			Annotations: &pbApps.McpToolAnnotations{
				Title:           proto.String("Annotation Title"),
				ReadOnlyHint:    proto.Bool(true),
				IdempotentHint:  proto.Bool(true),
				DestructiveHint: &destructive,
				OpenWorldHint:   &openWorld,
			},
			Handler: &pbApps.McpTool_Builtin{
				Builtin: &pbApps.McpBuiltinHandler{
					Type: mcpBuiltinTypePtr(pbApps.McpBuiltinHandler_ECHO),
				},
			},
		}

		tool, err := toolFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, tool)
		assert.Equal(t, "complete", tool.Name)
		assert.Equal(t, "Complete tool", tool.Description)
		assert.Equal(t, "Complete Tool", tool.Title)
		assert.Equal(t, `{"type": "object"}`, tool.InputSchema)
		assert.Equal(t, `{"type": "string"}`, tool.OutputSchema)
		assert.NotNil(t, tool.Annotations)
		assert.Equal(t, "Annotation Title", tool.Annotations.Title)
		assert.True(t, tool.Annotations.ReadOnlyHint)
		assert.True(t, tool.Annotations.IdempotentHint)
		assert.Equal(t, &destructive, tool.Annotations.DestructiveHint)
		assert.Equal(t, &openWorld, tool.Annotations.OpenWorldHint)
	})
}

//nolint:dupl
func TestBuiltinHandlerFromProto(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestScriptHandlerFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		handler, err := scriptHandlerFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, handler)
	})

	t.Run("handler with static data", func(t *testing.T) {
		proto := &pbApps.McpScriptHandler{
			StaticData: &pbData.StaticData{
				Data: map[string]*structpb.Value{
					"key": structpb.NewStringValue("value"),
				},
			},
		}

		handler, err := scriptHandlerFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.StaticData)
		assert.Equal(t, "value", handler.StaticData.Data["key"])
	})

	t.Run("handler without static data", func(t *testing.T) {
		proto := &pbApps.McpScriptHandler{}

		handler, err := scriptHandlerFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.StaticData)
		assert.Nil(t, handler.Evaluator)
	})
}

func TestResourceFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		resource, err := resourceFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, resource)
	})

	t.Run("complete resource", func(t *testing.T) {
		proto := &pbApps.McpResource{
			Uri:         proto.String("test://resource"),
			Name:        proto.String("Test Resource"),
			Description: proto.String("A test resource"),
			MimeType:    proto.String("text/plain"),
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
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		prompt, err := promptFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, prompt)
	})

	t.Run("complete prompt", func(t *testing.T) {
		proto := &pbApps.McpPrompt{
			Name:        proto.String("Test Prompt"),
			Description: proto.String("A test prompt"),
		}

		prompt, err := promptFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, prompt)
		assert.Equal(t, "Test Prompt", prompt.Name)
		assert.Equal(t, "A test prompt", prompt.Description)
	})
}

func mcpBuiltinTypePtr(t pbApps.McpBuiltinHandler_Type) *pbApps.McpBuiltinHandler_Type {
	return &t
}

func mcpMiddlewareTypePtr(t pbApps.McpMiddleware_Type) *pbApps.McpMiddleware_Type {
	return &t
}

func TestScriptHandlerToProto(t *testing.T) {
	t.Run("nil handler", func(t *testing.T) {
		var handler *ScriptToolHandler
		proto := handler.toProto()
		assert.Nil(t, proto)
	})

	t.Run("handler with static data", func(t *testing.T) {
		handler := &ScriptToolHandler{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{
					"key": "value",
				},
			},
		}

		proto := handler.toProto()
		assert.NotNil(t, proto)
		assert.NotNil(t, proto.StaticData)
	})

	t.Run("handler without static data", func(t *testing.T) {
		handler := &ScriptToolHandler{}

		proto := handler.toProto()
		assert.NotNil(t, proto)
		assert.Nil(t, proto.StaticData)
	})
}

func TestResourceToProto(t *testing.T) {
	t.Run("nil resource", func(t *testing.T) {
		var resource *Resource
		proto := resource.toProto()
		assert.Nil(t, proto)
	})

	t.Run("complete resource", func(t *testing.T) {
		resource := &Resource{
			URI:         "test://resource",
			Name:        "Test Resource",
			Description: "A test resource",
			MIMEType:    "text/plain",
		}

		proto := resource.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, "test://resource", *proto.Uri)
		assert.Equal(t, "Test Resource", *proto.Name)
		assert.Equal(t, "A test resource", *proto.Description)
		assert.Equal(t, "text/plain", *proto.MimeType)
	})

	t.Run("minimal resource", func(t *testing.T) {
		resource := &Resource{}

		proto := resource.toProto()
		assert.NotNil(t, proto)
		assert.Nil(t, proto.Uri)
		assert.Nil(t, proto.Name)
		assert.Nil(t, proto.Description)
		assert.Nil(t, proto.MimeType)
	})
}

func TestPromptToProto(t *testing.T) {
	t.Run("nil prompt", func(t *testing.T) {
		var prompt *Prompt
		proto := prompt.toProto()
		assert.Nil(t, proto)
	})

	t.Run("complete prompt", func(t *testing.T) {
		prompt := &Prompt{
			Name:        "Test Prompt",
			Description: "A test prompt",
		}

		proto := prompt.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, "Test Prompt", *proto.Name)
		assert.Equal(t, "A test prompt", *proto.Description)
	})

	t.Run("minimal prompt", func(t *testing.T) {
		prompt := &Prompt{}

		proto := prompt.toProto()
		assert.NotNil(t, proto)
		assert.Nil(t, proto.Name)
		assert.Nil(t, proto.Description)
	})
}

func TestMiddlewareToProto(t *testing.T) {
	t.Run("nil middleware", func(t *testing.T) {
		var middleware *Middleware
		proto := middleware.toProto()
		assert.Nil(t, proto)
	})

	t.Run("rate limiting middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type: MiddlewareRateLimiting,
			Config: map[string]string{
				"rate": "100",
			},
		}

		proto := middleware.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpMiddleware_RATE_LIMITING, *proto.Type)
		assert.Equal(t, "100", proto.Config["rate"])
	})

	t.Run("logging middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareLogging,
			Config: map[string]string{},
		}

		proto := middleware.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpMiddleware_MCP_LOGGING, *proto.Type)
	})

	t.Run("authentication middleware", func(t *testing.T) {
		middleware := &Middleware{
			Type:   MiddlewareAuthentication,
			Config: map[string]string{},
		}

		proto := middleware.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, pbApps.McpMiddleware_MCP_AUTHENTICATION, *proto.Type)
	})
}

func TestToolAnnotationsFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		annotations, err := toolAnnotationsFromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, annotations)
	})

	t.Run("complete annotations", func(t *testing.T) {
		destructive := true
		openWorld := false
		proto := &pbApps.McpToolAnnotations{
			Title:           proto.String("Test Tool"),
			ReadOnlyHint:    proto.Bool(true),
			IdempotentHint:  proto.Bool(true),
			DestructiveHint: &destructive,
			OpenWorldHint:   &openWorld,
		}

		annotations, err := toolAnnotationsFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, annotations)
		assert.Equal(t, "Test Tool", annotations.Title)
		assert.True(t, annotations.ReadOnlyHint)
		assert.True(t, annotations.IdempotentHint)
		assert.Equal(t, &destructive, annotations.DestructiveHint)
		assert.Equal(t, &openWorld, annotations.OpenWorldHint)
	})

	t.Run("minimal annotations", func(t *testing.T) {
		proto := &pbApps.McpToolAnnotations{}

		annotations, err := toolAnnotationsFromProto(proto)
		assert.NoError(t, err)
		assert.NotNil(t, annotations)
		assert.Empty(t, annotations.Title)
		assert.False(t, annotations.ReadOnlyHint)
		assert.False(t, annotations.IdempotentHint)
		assert.Nil(t, annotations.DestructiveHint)
		assert.Nil(t, annotations.OpenWorldHint)
	})
}

func TestToolAnnotationsToProto(t *testing.T) {
	t.Run("nil annotations", func(t *testing.T) {
		var annotations *ToolAnnotations
		proto := annotations.toProto()
		assert.Nil(t, proto)
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

		proto := annotations.toProto()
		assert.NotNil(t, proto)
		assert.Equal(t, "Test Tool", *proto.Title)
		assert.True(t, *proto.ReadOnlyHint)
		assert.True(t, *proto.IdempotentHint)
		assert.Equal(t, &destructive, proto.DestructiveHint)
		assert.Equal(t, &openWorld, proto.OpenWorldHint)
	})

	t.Run("minimal annotations", func(t *testing.T) {
		annotations := &ToolAnnotations{
			ReadOnlyHint:   false,
			IdempotentHint: false,
		}

		proto := annotations.toProto()
		assert.NotNil(t, proto)
		assert.Nil(t, proto.Title)
		assert.False(t, *proto.ReadOnlyHint)
		assert.False(t, *proto.IdempotentHint)
		assert.Nil(t, proto.DestructiveHint)
		assert.Nil(t, proto.OpenWorldHint)
	})
}
