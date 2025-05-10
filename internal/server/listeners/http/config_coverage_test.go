package http

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRegistry implements apps.Registry for testing
type mockRegistry struct {
	apps map[string]apps.App
}

func (r *mockRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *mockRegistry) RegisterApp(app apps.App) error {
	r.apps[app.ID()] = app
	return nil
}

func (r *mockRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with app registry",
			config: &Config{
				AppRegistry: &mockRegistry{apps: make(map[string]apps.App)},
				Listeners: []ListenerConfig{
					{
						ID:         "test-listener",
						Address:    "localhost:8080",
						EndpointID: "endpoint1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with route registry",
			config: &Config{
				RouteRegistry: &routing.Registry{},
				Listeners: []ListenerConfig{
					{
						ID:         "test-listener",
						Address:    "localhost:8080",
						EndpointID: "endpoint1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing both registries",
			config: &Config{
				Listeners: []ListenerConfig{
					{
						ID:         "test-listener",
						Address:    "localhost:8080",
						EndpointID: "endpoint1",
					},
				},
			},
			wantErr:     true,
			errContains: "either AppRegistry or RouteRegistry must be provided",
		},
		{
			name: "invalid listener",
			config: &Config{
				AppRegistry: &mockRegistry{apps: make(map[string]apps.App)},
				Listeners: []ListenerConfig{
					{
						// Missing ID
						Address:    "localhost:8080",
						EndpointID: "endpoint1",
					},
				},
			},
			wantErr:     true,
			errContains: "ID cannot be empty",
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

func TestListenerConfigValidate(t *testing.T) {
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

func TestConfigOptions(t *testing.T) {
	// Test logger option
	t.Run("WithConfigLogger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		config := NewConfig(
			&mockRegistry{apps: make(map[string]apps.App)},
			nil,
			WithConfigLogger(logger),
		)
		assert.Equal(t, logger, config.logger)
	})

	// Test nil logger
	t.Run("WithConfigLogger nil", func(t *testing.T) {
		config := NewConfig(&mockRegistry{apps: make(map[string]apps.App)}, nil)
		// Can't directly compare loggers, so we'll just ensure it's not nil
		assert.NotNil(t, config.logger)
	})

	// Test route registry option
	t.Run("WithRouteRegistry", func(t *testing.T) {
		registry := &routing.Registry{}
		config := NewConfig(
			&mockRegistry{apps: make(map[string]apps.App)},
			nil,
			WithRouteRegistry(registry),
		)
		assert.Equal(t, registry, config.RouteRegistry)
	})
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
			AppRegistry: &mockRegistry{apps: make(map[string]apps.App)},
		}
		assert.False(t, config.IsUsingRouteRegistry())
	})
}

func TestRegistryBackwardCompatibility(t *testing.T) {
	mockReg := &mockRegistry{apps: make(map[string]apps.App)}
	config := &Config{
		AppRegistry: mockReg,
	}

	// Test that Registry() returns AppRegistry for backward compatibility
	assert.Equal(t, mockReg, config.Registry())
}
