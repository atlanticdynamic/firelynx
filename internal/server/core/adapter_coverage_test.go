package core

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	http "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigAdapterWithNilLogger(t *testing.T) {
	// Test that NewConfigAdapter with nil logger uses default logger
	adapter := NewConfigAdapter(nil, nil, nil)
	assert.NotNil(t, adapter.logger)
}

func TestSetDomainConfig(t *testing.T) {
	// Create a test adapter
	adapter := NewConfigAdapter(nil, nil, nil)

	// Create a test domain config
	testConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "test-listener",
				Address: "localhost:8080",
				Options: options.HTTP{},
			},
		},
	}

	// Set the domain config
	adapter.SetDomainConfig(testConfig)

	// Verify the domain config was set
	assert.Equal(t, testConfig, adapter.domainConfig)
}

func TestRoutingConfigCallbackWithNilConfig(t *testing.T) {
	// Create an adapter with nil domain config
	adapter := NewConfigAdapter(nil, nil, nil)

	// Get the routing config callback
	callback := adapter.RoutingConfigCallback()

	// Execute the callback
	config, err := callback()

	// Verify we get an empty config without error
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.EndpointRoutes)
}

func TestHTTPConfigCallbackWithNilConfig(t *testing.T) {
	// Setup
	appRegistry := newMockAppRegistry()
	adapter := NewConfigAdapter(nil, appRegistry, nil)
	routeRegistry := routing.NewRegistry(appRegistry, adapter.RoutingConfigCallback(), nil)

	// Get the HTTP config callback
	callback := adapter.HTTPConfigCallback(routeRegistry)

	// Execute the callback
	config, err := callback()

	// Verify we get a valid config without error
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, routeRegistry, config.RouteRegistry)
	assert.Empty(t, config.Listeners)
}

func TestConvertToHTTPConfigWithNoEndpoints(t *testing.T) {
	// Setup
	domainListeners := listeners.ListenerCollection{
		{
			ID:      "http-listener",
			Address: "localhost:8080",
			Options: options.HTTP{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  10 * time.Second,
				DrainTimeout: 30 * time.Second,
			},
		},
	}

	appRegistry := newMockAppRegistry()
	routeRegistry := routing.NewRegistry(appRegistry, nil, nil)

	// Create adapter with logger to test logging path
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewConfigAdapter(nil, appRegistry, logger)

	// Convert to HTTP config
	config, err := adapter.ConvertToHTTPConfig(domainListeners, routeRegistry)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, routeRegistry, config.RouteRegistry)

	// Since there are no endpoints associated with the listener, it should be skipped
	assert.Empty(t, config.Listeners)
}

func TestConvertToHTTPConfigWithFullConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := newMockAppRegistry()
	routeRegistry := routing.NewRegistry(appRegistry, nil, nil)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Test with full config
	config, err := adapter.ConvertToHTTPConfig(domainConfig.Listeners, routeRegistry)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, routeRegistry, config.RouteRegistry)

	// Should have HTTP listeners
	assert.Len(t, config.Listeners, 2) // http-main and http-admin

	// Verify first listener properties
	assert.Equal(t, "http-main", config.Listeners[0].ID)
	assert.Equal(t, "127.0.0.1:8000", config.Listeners[0].Address)
	assert.Equal(t, "main-api", config.Listeners[0].EndpointID)
	assert.Equal(t, 1*time.Minute, config.Listeners[0].ReadTimeout)
	assert.Equal(t, 1*time.Minute, config.Listeners[0].WriteTimeout)
	assert.Equal(t, 1*time.Minute, config.Listeners[0].IdleTimeout)
	assert.Equal(t, 10*time.Minute, config.Listeners[0].DrainTimeout)

	// Verify second listener properties
	assert.Equal(t, "http-admin", config.Listeners[1].ID)
	assert.Equal(t, "127.0.0.1:8001", config.Listeners[1].Address)
	assert.Equal(t, "admin-api", config.Listeners[1].EndpointID)
	assert.Equal(t, 30*time.Second, config.Listeners[1].ReadTimeout)
	assert.Equal(t, 30*time.Second, config.Listeners[1].WriteTimeout)
	assert.Equal(t, 30*time.Second, config.Listeners[1].IdleTimeout)
	assert.Equal(t, 5*time.Minute, config.Listeners[1].DrainTimeout)
}

func TestConvertToHTTPConfigWithNonHTTPListeners(t *testing.T) {
	// Setup with a non-HTTP listener
	domainListeners := listeners.ListenerCollection{
		{
			ID:      "grpc-listener",
			Address: "localhost:9000",
			Options: options.GRPC{}, // Not HTTP
		},
	}

	// Create domain config with an endpoint that references this listener
	domainConfig := &config.Config{
		Listeners: domainListeners,
		Endpoints: endpoints.EndpointCollection{
			{
				ID:          "grpc-api",
				ListenerIDs: []string{"grpc-listener"},
			},
		},
	}

	appRegistry := newMockAppRegistry()
	routeRegistry := routing.NewRegistry(appRegistry, nil, nil)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Convert to HTTP config
	config, err := adapter.ConvertToHTTPConfig(domainListeners, routeRegistry)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, routeRegistry, config.RouteRegistry)

	// No HTTP listeners should be found
	assert.Empty(t, config.Listeners)
}

func TestConvertToHTTPConfigWithMissingTimeouts(t *testing.T) {
	// Setup with zero timeouts that should get default values
	domainListeners := listeners.ListenerCollection{
		{
			ID:      "http-listener",
			Address: "localhost:8080",
			Options: options.HTTP{
				// All timeouts are zero
			},
		},
	}

	// Create domain config with an endpoint that references this listener
	domainConfig := &config.Config{
		Listeners: domainListeners,
		Endpoints: endpoints.EndpointCollection{
			{
				ID:          "http-api",
				ListenerIDs: []string{"http-listener"},
			},
		},
	}

	appRegistry := newMockAppRegistry()
	routeRegistry := routing.NewRegistry(appRegistry, nil, nil)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Convert to HTTP config
	config, err := adapter.ConvertToHTTPConfig(domainListeners, routeRegistry)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, appRegistry, config.AppRegistry)
	assert.Equal(t, routeRegistry, config.RouteRegistry)

	// Should have one HTTP listener with default timeouts
	require.Len(t, config.Listeners, 1)
	assert.Equal(t, "http-listener", config.Listeners[0].ID)
	assert.Equal(t, "localhost:8080", config.Listeners[0].Address)
	assert.Equal(t, "http-api", config.Listeners[0].EndpointID)

	// Should have default timeouts
	assert.Equal(t, http.DefaultReadTimeout, config.Listeners[0].ReadTimeout)
	assert.Equal(t, http.DefaultWriteTimeout, config.Listeners[0].WriteTimeout)
	assert.Equal(t, http.DefaultIdleTimeout, config.Listeners[0].IdleTimeout)
	assert.Equal(t, http.DefaultDrainTimeout, config.Listeners[0].DrainTimeout)
}
