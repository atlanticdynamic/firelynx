package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configComposite "github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcpserver"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcpserver"
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
	t.Run("nil config", func(t *testing.T) {
		result, err := convertMCPConfig("test-id", nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrConfigNil)
		assert.Nil(t, result)
	})

	t.Run("empty config", func(t *testing.T) {
		result, err := convertMCPConfig("test-id", &configMCP.App{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test-id", result.ID)
		assert.Empty(t, result.Tools)
		assert.Empty(t, result.Prompts)
		assert.Empty(t, result.Resources)
	})

	t.Run("preserves per-primitive refs and schema overrides", func(t *testing.T) {
		domain := &configMCP.App{
			ID: "mcp",
			Tools: []configMCP.Tool{
				{
					ID:    "calculate",
					AppID: "calc-app",
				},
				{
					AppID: "unit-converter-app",
				},
			},
			Prompts: []configMCP.Prompt{
				{
					ID:    "greeting",
					AppID: "echo-app",
				},
			},
			Resources: []configMCP.Resource{
				{
					ID:          "workspace",
					AppID:       "file-reader",
					URITemplate: "file://{path}",
				},
			},
		}

		result, err := convertMCPConfig("mcp", domain)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "mcp", result.ID)

		require.Len(t, result.Tools, 2)
		assert.Equal(t, "calculate", result.Tools[0].ID)
		assert.Equal(t, "calc-app", result.Tools[0].AppID)
		assert.Empty(t, result.Tools[1].ID)
		assert.Equal(t, "unit-converter-app", result.Tools[1].AppID)

		require.Len(t, result.Prompts, 1)
		assert.Equal(t, "greeting", result.Prompts[0].ID)
		assert.Equal(t, "echo-app", result.Prompts[0].AppID)

		require.Len(t, result.Resources, 1)
		assert.Equal(t, "workspace", result.Resources[0].ID)
		assert.Equal(t, "file-reader", result.Resources[0].AppID)
		assert.Equal(t, "file://{path}", result.Resources[0].URITemplate)
	})
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
			name:       "mcp app basic config",
			id:         "mcp-test",
			config:     &configMCP.App{},
			wantErr:    false,
			wantAppStr: "mcp-test",
		},
		{
			name: "mcp app with tools config",
			id:   "mcp-test",
			setupMock: func() apps.AppConfig {
				mcpConfig := &configMCP.App{}
				mcpConfig.ID = "mcp-test"
				mcpConfig.Tools = []configMCP.Tool{
					{AppID: "calc-app"},
				}
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

func TestConvertAndCreateApps_MCPCrossRefValidation(t *testing.T) {
	tests := []struct {
		name        string
		apps        []apps.App
		expectedErr error
		errSubstr   string
	}{
		{
			name: "mcp tool ref to existing typed-tool provider succeeds",
			apps: []apps.App{
				{ID: "calc-app", Config: &configEcho.EchoApp{Response: "calc"}}, // echo implements MCPTypedToolProvider
				{ID: "mcp-server", Config: &configMCP.App{
					ID: "mcp-server",
					Tools: []configMCP.Tool{
						{AppID: "calc-app"},
					},
				}},
			},
		},
		{
			name: "mcp tool ref to missing app fails with ErrUnknownAppRef",
			apps: []apps.App{
				{ID: "mcp-server", Config: &configMCP.App{
					ID: "mcp-server",
					Tools: []configMCP.Tool{
						{AppID: "ghost"},
					},
				}},
			},
			expectedErr: mcpserver.ErrUnknownAppRef,
			errSubstr:   "ghost",
		},
		{
			name: "mcp prompt ref fails as unsupported primitive",
			apps: []apps.App{
				{ID: "echo-app", Config: &configEcho.EchoApp{Response: "hi"}},
				{ID: "mcp-server", Config: &configMCP.App{
					ID: "mcp-server",
					Prompts: []configMCP.Prompt{
						{ID: "greeting", AppID: "echo-app"},
					},
				}},
			},
			expectedErr: mcpserver.ErrMCPPrimitiveNotSupported,
			errSubstr:   "prompt registration is not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Apps: createAppCollection(t, tt.apps),
			}

			result, err := convertAndCreateApps(cfg)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
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
