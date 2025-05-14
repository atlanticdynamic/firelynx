package core

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	serverhttp "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// createTestDomainConfig creates a test domain config with endpoints and listeners
func createTestDomainConfig() *config.Config {
	// Create test HTTP routes
	route1 := routes.Route{
		AppID:      "echo-app",
		Condition:  conditions.NewHTTP("/api/echo", "GET"),
		StaticData: map[string]any{"version": "1.0"},
	}

	route2 := routes.Route{
		AppID:      "script-app",
		Condition:  conditions.NewHTTP("/api/script", "GET"),
		StaticData: map[string]any{"debug": true},
	}

	// Create test endpoints
	endpoint1 := endpoints.Endpoint{
		ID:         "main-api",
		ListenerID: "http-main",
		Routes:     routes.RouteCollection{route1, route2},
	}

	endpoint2 := endpoints.Endpoint{
		ID:         "admin-api",
		ListenerID: "http-admin",
		Routes: routes.RouteCollection{
			{
				AppID:      "admin-app",
				Condition:  conditions.NewHTTP("/admin", "GET"),
				StaticData: map[string]any{"role": "admin"},
			},
		},
	}

	// Create HTTP listeners
	httpMainListener := listeners.Listener{
		ID:      "http-main",
		Address: "127.0.0.1:8000",
		Options: options.HTTP{
			ReadTimeout:  1 * time.Minute,
			WriteTimeout: 1 * time.Minute,
			IdleTimeout:  1 * time.Minute,
			DrainTimeout: 10 * time.Minute,
		},
	}

	httpAdminListener := listeners.Listener{
		ID:      "http-admin",
		Address: "127.0.0.1:8001",
		Options: options.HTTP{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  30 * time.Second,
			DrainTimeout: 5 * time.Minute,
		},
	}

	// Create domain config
	return &config.Config{
		Version:   "v1",
		Endpoints: endpoints.EndpointCollection{endpoint1, endpoint2},
		Listeners: listeners.ListenerCollection{httpMainListener, httpAdminListener},
	}
}

func TestNewConfigAdapter(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test
	adapter := NewConfigAdapter(domainConfig, appRegistry, logger)

	// Verify
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
	if adapter.domainConfig != domainConfig {
		t.Errorf("Expected domainConfig to be set")
	}
	if adapter.appCollection != appRegistry {
		t.Errorf("Expected appCollection to be set")
	}
	if adapter.logger != logger {
		t.Errorf("Expected logger to be set")
	}
}

func TestConfigAdapter_ConvertToRoutingConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Create expected result
	expected := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "main-api",
				Routes: []routing.Route{
					{
						Path:       "/api/echo",
						AppID:      "echo-app",
						StaticData: map[string]any{"version": "1.0"},
					},
					{
						Path:       "/api/script",
						AppID:      "script-app",
						StaticData: map[string]any{"debug": true},
					},
				},
			},
			{
				EndpointID: "admin-api",
				Routes: []routing.Route{
					{
						Path:       "/admin",
						AppID:      "admin-app",
						StaticData: map[string]any{"role": "admin"},
					},
				},
			},
		},
	}

	// Test
	result, err := adapter.ConvertToRoutingConfig(domainConfig.Endpoints)
	// Verify
	if err != nil {
		t.Fatalf("ConvertToRoutingConfig() error = %v", err)
	}

	// Check that the structure matches
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ConvertToRoutingConfig() = %v, want %v", result, expected)
	}
}

func TestConfigAdapter_ConvertToRoutingConfig_EmptyEndpoints(t *testing.T) {
	// Setup with empty endpoints
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(&config.Config{
		Endpoints: endpoints.EndpointCollection{},
	}, appRegistry, nil)

	// Test
	result, err := adapter.ConvertToRoutingConfig(endpoints.EndpointCollection{})
	// Verify
	if err != nil {
		t.Fatalf("ConvertToRoutingConfig() error = %v", err)
	}

	if len(result.EndpointRoutes) != 0 {
		t.Errorf("Expected empty EndpointRoutes, got %v", result.EndpointRoutes)
	}
}

func TestConfigAdapter_RoutingConfigCallback(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Get the callback function
	callback := adapter.RoutingConfigCallback()

	// Test
	config, err := callback()
	// Verify
	if err != nil {
		t.Fatalf("RoutingConfigCallback() error = %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Check that it has the expected endpoints
	if len(config.EndpointRoutes) != 2 {
		t.Errorf("Expected 2 endpoint routes, got %d", len(config.EndpointRoutes))
	}

	// Check contents of the first endpoint
	if len(config.EndpointRoutes) > 0 {
		endpoint := config.EndpointRoutes[0]
		if endpoint.EndpointID != "main-api" {
			t.Errorf("Expected endpoint ID main-api, got %s", endpoint.EndpointID)
		}

		if len(endpoint.Routes) != 2 {
			t.Errorf("Expected 2 routes, got %d", len(endpoint.Routes))
		}
	}

	// Set nil domain config and test again
	adapter.SetDomainConfig(nil)
	config, err = callback()
	// Should still work but return empty config
	if err != nil {
		t.Fatalf("RoutingConfigCallback() with nil domain config error = %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config even with nil domain config")
	}

	if len(config.EndpointRoutes) != 0 {
		t.Errorf(
			"Expected empty EndpointRoutes with nil domain config, got %v",
			config.EndpointRoutes,
		)
	}
}

func TestConfigAdapter_ConvertToHTTPConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()

	// Setup the mock app and registry behavior for all apps used in the test
	echoApp := mocks.NewMockApp("echo-app")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	appRegistry.On("GetApp", "echo-app").Return(echoApp, true)

	scriptApp := mocks.NewMockApp("script-app")
	scriptApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	appRegistry.On("GetApp", "script-app").Return(scriptApp, true)

	adminApp := mocks.NewMockApp("admin-app")
	adminApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	appRegistry.On("GetApp", "admin-app").Return(adminApp, true)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Create route registry for test
	routingCallback := adapter.RoutingConfigCallback()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	routeRegistry := routing.NewRegistry(appRegistry, routingCallback, logger)

	// Initialize registry
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Test ConvertToHTTPConfig
	httpConfig, err := adapter.ConvertToHTTPConfig(domainConfig.Listeners, routeRegistry)
	require.NoError(t, err)

	// Verify the config
	assert.NotNil(t, httpConfig)
	assert.Equal(t, appRegistry, httpConfig.AppRegistry)
	assert.Equal(t, routeRegistry, httpConfig.RouteRegistry)
	assert.Len(t, httpConfig.Listeners, 2)

	// Check first listener
	mainListener := httpConfig.Listeners[0]
	assert.Equal(t, "http-main", mainListener.ID)
	assert.Equal(t, "127.0.0.1:8000", mainListener.Address)
	assert.Equal(t, "main-api", mainListener.EndpointID)
	assert.Equal(t, 1*time.Minute, mainListener.ReadTimeout)
	assert.Equal(t, 1*time.Minute, mainListener.WriteTimeout)
	assert.Equal(t, 1*time.Minute, mainListener.IdleTimeout)
	assert.Equal(t, 10*time.Minute, mainListener.DrainTimeout)

	// Check second listener
	adminListener := httpConfig.Listeners[1]
	assert.Equal(t, "http-admin", adminListener.ID)
	assert.Equal(t, "127.0.0.1:8001", adminListener.Address)
	assert.Equal(t, "admin-api", adminListener.EndpointID)
	assert.Equal(t, 30*time.Second, adminListener.ReadTimeout)
	assert.Equal(t, 30*time.Second, adminListener.WriteTimeout)
	assert.Equal(t, 30*time.Second, adminListener.IdleTimeout)
	assert.Equal(t, 5*time.Minute, adminListener.DrainTimeout)
}

func TestConfigAdapter_HTTPConfigCallback(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()

	// Setup the mock app and registry behavior for all apps used in the test
	echoApp := mocks.NewMockApp("echo-app")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	appRegistry.On("GetApp", "echo-app").Return(echoApp, true)

	scriptApp := mocks.NewMockApp("script-app")
	scriptApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	appRegistry.On("GetApp", "script-app").Return(scriptApp, true)

	adminApp := mocks.NewMockApp("admin-app")
	adminApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	appRegistry.On("GetApp", "admin-app").Return(adminApp, true)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Create route registry for test
	routingCallback := adapter.RoutingConfigCallback()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	routeRegistry := routing.NewRegistry(appRegistry, routingCallback, logger)

	// Initialize registry
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Get HTTP config callback
	httpCallback := adapter.HTTPConfigCallback(routeRegistry)

	// Test callback
	httpConfig, err := httpCallback()
	require.NoError(t, err)

	// Verify the config
	assert.NotNil(t, httpConfig)
	assert.Equal(t, appRegistry, httpConfig.AppRegistry)
	assert.Equal(t, routeRegistry, httpConfig.RouteRegistry)
	assert.Len(t, httpConfig.Listeners, 2)

	// Test callback with nil domain config
	adapter.SetDomainConfig(nil)
	httpConfig, err = httpCallback()
	require.NoError(t, err)

	// Verify empty config
	assert.NotNil(t, httpConfig)
	assert.Equal(t, appRegistry, httpConfig.AppRegistry)
	assert.Equal(t, routeRegistry, httpConfig.RouteRegistry)
	assert.Len(t, httpConfig.Listeners, 0)
}

func TestConfigAdapter_ConvertToHTTPConfig_SkipNonHTTPListeners(t *testing.T) {
	// Setup with a non-HTTP listener
	domainConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "grpc-main",
				Address: "127.0.0.1:9000",
				Options: options.GRPC{}, // non-HTTP option
			},
			{
				ID:      "http-main",
				Address: "127.0.0.1:8000",
				Options: options.HTTP{
					ReadTimeout: 1 * time.Minute,
				},
			},
		},
		// Add endpoints that reference these listeners
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "grpc-api",
				ListenerID: "grpc-main",
			},
			{
				ID:         "main-api",
				ListenerID: "http-main",
			},
		},
	}

	appRegistry := mocks.NewMockRegistry()

	// Setup the mock app and registry behavior
	mainApp := mocks.NewMockApp("main-app")
	mainApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	appRegistry.On("GetApp", "main-app").Return(mainApp, true)

	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)
	routeRegistry := routing.NewRegistry(appRegistry, adapter.RoutingConfigCallback(), nil)

	// Test
	httpConfig, err := adapter.ConvertToHTTPConfig(domainConfig.Listeners, routeRegistry)
	require.NoError(t, err)

	// Should only have the HTTP listener
	assert.Len(t, httpConfig.Listeners, 1)
	assert.Equal(t, "http-main", httpConfig.Listeners[0].ID)
}

func TestConfigAdapter_ConvertToHTTPConfig_DefaultTimeouts(t *testing.T) {
	// Create a listener with zero timeouts
	domainConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http-main",
				Address: "127.0.0.1:8000",
				Options: options.HTTP{}, // Zero timeouts
			},
		},
		// Add endpoint that references this listener
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "main-api",
				ListenerID: "http-main",
			},
		},
	}

	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)
	routeRegistry := routing.NewRegistry(appRegistry, adapter.RoutingConfigCallback(), nil)

	// Test
	httpConfig, err := adapter.ConvertToHTTPConfig(domainConfig.Listeners, routeRegistry)
	require.NoError(t, err)

	// Should have default timeouts
	assert.Len(t, httpConfig.Listeners, 1)
	listener := httpConfig.Listeners[0]
	assert.Equal(t, serverhttp.DefaultReadTimeout, listener.ReadTimeout)
	assert.Equal(t, serverhttp.DefaultWriteTimeout, listener.WriteTimeout)
	assert.Equal(t, serverhttp.DefaultIdleTimeout, listener.IdleTimeout)
	assert.Equal(t, serverhttp.DefaultDrainTimeout, listener.DrainTimeout)
}
