package transaction

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				assert.NotNil(t, result)
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
				assert.NotNil(t, result)
			}
		})
	}
}

func TestConvertDomainToServerApp(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		config     apps.AppConfig
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
			name:    "mcp app with nil server",
			id:      "mcp-test",
			config:  &configMCP.App{},
			wantErr: true,
			errMsg:  "MCP server is nil",
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
			result, err := convertDomainToServerApp(tt.id, tt.config)

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
