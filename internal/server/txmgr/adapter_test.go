package txmgr

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
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
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
