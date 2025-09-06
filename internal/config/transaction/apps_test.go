package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configComposite "github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
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
		name        string
		id          string
		config      *configScripts.AppScript
		setupMock   func(*configScripts.AppScript)
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "nil config",
			id:          "test-id",
			config:      nil,
			wantErr:     true,
			expectedErr: ErrConfigNil,
		},
		{
			name:        "nil evaluator",
			id:          "test-id",
			config:      &configScripts.AppScript{},
			wantErr:     true,
			expectedErr: ErrEvaluatorNil,
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
			// We expect the wrapped error from GetCompiledEvaluator, not a specific sentinel
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
			wantErr:     true,
			expectedErr: ErrCompiledEvaluatorNil,
		},
		{
			name:   "successful conversion with minimal config",
			id:     "test-id",
			config: &configScripts.AppScript{},
			setupMock: func(cfg *configScripts.AppScript) {
				mockPlatformEvaluator := &mockPlatformEvaluator{}
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(mockPlatformEvaluator, nil)
				mockEvaluator.On("GetTimeout").Return(testTimeout)
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
				mockEvaluator.On("GetTimeout").Return(testTimeout)
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
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
				assert.NotNil(t, result.CompiledEvaluator)
				assert.NotNil(t, result.Logger)
				assert.Equal(t, testTimeout, result.ExecTimeout)

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
		name        string
		id          string
		config      *configMCP.App
		setupMock   func(*configMCP.App)
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "nil config",
			id:          "test-id",
			config:      nil,
			wantErr:     true,
			expectedErr: ErrConfigNil,
		},
		{
			name:   "nil compiled server",
			id:     "test-id",
			config: &configMCP.App{},
			setupMock: func(cfg *configMCP.App) {
				// Default behavior returns nil compiled server
			},
			wantErr:     true,
			expectedErr: mcp.ErrServerNotCompiled,
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
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
				assert.Nil(t, result)
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
		name        string
		id          string
		config      apps.AppConfig
		setupMock   func() apps.AppConfig
		wantErr     bool
		expectedErr error
		wantAppStr  string
	}{
		{
			name:       "echo app",
			id:         "echo-test",
			config:     &configEcho.EchoApp{Response: "test response"},
			wantErr:    false,
			wantAppStr: "echo-test",
		},
		{
			name:        "script app with nil evaluator",
			id:          "script-test",
			config:      &configScripts.AppScript{},
			wantErr:     true,
			expectedErr: ErrEvaluatorNil,
		},
		{
			name: "script app with valid evaluator",
			id:   "script-test",
			setupMock: func() apps.AppConfig {
				scriptConfig := &configScripts.AppScript{}
				mockPlatformEvaluator := &mockPlatformEvaluator{}
				mockEvaluator := &mockEvaluatorAdapter{}
				mockEvaluator.On("GetCompiledEvaluator").Return(mockPlatformEvaluator, nil)
				mockEvaluator.On("GetTimeout").Return(testTimeout)
				scriptConfig.Evaluator = mockEvaluator
				return scriptConfig
			},
			wantErr:    false,
			wantAppStr: "script-test",
		},
		{
			name:        "mcp app with nil server",
			id:          "mcp-test",
			config:      &configMCP.App{},
			wantErr:     true,
			expectedErr: mcp.ErrServerNotCompiled,
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
			name:        "composite app not supported",
			id:          "composite-test",
			config:      &configComposite.CompositeScript{},
			wantErr:     true,
			expectedErr: ErrCompositeNotSupported,
		},
		{
			name:        "unknown app type",
			id:          "unknown",
			config:      &unknownAppConfig{},
			wantErr:     true,
			expectedErr: ErrUnknownAppType,
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
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
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
		name        string
		config      *config.Config
		wantErr     bool
		expectedErr error
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
			wantErr:     true,
			expectedErr: ErrDuplicateAppID,
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
			wantErr:     true,
			expectedErr: ErrDuplicateAppID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertAndCreateApps(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
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

const testTimeout = 30 * time.Second
