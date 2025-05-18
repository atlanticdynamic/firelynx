package cfg

import (
	"log/slog"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConfigProvider implements the ConfigProvider interface for testing
type MockConfigProvider struct {
	mock.Mock
}

func (m *MockConfigProvider) GetTransactionID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfigProvider) GetConfig() *config.Config {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*config.Config)
}

func TestNewAdapter_NilProvider(t *testing.T) {
	// Test creating an adapter with a nil provider
	_, err := NewAdapter(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config provider cannot be nil")
}

func TestNewAdapter_NilConfig(t *testing.T) {
	// Set up a mock provider that returns nil config
	provider := new(MockConfigProvider)
	provider.On("GetTransactionID").Return("tx-123")
	provider.On("GetConfig").Return(nil)

	// Test creating an adapter with a provider that returns nil config
	_, err := NewAdapter(provider, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider has no configuration")
}

func TestNewAdapter_EmptyConfig(t *testing.T) {
	// Set up a mock provider with an empty config
	provider := new(MockConfigProvider)
	provider.On("GetTransactionID").Return("tx-123")
	provider.On("GetConfig").Return(&config.Config{})

	// Create a test logger
	logger := slog.Default()

	// Test creating an adapter with an empty config
	adapter, err := NewAdapter(provider, logger)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, "tx-123", adapter.TxID)
	assert.Empty(t, adapter.Listeners)
	assert.Empty(t, adapter.Routes)
}

func TestAdapter_GetListenerIDs(t *testing.T) {
	// Create an adapter with some listeners
	adapter := &Adapter{
		TxID: "tx-123",
		Listeners: map[string]*http.ListenerConfig{
			"listener-1": nil,
			"listener-2": nil,
			"listener-3": nil,
		},
	}

	// Test getting listener IDs
	ids := adapter.GetListenerIDs()
	assert.Len(t, ids, 3)
	assert.ElementsMatch(t, []string{"listener-1", "listener-2", "listener-3"}, ids)

	// Verify deterministic ordering
	assert.Equal(t, []string{"listener-1", "listener-2", "listener-3"}, ids)
}

func TestAdapter_GetListenerConfig(t *testing.T) {
	// Create a test listener config
	listenerConfig := &http.ListenerConfig{
		ID:      "listener-1",
		Address: "localhost:8080",
	}

	// Create an adapter with the test listener
	adapter := &Adapter{
		TxID: "tx-123",
		Listeners: map[string]*http.ListenerConfig{
			"listener-1": listenerConfig,
		},
	}

	// Test getting an existing listener config
	cfg, ok := adapter.GetListenerConfig("listener-1")
	assert.True(t, ok)
	assert.Equal(t, listenerConfig, cfg)

	// Test getting a non-existent listener config
	cfg, ok = adapter.GetListenerConfig("non-existent")
	assert.False(t, ok)
	assert.Nil(t, cfg)
}

func TestAdapter_GetRoutesForListener(t *testing.T) {
	// Create a test route
	testRoute := httpserver.Route{
		Path: "/test",
	}

	// Create an adapter with a listener and route
	adapter := &Adapter{
		TxID: "tx-123",
		Listeners: map[string]*http.ListenerConfig{
			"listener-1": nil,
		},
		Routes: map[string][]httpserver.Route{
			"listener-1": {testRoute},
		},
	}

	// Test getting routes for an existing listener
	routes := adapter.GetRoutesForListener("listener-1")
	assert.Len(t, routes, 1)
	assert.Equal(t, testRoute.Path, routes[0].Path)

	// Test getting routes for a non-existent listener
	routes = adapter.GetRoutesForListener("non-existent")
	assert.Empty(t, routes)
}
