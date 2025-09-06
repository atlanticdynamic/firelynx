package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEvaluatorAdapter adapts a mock evaluator to implement evaluators.Evaluator interface
type mockEvaluatorAdapter struct {
	mock.Mock
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

// mockPlatformEvaluator mocks the platform.Evaluator interface
type mockPlatformEvaluator struct {
	mock.Mock
}

func (m *mockPlatformEvaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(platform.EvaluatorResponse), args.Error(1)
}

func (m *mockPlatformEvaluator) AddDataToContext(ctx context.Context, data ...map[string]any) (context.Context, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(context.Context), args.Error(1)
}

func TestConvertEchoConfig(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		config   *configEcho.EchoApp
		wantErr  bool
		wantID   string
		wantResp string
	}{
		{
			name:    "nil config",
			id:      "test-id",
			config:  nil,
			wantErr: true,
		},
		{
			name:     "empty response defaults to ID",
			id:       "test-app",
			config:   &configEcho.EchoApp{Response: ""},
			wantErr:  false,
			wantID:   "test-app",
			wantResp: "test-app",
		},
		{
			name:     "custom response",
			id:       "test-app",
			config:   &configEcho.EchoApp{Response: "custom response"},
			wantErr:  false,
			wantID:   "test-app",
			wantResp: "custom response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertEchoConfig(tt.id, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantID, result.ID)
				assert.Equal(t, tt.wantResp, result.Response)
			}
		})
	}
}

func TestConvertScriptConfig(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		config    *configScripts.AppScript
		setupMock func(*configScripts.AppScript)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "nil config",
			id:      "test-id",
			config:  nil,
			wantErr: true,
			errMsg:  "script config cannot be nil",
		},
		{
			name:    "nil evaluator",
			id:      "test-id",
			config:  &configScripts.AppScript{},
			wantErr: true,
			errMsg:  "script app must have an evaluator",
		},
		{
			name:   "evaluator GetCompiledEvaluator error",
			id:     "test-id",
			config: &configScripts.AppScript{},
			setupMock: func(cfg *configScripts.AppScript) {
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(nil, assert.AnError)
				cfg.Evaluator = mockEvaluator
			},
			wantErr: true,
			errMsg:  "failed to get compiled evaluator for app test-id",
		},
		{
			name:   "nil compiled evaluator",
			id:     "test-id",
			config: &configScripts.AppScript{},
			setupMock: func(cfg *configScripts.AppScript) {
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(nil, nil)
				cfg.Evaluator = mockEvaluator
			},
			wantErr: true,
			errMsg:  "compiled evaluator is nil for app test-id - domain validation may not have been run",
		},
		{
			name:   "successful conversion with minimal config",
			id:     "test-id",
			config: &configScripts.AppScript{},
			setupMock: func(cfg *configScripts.AppScript) {
				mockPlatformEvaluator := &mockPlatformEvaluator{}
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(mockPlatformEvaluator, nil)
				mockEvaluator.On("GetTimeout").Return(30 * time.Second)
				cfg.Evaluator = mockEvaluator
			},
			wantErr: false,
		},
		{
			name:   "successful conversion with static data",
			id:     "test-id",
			config: &configScripts.AppScript{},
			setupMock: func(cfg *configScripts.AppScript) {
				mockPlatformEvaluator := &mockPlatformEvaluator{}
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(mockPlatformEvaluator, nil)
				mockEvaluator.On("GetTimeout").Return(30 * time.Second)
				cfg.Evaluator = mockEvaluator
				cfg.StaticData = &staticdata.StaticData{
					Data: map[string]any{
						"key1": "value1",
						"key2": 42,
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(tt.config)
			}

			result, err := convertScriptConfig(tt.id, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
				assert.NotNil(t, result.CompiledEvaluator)
				assert.NotNil(t, result.Logger)
				assert.Equal(t, 30*time.Second, result.Timeout)

				// Check static data
				if tt.config.StaticData != nil {
					assert.Equal(t, tt.config.StaticData.Data, result.StaticData)
				} else {
					assert.Nil(t, result.StaticData)
				}
			}
		})
	}
}

func TestConvertMCPConfig(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		config    *configMCP.App
		setupMock func(*configMCP.App)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "nil config",
			id:      "test-id",
			config:  nil,
			wantErr: true,
			errMsg:  "MCP config cannot be nil",
		},
		{
			name:   "nil compiled server",
			id:     "test-id",
			config: &configMCP.App{},
			setupMock: func(cfg *configMCP.App) {
				// Default behavior returns nil compiled server
			},
			wantErr: true,
			errMsg:  "MCP server is nil",
		},
		{
			name:   "successful conversion",
			id:     "test-id",
			config: &configMCP.App{},
			setupMock: func(cfg *configMCP.App) {
				// Create a mock MCP server and set it in the config
				// We need to first validate the config to compile the server
				cfg.ID = "test-id"
				cfg.ServerName = "Test Server"
				cfg.ServerVersion = "1.0.0"
				cfg.Transport = &configMCP.Transport{}
				cfg.Tools = []*configMCP.Tool{}
				cfg.Resources = []*configMCP.Resource{}
				cfg.Prompts = []*configMCP.Prompt{}
				cfg.Middlewares = []*configMCP.Middleware{}

				err := cfg.Validate()
				require.NoError(t, err)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(tt.config)
			}

			result, err := convertMCPConfig(tt.id, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
				// Check for specific sentinel error
				if tt.errMsg == "MCP server is nil" {
					require.ErrorIs(t, err, mcp.ErrServerNotCompiled)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
				assert.NotNil(t, result.CompiledServer)
			}
		})
	}
}

func TestConvertDomainToServerApp(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		config     apps.AppConfig
		setupMock  func() apps.AppConfig
		wantErr    bool
		errMsg     string
		wantAppStr string
	}{
		{
			name:       "echo app",
			id:         "echo-test",
			config:     &configEcho.EchoApp{Response: "test response"},
			wantErr:    false,
			wantAppStr: "echo-test",
		},
		{
			name:    "script app with nil evaluator",
			id:      "script-test",
			config:  &configScripts.AppScript{},
			wantErr: true,
			errMsg:  "script app must have an evaluator",
		},
		{
			name: "script app with valid evaluator",
			id:   "script-test",
			setupMock: func() apps.AppConfig {
				scriptConfig := &configScripts.AppScript{}
				mockPlatformEvaluator := &mockPlatformEvaluator{}
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(mockPlatformEvaluator, nil)
				mockEvaluator.On("GetTimeout").Return(30 * time.Second)
				scriptConfig.Evaluator = mockEvaluator
				return scriptConfig
			},
			wantErr:    false,
			wantAppStr: "script-test",
		},
		{
			name:    "mcp app with nil server",
			id:      "mcp-test",
			config:  &configMCP.App{},
			wantErr: true,
			errMsg:  "MCP server is nil",
		},
		{
			name: "mcp app with valid server",
			id:   "mcp-test",
			setupMock: func() apps.AppConfig {
				mcpConfig := &configMCP.App{}
				mcpConfig.ID = "mcp-test"
				mcpConfig.ServerName = "Test Server"
				mcpConfig.ServerVersion = "1.0.0"
				mcpConfig.Transport = &configMCP.Transport{}
				mcpConfig.Tools = []*configMCP.Tool{}
				mcpConfig.Resources = []*configMCP.Resource{}
				mcpConfig.Prompts = []*configMCP.Prompt{}
				mcpConfig.Middlewares = []*configMCP.Middleware{}

				err := mcpConfig.Validate()
				require.NoError(t, err)
				return mcpConfig
			},
			wantErr:    false,
			wantAppStr: "mcp-test",
		},
		{
			name:    "unknown app type",
			id:      "unknown",
			config:  &unknownAppConfig{},
			wantErr: true,
			errMsg:  "unknown app type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			if tt.setupMock != nil {
				config = tt.setupMock()
			}

			result, err := convertDomainToServerApp(tt.id, config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantAppStr, result.String())
			}
		})
	}
}

func TestConvertAndCreateApps_DuplicateIDs(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "duplicate app ID in routes",
			config: &config.Config{
				Endpoints: endpoints.NewEndpointCollection(endpoints.Endpoint{
					ID:         "test-endpoint",
					ListenerID: "test-listener",
					Routes: routes.RouteCollection{
						routes.Route{
							AppID:     "duplicate-id",
							Condition: conditions.NewHTTP("/test1", "GET"),
							App: &apps.App{
								ID:     "duplicate-id",
								Config: &configEcho.EchoApp{Response: "first"},
							},
						},
						routes.Route{
							AppID:     "duplicate-id",
							Condition: conditions.NewHTTP("/test2", "GET"),
							App: &apps.App{
								ID:     "duplicate-id",
								Config: &configEcho.EchoApp{Response: "second"},
							},
						},
					},
				}),
			},
			wantErr: true,
			errMsg:  "duplicate app ID in routes: duplicate-id",
		},
		{
			name: "duplicate app ID in apps collection",
			config: &config.Config{
				Apps: createAppCollection(t, []apps.App{
					{
						ID:     "duplicate-in-apps",
						Config: &configEcho.EchoApp{Response: "first app"},
					},
					{
						ID:     "duplicate-in-apps",
						Config: &configEcho.EchoApp{Response: "second app"},
					},
				}),
			},
			wantErr: true,
			errMsg:  "duplicate app ID: duplicate-in-apps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertAndCreateApps(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// unknownAppConfig is a test helper for testing unknown app types
type unknownAppConfig struct{}

func (u *unknownAppConfig) Type() string                 { return "unknown" }
func (u *unknownAppConfig) Validate() error              { return nil }
func (u *unknownAppConfig) ToProto() any                 { return nil }
func (u *unknownAppConfig) String() string               { return "unknown-config" }
func (u *unknownAppConfig) ToTree() *fancy.ComponentTree { return nil }

// createAppCollection creates an AppCollection for testing
func createAppCollection(t *testing.T, appsList []apps.App) *apps.AppCollection {
	t.Helper()
	return apps.NewAppCollection(appsList...)
}
