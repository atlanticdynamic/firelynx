package txmgr

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
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

func TestNewConfigAdapterWithNilLogger(t *testing.T) {
	// Test that NewConfigAdapter with nil logger uses default logger
	adapter := NewConfigAdapter(nil, nil, nil)
	assert.NotNil(t, adapter.logger)
}

func TestValidateConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Test
	err := adapter.ValidateConfig()
	// Verify
	if err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}

	// Test with nil config
	adapter.SetDomainConfig(nil)
	err = adapter.ValidateConfig()

	// Should return an error
	if err == nil {
		t.Fatal("Expected error with nil domain config")
	}
}

func TestGetDomainConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Test
	result := adapter.GetDomainConfig()

	// Verify
	if result != domainConfig {
		t.Errorf("GetDomainConfig() = %v, want %v", result, domainConfig)
	}
}

func TestGetAppRegistry(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(domainConfig, appRegistry, nil)

	// Test
	result := adapter.GetAppRegistry()

	// Verify
	if result != appRegistry {
		t.Errorf("GetAppRegistry() = %v, want %v", result, appRegistry)
	}
}

func TestSetDomainConfig(t *testing.T) {
	// Setup
	domainConfig := createTestDomainConfig()
	appRegistry := mocks.NewMockRegistry()
	adapter := NewConfigAdapter(nil, appRegistry, nil)

	// Test
	adapter.SetDomainConfig(domainConfig)

	// Verify
	if adapter.domainConfig != domainConfig {
		t.Errorf("SetDomainConfig() did not set domain config correctly")
	}
}
