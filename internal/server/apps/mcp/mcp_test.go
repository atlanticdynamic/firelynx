package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mcpconfig "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create MCP config DTO from domain config
func createMCPConfig(t *testing.T, id string, domainConfig *mcpconfig.App) *Config {
	t.Helper()

	compiledServer := domainConfig.GetCompiledServer()
	if compiledServer == nil {
		t.Fatalf("compiled server is nil for app %s - domain validation may not have been run", id)
	}

	return &Config{
		ID:             id,
		CompiledServer: compiledServer,
	}
}

func TestNew(t *testing.T) {
	t.Run("valid config with compiled server", func(t *testing.T) {
		// Create and validate MCP config to get compiled server
		config := &mcpconfig.App{
			ID:            "test-server",
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &mcpconfig.Transport{},
			Tools:         []*mcpconfig.Tool{},
			Resources:     []*mcpconfig.Resource{},
			Prompts:       []*mcpconfig.Prompt{},
			Middlewares:   []*mcpconfig.Middleware{},
		}

		err := config.Validate()
		require.NoError(t, err, "config validation should succeed")
		require.NotNil(t, config.GetCompiledServer(), "compiled server should exist after validation")

		mcpConfig := createMCPConfig(t, "test-app", config)
		app, err := New(mcpConfig)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "test-app", app.id)
		assert.NotNil(t, app.handler)
	})

	t.Run("config without compiled server", func(t *testing.T) {
		// Create config without validation (no compiled server)
		config := &mcpconfig.App{
			ID:            "test-no-compile",
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
		}

		// Creating config should fail because server not compiled
		// We can't use the helper directly since it would call t.Fatalf
		// Instead, test the condition directly
		compiledServer := config.GetCompiledServer()
		assert.Nil(t, compiledServer, "compiled server should be nil when validation not run")
	})

	t.Run("SSE enabled should fail validation", func(t *testing.T) {
		// Create config with SSE enabled
		config := &mcpconfig.App{
			ID:            "test-sse",
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport: &mcpconfig.Transport{
				SSEEnabled: true,
				SSEPath:    "/events",
			},
			Tools:       []*mcpconfig.Tool{},
			Resources:   []*mcpconfig.Resource{},
			Prompts:     []*mcpconfig.Prompt{},
			Middlewares: []*mcpconfig.Middleware{},
		}

		// SSE enabled should fail validation
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE transport is not yet implemented for MCP apps")
	})
}

func TestApp_String(t *testing.T) {
	app := &App{
		id: "test-app-id",
	}

	assert.Equal(t, "test-app-id", app.String())
}

func TestApp_HandleHTTP(t *testing.T) {
	t.Run("successful HTTP handling", func(t *testing.T) {
		// Create valid MCP config
		config := &mcpconfig.App{
			ID:            "test-http",
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &mcpconfig.Transport{},
			Tools: []*mcpconfig.Tool{
				{
					Name:        "echo",
					Description: "Echo tool",
					Handler: &mcpconfig.BuiltinToolHandler{
						BuiltinType: mcpconfig.BuiltinEcho,
						Config:      map[string]string{},
					},
				},
			},
			Resources:   []*mcpconfig.Resource{},
			Prompts:     []*mcpconfig.Prompt{},
			Middlewares: []*mcpconfig.Middleware{},
		}

		err := config.Validate()
		require.NoError(t, err)

		mcpConfig := createMCPConfig(t, "test-app", config)
		app, err := New(mcpConfig)
		require.NoError(t, err)

		// Create test HTTP request
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()
		ctx := t.Context()

		// HandleHTTP should not return an error (MCP SDK handles errors internally)
		err = app.HandleHTTP(ctx, w, req)
		require.NoError(t, err)

		// The actual response depends on MCP SDK implementation
		// We just verify that the handler was called without panicking
	})

	t.Run("nil handler edge case", func(t *testing.T) {
		// Create app with nil handler (should not happen in practice)
		app := &App{
			id:      "test",
			handler: nil,
		}

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()
		ctx := t.Context()

		// Should panic with nil handler
		assert.Panics(t, func() {
			app.HandleHTTP(ctx, w, req) //nolint:errcheck // Expected to panic
		})
	})
}

func TestApp_Integration(t *testing.T) {
	t.Run("end-to-end MCP app creation and HTTP handling", func(t *testing.T) {
		// Create MCP config with multiple tools
		config := &mcpconfig.App{
			ID:            "test-integration",
			ServerName:    "Integration Test Server",
			ServerVersion: "1.0.0",
			Transport: &mcpconfig.Transport{
				SSEEnabled: false, // SSE disabled
			},
			Tools: []*mcpconfig.Tool{
				{
					Name:        "echo",
					Description: "Echo back input",
					Handler: &mcpconfig.BuiltinToolHandler{
						BuiltinType: mcpconfig.BuiltinEcho,
						Config:      map[string]string{},
					},
				},
				{
					Name:        "calculate",
					Description: "Perform calculations",
					Handler: &mcpconfig.BuiltinToolHandler{
						BuiltinType: mcpconfig.BuiltinCalculation,
						Config:      map[string]string{},
					},
				},
			},
		}

		// Validate config (compiles MCP server)
		err := config.Validate()
		require.NoError(t, err)

		mcpConfig := createMCPConfig(t, "integration-test", config)
		// Create MCP app
		app, err := New(mcpConfig)
		require.NoError(t, err)
		assert.Equal(t, "integration-test", app.String())

		// Test HTTP handling
		req := httptest.NewRequest(http.MethodPost, "/mcp/test", nil)
		w := httptest.NewRecorder()
		ctx := t.Context()

		err = app.HandleHTTP(ctx, w, req)
		require.NoError(t, err)

		// Response handling is delegated to MCP SDK
		// We verify the app was created successfully and handled the request
	})
}
