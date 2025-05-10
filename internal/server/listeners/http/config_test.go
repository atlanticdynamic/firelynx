package http

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
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
	// Test valid config with routes
	validConfig := ListenerConfig{
		ID:      "test1",
		Address: ":8080",
		Routes: []RouteConfig{
			{
				Path:  "/test",
				AppID: "test-app",
			},
		},
	}
	assert.NoError(t, validConfig.Validate())

	// Test missing ID
	missingID := ListenerConfig{
		Address: ":8080",
	}
	assert.Error(t, missingID.Validate())
	assert.Contains(t, missingID.Validate().Error(), "ID cannot be empty")

	// Test missing address
	missingAddress := ListenerConfig{
		ID: "test1",
		Routes: []RouteConfig{
			{
				Path:  "/test",
				AppID: "test-app",
			},
		},
	}
	assert.Error(t, missingAddress.Validate())
	assert.Contains(t, missingAddress.Validate().Error(), "address cannot be empty")

	// Test with timeouts
	configWithTimeouts := ListenerConfig{
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
	}
	assert.NoError(t, configWithTimeouts.Validate())
}
