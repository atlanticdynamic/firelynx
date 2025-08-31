package apps

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/script"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockApp is a test implementation of the App interface
type MockApp struct {
	id string
}

func (m *MockApp) String() string {
	return m.id
}

func (m *MockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	return nil
}

// mockInstantiator is a test instantiator that returns a MockApp
func mockInstantiator(id string, _ any) (App, error) {
	return &MockApp{id: id}, nil
}

func TestCreateEchoApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		id               string
		config           any
		expectedResponse string
		expectedError    error
	}{
		{
			name:          "fails with nil config",
			id:            "test-echo",
			config:        nil,
			expectedError: ErrInvalidConfigType,
		},
		{
			name:          "fails with non-echo config",
			id:            "echo-with-config",
			config:        struct{ foo string }{foo: "bar"},
			expectedError: ErrInvalidConfigType,
		},
		{
			name:             "creates echo app with custom response",
			id:               "custom-echo",
			config:           &configEcho.EchoApp{Response: "Custom Response"},
			expectedResponse: "Custom Response",
			expectedError:    nil,
		},
		{
			name:             "creates echo app with empty response string",
			id:               "empty-response-echo",
			config:           &configEcho.EchoApp{Response: ""},
			expectedResponse: "empty-response-echo", // defaults to ID when response is empty
			expectedError:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := createEchoApp(tt.id, tt.config)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, app)
			} else {
				require.NoError(t, err)
				require.NotNil(t, app)

				// Verify it returns the correct ID
				assert.Equal(t, tt.id, app.String())

				// Verify it's actually an echo.App instance
				echoApp, ok := app.(*echo.App)
				assert.True(t, ok, "should return an echo.App instance")
				assert.NotNil(t, echoApp)

				// Test the actual response by calling HandleHTTP
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/test", nil)
				ctx := t.Context()

				err = echoApp.HandleHTTP(ctx, w, r)
				require.NoError(t, err)

				// Verify the response matches expected
				assert.Equal(t, tt.expectedResponse, w.Body.String())
			}
		})
	}
}

func TestCreateScriptApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		config        any
		expectedError error
	}{
		{
			name: "creates script app with valid risor config",
			id:   "test-risor-script",
			config: &configScripts.AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{"message": "Hello Risor!"},
				},
				Evaluator: &evaluators.RisorEvaluator{
					Code:    `{"greeting": "Hello from Risor!"}`,
					Timeout: 30 * time.Second,
				},
			},
			expectedError: nil,
		},
		{
			name: "creates script app with valid starlark config",
			id:   "test-starlark-script",
			config: &configScripts.AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{"message": "Hello Starlark!"},
				},
				Evaluator: &evaluators.StarlarkEvaluator{
					Code: `result = {"greeting": "Hello from Starlark!"}
_ = result`,
					Timeout: 30 * time.Second,
				},
			},
			expectedError: nil,
		},
		{
			name: "creates script app with valid extism config",
			id:   "test-extism-script",
			config: &configScripts.AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{"input": "test data"},
				},
				Evaluator: &evaluators.ExtismEvaluator{
					Code:       base64.StdEncoding.EncodeToString(wasmdata.TestModule),
					Entrypoint: "greet",
					Timeout:    30 * time.Second,
				},
			},
			expectedError: nil,
		},
		{
			name: "creates script app with nil static data",
			id:   "test-nil-static",
			config: &configScripts.AppScript{
				StaticData: nil,
				Evaluator: &evaluators.RisorEvaluator{
					Code:    `{"greeting": "Hello!"}`,
					Timeout: 30 * time.Second,
				},
			},
			expectedError: nil,
		},
		{
			name:          "fails with wrong config type",
			id:            "test-wrong-type",
			config:        &configEcho.EchoApp{Response: "not a script config"},
			expectedError: ErrInvalidConfigType,
		},
		{
			name:          "fails with nil config",
			id:            "test-nil-config",
			config:        nil,
			expectedError: ErrInvalidConfigType,
		},
		{
			name: "fails with script app config that has nil evaluator",
			id:   "test-nil-evaluator",
			config: &configScripts.AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{"test": "data"},
				},
				Evaluator: nil,
			},
			expectedError: ErrConfigConversionFailed,
		},
		{
			name: "succeeds with evaluator that builds on demand",
			id:   "test-on-demand-build",
			config: &configScripts.AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{"test": "data"},
				},
				Evaluator: &evaluators.RisorEvaluator{
					Code:    `{"greeting": "Hello!"}`,
					Timeout: 30 * time.Second,
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For script configs, call Validate() to compile the evaluator
			if tt.config != nil {
				if scriptConfig, ok := tt.config.(*configScripts.AppScript); ok &&
					scriptConfig.Evaluator != nil {
					err := scriptConfig.Evaluator.Validate()
					if tt.expectedError == nil {
						require.NoError(
							t,
							err,
							"Test setup: evaluator validation should succeed for valid configs",
						)
					}
					// For error cases, validation might fail, but we still want to test createScriptApp
				}
			}

			app, err := createScriptApp(tt.id, tt.config)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, app)
			} else {
				require.NoError(t, err)
				require.NotNil(t, app)

				// Verify it returns the correct ID
				assert.Equal(t, tt.id, app.String())

				// Verify it's actually a script.ScriptApp instance
				scriptApp, ok := app.(*script.ScriptApp)
				assert.True(t, ok, "should return a script.ScriptApp instance")
				assert.NotNil(t, scriptApp)

				// Test that the app can handle HTTP requests (basic smoke test)
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/test", nil)
				ctx := t.Context()

				// Note: We don't require HandleHTTP to succeed since that depends on
				// script content and execution, but we verify it doesn't panic
				err := scriptApp.HandleHTTP(ctx, w, r)
				_ = err // Intentionally ignore error in smoke test
			}
		})
	}
}

func TestCreateScriptApp_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles empty app ID", func(t *testing.T) {
		config := &configScripts.AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"test": "data"},
			},
			Evaluator: &evaluators.RisorEvaluator{
				Code:    `{"greeting": "Hello!"}`,
				Timeout: 30 * time.Second,
			},
		}

		// Validate the evaluator first
		err := config.Evaluator.Validate()
		require.NoError(t, err)

		app, err := createScriptApp("", config)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Should handle empty ID gracefully
		assert.Empty(t, app.String())
	})

	t.Run("handles very long app ID", func(t *testing.T) {
		longID := "very-long-app-id-that-exceeds-normal-length-expectations-and-tests-boundary-conditions"
		config := &configScripts.AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"test": "data"},
			},
			Evaluator: &evaluators.RisorEvaluator{
				Code:    `{"greeting": "Hello!"}`,
				Timeout: 30 * time.Second,
			},
		}

		// Validate the evaluator first
		err := config.Evaluator.Validate()
		require.NoError(t, err)

		app, err := createScriptApp(longID, config)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, longID, app.String())
	})

	t.Run("handles special characters in app ID", func(t *testing.T) {
		specialID := "test-app-with-special-chars_123!@#"
		config := &configScripts.AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"test": "data"},
			},
			Evaluator: &evaluators.RisorEvaluator{
				Code:    `{"greeting": "Hello!"}`,
				Timeout: 30 * time.Second,
			},
		}

		// Validate the evaluator first
		err := config.Evaluator.Validate()
		require.NoError(t, err)

		app, err := createScriptApp(specialID, config)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, specialID, app.String())
	})
}

func TestCreateScriptApp_LoggerFields(t *testing.T) {
	t.Parallel()

	t.Run("app receives logger with correct fields", func(t *testing.T) {
		config := &configScripts.AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"test": "data"},
			},
			Evaluator: &evaluators.RisorEvaluator{
				Code:    `{"greeting": "Hello!"}`,
				Timeout: 30 * time.Second,
			},
		}

		// Validate the evaluator first
		err := config.Evaluator.Validate()
		require.NoError(t, err)

		appID := "test-script-app"
		app, err := createScriptApp(appID, config)
		require.NoError(t, err)
		require.NotNil(t, app)

		// The logger should have been configured with app_type and app_id fields
		// We can't easily test the logger fields directly, but we can verify
		// the app was created successfully with the logger
		assert.Equal(t, appID, app.String())
	})
}

func TestCreateScriptApp_Debug(t *testing.T) {
	t.Parallel()

	t.Run("debug unvalidated evaluator behavior", func(t *testing.T) {
		// Create a completely zero-value evaluator
		evaluator := &evaluators.RisorEvaluator{}

		// Check what GetCompiledEvaluator returns for zero-value evaluator
		compiledBefore, err := evaluator.GetCompiledEvaluator()
		t.Logf("Zero value GetCompiledEvaluator: %v, err: %v", compiledBefore, err)
		require.Error(t, err)
		assert.Nil(t, compiledBefore)

		// Now try with valid Code/Timeout
		evaluator2 := &evaluators.RisorEvaluator{
			Code:    `{"greeting": "Hello!"}`,
			Timeout: 30 * time.Second,
		}

		compiledBefore2, err2 := evaluator2.GetCompiledEvaluator()
		t.Logf("With fields GetCompiledEvaluator: %v, err: %v", compiledBefore2, err2)
		require.NoError(t, err2)
		assert.NotNil(t, compiledBefore2)
		t.Logf("Type of compiled evaluator: %T", compiledBefore2)

		config := &configScripts.AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"test": "data"},
			},
			Evaluator: evaluator2,
		}

		// Try to create the app directly
		app, err := createScriptApp("test-debug", config)
		t.Logf("createScriptApp result: app=%v, err=%v", app, err)

		require.NoError(t, err)
		assert.NotNil(t, app)
	})
}

func TestCreateMCPApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		config        any
		expectedError error
	}{
		{
			name: "creates MCP app with valid config",
			id:   "test-mcp-app",
			config: &configMCP.App{
				ID:            "test-mcp-app",
				ServerName:    "Test MCP Server",
				ServerVersion: "1.0.0",
				Transport:     &configMCP.Transport{},
				Tools:         []*configMCP.Tool{},
				Resources:     []*configMCP.Resource{},
				Prompts:       []*configMCP.Prompt{},
				Middlewares:   []*configMCP.Middleware{},
			},
			expectedError: nil,
		},
		{
			name: "creates MCP app with tools",
			id:   "test-mcp-with-tools",
			config: &configMCP.App{
				ID:            "test-mcp-with-tools",
				ServerName:    "MCP Server with Tools",
				ServerVersion: "1.0.0",
				Transport:     &configMCP.Transport{},
				Tools: []*configMCP.Tool{
					{
						Name:        "echo",
						Description: "Echo tool",
						Handler: &configMCP.BuiltinToolHandler{
							BuiltinType: configMCP.BuiltinEcho,
							Config:      map[string]string{},
						},
					},
				},
				Resources:   []*configMCP.Resource{},
				Prompts:     []*configMCP.Prompt{},
				Middlewares: []*configMCP.Middleware{},
			},
			expectedError: nil,
		},
		{
			name:          "fails with wrong config type",
			id:            "test-wrong-type",
			config:        &configEcho.EchoApp{Response: "not an MCP config"},
			expectedError: ErrInvalidConfigType,
		},
		{
			name:          "fails with nil config",
			id:            "test-nil-config",
			config:        nil,
			expectedError: ErrInvalidConfigType,
		},
		{
			name: "fails with unvalidated config",
			id:   "test-unvalidated",
			config: &configMCP.App{
				ID:            "test-unvalidated",
				ServerName:    "Unvalidated Server",
				ServerVersion: "1.0.0",
				// Not calling Validate() - should fail with no compiled server
			},
			expectedError: ErrConfigConversionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For MCP configs, call Validate() to compile the MCP server
			// Exception: skip validation for the "unvalidated" test case
			if tt.config != nil && tt.name != "fails with unvalidated config" {
				if mcpConfig, ok := tt.config.(*configMCP.App); ok {
					err := mcpConfig.Validate()
					if tt.expectedError == nil {
						require.NoError(
							t,
							err,
							"Test setup: MCP config validation should succeed for valid configs",
						)
					}
				}
			}

			app, err := createMCPApp(tt.id, tt.config)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, app)
			} else {
				require.NoError(t, err)
				require.NotNil(t, app)

				// Verify it returns the correct ID
				assert.Equal(t, tt.id, app.String())

				// Verify it's actually an mcp.App instance
				mcpApp, ok := app.(*mcp.App)
				assert.True(t, ok, "should return an mcp.App instance")
				assert.NotNil(t, mcpApp)

				// Test that the app can handle HTTP requests (basic smoke test)
				w := httptest.NewRecorder()
				r := httptest.NewRequest("POST", "/mcp/test", nil)
				ctx := t.Context()

				// Note: We don't require HandleHTTP to succeed since that depends on
				// MCP SDK implementation, but we verify it doesn't panic
				err := mcpApp.HandleHTTP(ctx, w, r)
				_ = err // Intentionally ignore error in smoke test
			}
		})
	}
}

func TestCreateMCPApp_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles empty app ID", func(t *testing.T) {
		config := &configMCP.App{
			ID:            "test-empty-id",
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &configMCP.Transport{},
			Tools:         []*configMCP.Tool{},
			Resources:     []*configMCP.Resource{},
			Prompts:       []*configMCP.Prompt{},
			Middlewares:   []*configMCP.Middleware{},
		}

		// Validate the config first
		err := config.Validate()
		require.NoError(t, err)

		app, err := createMCPApp("", config)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Should handle empty ID gracefully
		assert.Empty(t, app.String())
	})

	t.Run("handles very long app ID", func(t *testing.T) {
		longID := "very-long-mcp-app-id-that-exceeds-normal-length-expectations-and-tests-boundary-conditions"
		config := &configMCP.App{
			ID:            longID,
			ServerName:    "Test Server",
			ServerVersion: "1.0.0",
			Transport:     &configMCP.Transport{},
			Tools:         []*configMCP.Tool{},
			Resources:     []*configMCP.Resource{},
			Prompts:       []*configMCP.Prompt{},
			Middlewares:   []*configMCP.Middleware{},
		}

		// Validate the config first
		err := config.Validate()
		require.NoError(t, err)

		app, err := createMCPApp(longID, config)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, longID, app.String())
	})

	t.Run("fails validation with SSE enabled", func(t *testing.T) {
		config := &configMCP.App{
			ID:            "test-sse-server",
			ServerName:    "SSE Test Server",
			ServerVersion: "1.0.0",
			Transport: &configMCP.Transport{
				SSEEnabled: true,
				SSEPath:    "/events",
			},
			Tools:       []*configMCP.Tool{},
			Resources:   []*configMCP.Resource{},
			Prompts:     []*configMCP.Prompt{},
			Middlewares: []*configMCP.Middleware{},
		}

		// SSE enabled should fail validation
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE transport is not yet implemented for MCP apps")
	})
}
