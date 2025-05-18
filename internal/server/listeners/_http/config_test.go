package http

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// Create test dependencies
	appRegistry := mocks.NewMockRegistry()
	listeners := []ListenerConfig{
		{
			ID:      "test1",
			Address: ":8080",
		},
	}

	// Test basic creation without options
	config := NewConfig(appRegistry, listeners)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, listeners, config.Listeners)
	assert.NotNil(t, config.logger)

	// Test with custom logger option
	customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	configWithLogger := NewConfig(appRegistry, listeners, WithConfigLogger(customLogger))
	assert.NotNil(t, configWithLogger)
	assert.Equal(t, appRegistry, configWithLogger.AppRegistry)
	assert.Equal(t, listeners, configWithLogger.Listeners)
	assert.Equal(t, customLogger, configWithLogger.logger)

	// Test with nil logger (should use default)
	configWithNilLogger := NewConfig(appRegistry, listeners, WithConfigLogger(nil))
	assert.NotNil(t, configWithNilLogger)
	assert.NotNil(t, configWithNilLogger.logger)

	// Test with route registry
	routeRegistry := &routing.Registry{}
	configWithRouteRegistry := NewConfig(appRegistry, listeners, WithRouteRegistry(routeRegistry))
	assert.NotNil(t, configWithRouteRegistry)
	assert.Equal(t, appRegistry, configWithRouteRegistry.AppRegistry)
	assert.Equal(t, routeRegistry, configWithRouteRegistry.RouteRegistry)
	assert.Equal(t, listeners, configWithRouteRegistry.Listeners)
}

func TestConfig_Validate(t *testing.T) {
	// Create test dependencies
	appRegistry := mocks.NewMockRegistry()
	routeRegistry := &routing.Registry{}
	validListener := ListenerConfig{
		ID:      "test1",
		Address: ":8080",
		Routes: []RouteConfig{
			{
				Path:  "/test",
				AppID: "test-app",
			},
		},
	}

	// Test table-driven for various validation cases
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config with AppRegistry",
			config: &Config{
				AppRegistry: appRegistry,
				Listeners:   []ListenerConfig{validListener},
			},
			wantError: false,
		},
		{
			name: "valid config with RouteRegistry",
			config: &Config{
				RouteRegistry: routeRegistry,
				Listeners:     []ListenerConfig{validListener},
			},
			wantError: false,
		},
		{
			name: "nil registry",
			config: &Config{
				AppRegistry: nil,
				Listeners:   []ListenerConfig{validListener},
			},
			wantError: true,
			errorMsg:  "either AppRegistry or RouteRegistry must be provided",
		},
		{
			name: "empty listeners",
			config: &Config{
				AppRegistry: appRegistry,
				Listeners:   []ListenerConfig{},
			},
			wantError: false, // Empty listeners is valid
		},
		{
			name: "invalid listener",
			config: &Config{
				AppRegistry: appRegistry,
				Listeners: []ListenerConfig{
					{
						// Missing ID and Address
						Routes: []RouteConfig{
							{
								Path:  "/test",
								AppID: "test-app",
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListenerConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      ListenerConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with endpoint ID",
			config: ListenerConfig{
				ID:         "test-listener",
				Address:    "localhost:8080",
				EndpointID: "endpoint1",
			},
			wantErr: false,
		},
		{
			name: "valid config with routes",
			config: ListenerConfig{
				ID:      "test-listener",
				Address: "localhost:8080",
				Routes: []RouteConfig{
					{
						Path:  "/api",
						AppID: "app1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			config: ListenerConfig{
				Address:    "localhost:8080",
				EndpointID: "endpoint1",
			},
			wantErr:     true,
			errContains: "ID cannot be empty",
		},
		{
			name: "missing address",
			config: ListenerConfig{
				ID:         "test-listener",
				EndpointID: "endpoint1",
			},
			wantErr:     true,
			errContains: "address cannot be empty",
		},
		{
			name: "negative timeouts",
			config: ListenerConfig{
				ID:           "test-listener",
				Address:      "localhost:8080",
				EndpointID:   "endpoint1",
				ReadTimeout:  -1 * time.Second,
				WriteTimeout: -1 * time.Second,
				IdleTimeout:  -1 * time.Second,
				DrainTimeout: -1 * time.Second,
			},
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name: "missing both endpoint ID and routes",
			config: ListenerConfig{
				ID:      "test-listener",
				Address: "localhost:8080",
			},
			wantErr:     true,
			errContains: "either EndpointID or Routes must be provided",
		},
		{
			name: "invalid route",
			config: ListenerConfig{
				ID:      "test-listener",
				Address: "localhost:8080",
				Routes: []RouteConfig{
					{
						// Missing path
						AppID: "app1",
					},
				},
			},
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name: "valid with timeouts",
			config: ListenerConfig{
				ID:           "test1",
				Address:      ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  30 * time.Second,
				DrainTimeout: 30 * time.Second,
				Routes: []RouteConfig{
					{
						Path:  "/test",
						AppID: "test-app",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsUsingRouteRegistry(t *testing.T) {
	// Test with route registry
	t.Run("with route registry", func(t *testing.T) {
		config := &Config{
			RouteRegistry: &routing.Registry{},
		}
		assert.True(t, config.IsUsingRouteRegistry())
	})

	// Test without route registry
	t.Run("without route registry", func(t *testing.T) {
		config := &Config{
			AppRegistry: mocks.NewMockRegistry(),
		}
		assert.False(t, config.IsUsingRouteRegistry())
	})
}

func TestRegistryBackwardCompatibility(t *testing.T) {
	mockReg := mocks.NewMockRegistry()
	config := &Config{
		AppRegistry: mockReg,
	}

	// Test that Registry() returns AppRegistry for backward compatibility
	assert.Equal(t, mockReg, config.Registry())
}

func TestRouteConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      RouteConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: RouteConfig{
				Path:  "/api",
				AppID: "app1",
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: RouteConfig{
				AppID: "app1",
			},
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name: "missing app ID",
			config: RouteConfig{
				Path: "/api",
			},
			wantErr:     true,
			errContains: "appID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
