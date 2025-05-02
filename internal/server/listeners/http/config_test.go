package http

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	// Create test dependencies
	registry := mocks.NewMockRegistry()
	listeners := []ListenerConfig{
		{
			ID:      "test1",
			Address: ":8080",
		},
	}

	// Test basic creation without options
	config := NewConfig(registry, listeners)
	assert.NotNil(t, config)
	assert.Equal(t, registry, config.Registry)
	assert.Equal(t, listeners, config.Listeners)
	assert.NotNil(t, config.logger)

	// Test with custom logger option
	customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	configWithLogger := NewConfig(registry, listeners, WithConfigLogger(customLogger))
	assert.NotNil(t, configWithLogger)
	assert.Equal(t, registry, configWithLogger.Registry)
	assert.Equal(t, listeners, configWithLogger.Listeners)
	assert.Equal(t, customLogger, configWithLogger.logger)

	// Test with nil logger (should use default)
	configWithNilLogger := NewConfig(registry, listeners, WithConfigLogger(nil))
	assert.NotNil(t, configWithNilLogger)
	assert.NotNil(t, configWithNilLogger.logger)
}

func TestConfig_Validate(t *testing.T) {
	// Create test dependencies
	registry := mocks.NewMockRegistry()
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
			name: "valid config",
			config: &Config{
				Registry:  registry,
				Listeners: []ListenerConfig{validListener},
			},
			wantError: false,
		},
		{
			name: "nil registry",
			config: &Config{
				Registry:  nil,
				Listeners: []ListenerConfig{validListener},
			},
			wantError: true,
			errorMsg:  "registry cannot be nil",
		},
		{
			name: "empty listeners",
			config: &Config{
				Registry:  registry,
				Listeners: []ListenerConfig{},
			},
			wantError: false, // Empty listeners is valid
		},
		{
			name: "invalid listener",
			config: &Config{
				Registry: registry,
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
			errorMsg:  "invalid listener at index 0",
		},
		{
			name: "invalid listener timeouts",
			config: &Config{
				Registry: registry,
				Listeners: []ListenerConfig{
					{
						ID:           "test1",
						Address:      ":8080",
						ReadTimeout:  -1 * time.Second, // Negative timeout
						WriteTimeout: -1 * time.Second,
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
			errorMsg:  "invalid read timeout",
		},
		{
			name: "nil logger",
			config: &Config{
				Registry:  registry,
				Listeners: []ListenerConfig{validListener},
				logger:    nil,
			},
			wantError: false, // Should not cause validation failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListenerConfig_Validate(t *testing.T) {
	validRoute := RouteConfig{
		Path:  "/test",
		AppID: "test-app",
	}

	// Test valid config
	listener := ListenerConfig{
		ID:      "test1",
		Address: ":8080",
		Routes:  []RouteConfig{validRoute},
	}
	err := listener.Validate()
	assert.NoError(t, err)

	// Test missing ID
	invalidListener := ListenerConfig{
		Address: ":8080",
		Routes:  []RouteConfig{validRoute},
	}
	err = invalidListener.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID cannot be empty")

	// Test missing address
	invalidListener = ListenerConfig{
		ID:     "test1",
		Routes: []RouteConfig{validRoute},
	}
	err = invalidListener.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address cannot be empty")

	// Test negative timeouts
	invalidListener = ListenerConfig{
		ID:           "test1",
		Address:      ":8080",
		ReadTimeout:  -1 * time.Second,
		WriteTimeout: -1 * time.Second,
		DrainTimeout: -1 * time.Second,
		IdleTimeout:  -1 * time.Second,
		Routes:       []RouteConfig{validRoute},
	}
	err = invalidListener.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid read timeout")
	assert.Contains(t, err.Error(), "invalid write timeout")
	assert.Contains(t, err.Error(), "invalid drain timeout")
	assert.Contains(t, err.Error(), "invalid idle timeout")

	// Test with invalid route
	invalidListener = ListenerConfig{
		ID:      "test1",
		Address: ":8080",
		Routes: []RouteConfig{
			{
				// Missing Path and AppID
			},
		},
	}
	err = invalidListener.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid route at index 0")
}

func TestRouteConfig_Validate(t *testing.T) {
	// Create a valid route
	route := RouteConfig{
		Path:  "/test",
		AppID: "test-app",
		StaticData: map[string]any{
			"key": "value",
		},
	}
	err := route.Validate()
	assert.NoError(t, err)

	// Test missing path
	invalidRoute := RouteConfig{
		AppID: "test-app",
	}
	err = invalidRoute.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path cannot be empty")

	// Test missing appID
	invalidRoute = RouteConfig{
		Path: "/test",
	}
	err = invalidRoute.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "appID cannot be empty")

	// Test missing both
	invalidRoute = RouteConfig{}
	err = invalidRoute.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path cannot be empty")
	assert.Contains(t, err.Error(), "appID cannot be empty")
}

func TestWithConfigLogger(t *testing.T) {
	// Create a test Config
	config := &Config{}

	// Create a custom logger
	customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Apply the WithConfigLogger option
	option := WithConfigLogger(customLogger)
	option(config)

	// Verify the logger was set
	assert.Equal(t, customLogger, config.logger)

	// Test with nil logger (should not change existing logger)
	existingLogger := config.logger
	nilOption := WithConfigLogger(nil)
	nilOption(config)
	assert.Equal(t, existingLogger, config.logger)
}
