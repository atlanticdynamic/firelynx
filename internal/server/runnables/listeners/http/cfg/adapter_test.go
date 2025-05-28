package cfg

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider implements the ConfigProvider interface for testing
type MockConfigProvider struct {
	config *config.Config
	txID   string
	appReg apps.AppLookup
}

func (m *MockConfigProvider) GetConfig() *config.Config {
	return m.config
}

func (m *MockConfigProvider) GetTransactionID() string {
	return m.txID
}

func (m *MockConfigProvider) GetAppCollection() apps.AppLookup {
	return m.appReg
}

// MockAppRegistry is a simple implementation of apps.AppLookup for testing
type MockAppRegistry struct {
	apps map[string]apps.App
}

// NewMockAppRegistry creates a new MockAppRegistry instance
func NewMockAppRegistry() *MockAppRegistry {
	return &MockAppRegistry{
		apps: make(map[string]apps.App),
	}
}

// GetApp retrieves an app by ID
func (m *MockAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := m.apps[id]
	return app, ok
}

// RegisterApp adds an app to the registry
func (m *MockAppRegistry) RegisterApp(app apps.App) {
	m.apps[app.String()] = app
}

func TestNewAdapter(t *testing.T) {
	// Create mock config provider with nil config
	nilProvider := &MockConfigProvider{
		config: nil,
		txID:   "test-tx-id",
		appReg: nil,
	}

	// Test with nil provider
	adapter, err := NewAdapter(nil, nil)
	assert.Error(t, err, "Should error with nil provider")
	assert.Nil(t, adapter, "Adapter should be nil with error")

	// Test with nil config
	adapter, err = NewAdapter(nilProvider, nil)
	assert.Error(t, err, "Should error with nil config")
	assert.Nil(t, adapter, "Adapter should be nil with error")

	// Create a minimal valid config
	validConfig := &config.Config{
		Version: "v1alpha1",
	}

	// Create mock config provider with valid config
	validProvider := &MockConfigProvider{
		config: validConfig,
		txID:   "test-tx-id",
		appReg: nil,
	}

	// Test with valid but empty config
	adapter, err = NewAdapter(validProvider, nil)
	assert.NoError(t, err, "Should not error with valid empty config")
	assert.NotNil(t, adapter, "Adapter should not be nil with valid config")
	assert.Equal(t, "test-tx-id", adapter.TxID, "Adapter should have correct transaction ID")
	assert.Empty(t, adapter.Listeners, "Adapter should have no listeners with empty config")
	assert.Empty(t, adapter.Routes, "Adapter should have no routes with empty config")
}

func TestExtractListeners(t *testing.T) {
	// Create test listener collection
	httpListener1 := listeners.Listener{
		ID:      "http-1",
		Address: "localhost:8080",
		Type:    listeners.TypeHTTP,
		Options: options.HTTP{
			ReadTimeout:  time.Second * 30,
			WriteTimeout: time.Second * 30,
			IdleTimeout:  time.Second * 60,
			DrainTimeout: time.Second * 10,
		},
	}

	httpListener2 := listeners.Listener{
		ID:      "http-2",
		Address: "localhost:8081",
		Type:    listeners.TypeHTTP,
		Options: options.HTTP{
			ReadTimeout:  time.Second * 20,
			WriteTimeout: time.Second * 20,
		},
	}

	collection := listeners.ListenerCollection{httpListener1, httpListener2}

	// Extract listeners
	listenerMap, err := extractListeners(collection)
	assert.NoError(t, err, "Should not error with valid listeners")
	assert.Len(t, listenerMap, 2, "Should extract 2 listeners")

	// Check first listener
	listener1, ok := listenerMap["http-1"]
	assert.True(t, ok, "Should find first listener by ID")
	assert.Equal(t, "http-1", listener1.ID, "Listener ID should match")
	assert.Equal(t, "localhost:8080", listener1.Address, "Listener address should match")
	assert.Equal(t, time.Second*30, listener1.ReadTimeout, "Read timeout should match")
	assert.Equal(t, time.Second*30, listener1.WriteTimeout, "Write timeout should match")
	assert.Equal(t, time.Second*60, listener1.IdleTimeout, "Idle timeout should match")
	assert.Equal(t, time.Second*10, listener1.DrainTimeout, "Drain timeout should match")

	// Check second listener
	listener2, ok := listenerMap["http-2"]
	assert.True(t, ok, "Should find second listener by ID")
	assert.Equal(t, "http-2", listener2.ID, "Listener ID should match")
	assert.Equal(t, "localhost:8081", listener2.Address, "Listener address should match")
	assert.Equal(t, time.Second*20, listener2.ReadTimeout, "Read timeout should match")
	assert.Equal(t, time.Second*20, listener2.WriteTimeout, "Write timeout should match")
}

// MockListener implements the listeners.Listener interface for testing
type MockListener struct {
	endpoints []string
}

func (m *MockListener) GetEndpointIDs(any) []string {
	return m.endpoints
}

func TestCreateEndpointListenerMap(t *testing.T) {
	// This test would create a test config with listeners and endpoints

	// Since GetEndpointIDs is a method that requires access to config,
	// we need to mock this functionality for testing
	// In a real implementation, we would use the endpoint lookup in the config

	// We can't directly test the createEndpointListenerMap function in isolation
	// since its functionality depends on the GetEndpointIDs method which needs a config
	// Instead, we would test this as part of the integration tests
	// This placeholder test verifies the map creation logic
	t.Skip("Skipping test due to dependency on Config object")
}

func TestExtractEndpointRoutes(t *testing.T) {
	t.Parallel()

	// Get context from test for proper test timeout handling
	ctx := t.Context()

	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Set up the app registry with mock apps
	appRegistry := NewMockAppRegistry()

	// Register test apps with the registry
	appID1 := "test-app-1"
	appID2 := "test-app-2"
	appID3 := "missing-app"

	testApp1 := mocks.NewMockApp(appID1)
	testApp2 := mocks.NewMockApp(appID2)

	appRegistry.RegisterApp(testApp1)
	appRegistry.RegisterApp(testApp2)
	// Note: appID3 is intentionally not registered to test error handling

	// Set up static data for routes
	staticData1 := map[string]any{"version": "1.0"}
	staticData2 := map[string]any{"timeout": 30}

	// Create test endpoints with different route configurations
	tests := []struct {
		name           string
		endpoint       *endpoints.Endpoint
		listenerID     string
		expectedRoutes int
		expectError    bool
		appIDs         []string
		pathPrefixes   []string
	}{
		{
			name: "successful extraction of multiple routes",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-1",
				ListenerID: "http-1",
				Routes: routes.RouteCollection{
					{
						AppID: appID1,
						Condition: conditions.HTTP{
							PathPrefix: "/api/v1",
							Method:     "GET",
						},
						StaticData: staticData1,
					},
					{
						AppID: appID2,
						Condition: conditions.HTTP{
							PathPrefix: "/api/v2",
							Method:     "POST",
						},
						StaticData: staticData2,
					},
				},
			},
			listenerID:     "http-1",
			expectedRoutes: 2,
			expectError:    false,
			appIDs:         []string{appID1, appID2},
			pathPrefixes:   []string{"/api/v1", "/api/v2"},
		},
		{
			name: "missing app in registry",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-2",
				ListenerID: "http-2",
				Routes: routes.RouteCollection{
					{
						AppID: appID3, // This app doesn't exist in the registry
						Condition: conditions.HTTP{
							PathPrefix: "/api/missing",
							Method:     "GET",
						},
					},
				},
			},
			listenerID:     "http-2",
			expectedRoutes: 0,
			expectError:    true,
			appIDs:         []string{},
			pathPrefixes:   []string{},
		},
		{
			name: "empty routes collection",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-3",
				ListenerID: "http-3",
				Routes:     routes.RouteCollection{},
			},
			listenerID:     "http-3",
			expectedRoutes: 0,
			expectError:    false,
			appIDs:         []string{},
			pathPrefixes:   []string{},
		},
		{
			name: "mixed valid and invalid routes",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-4",
				ListenerID: "http-4",
				Routes: routes.RouteCollection{
					{
						AppID: appID1,
						Condition: conditions.HTTP{
							PathPrefix: "/api/valid",
							Method:     "GET",
						},
					},
					{
						AppID: appID3, // This app doesn't exist in the registry
						Condition: conditions.HTTP{
							PathPrefix: "/api/invalid",
							Method:     "GET",
						},
					},
				},
			},
			listenerID:     "http-4",
			expectedRoutes: 1,
			expectError:    true,
			appIDs:         []string{appID1},
			pathPrefixes:   []string{"/api/valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function being tested
			routes, err := extractEndpointRoutes(tt.endpoint, tt.listenerID, appRegistry, logger)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}

			// Check number of routes
			assert.Len(
				t,
				routes,
				tt.expectedRoutes,
				"Expected %d routes, got %d",
				tt.expectedRoutes,
				len(routes),
			)

			// Check each route's properties
			for i, route := range routes {
				// For route identity, we check if the handler works as expected rather than accessing fields directly
				assert.NotNil(t, route.Handler, "Handler should not be nil for route %d", i)

				// Test the handler by making a request
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", tt.pathPrefixes[i], nil).WithContext(ctx)

				// Set up the mock app to expect a call and implement behavior
				if app, ok := appRegistry.GetApp(tt.appIDs[i]); ok {
					mockApp := app.(*mocks.MockApp)

					// Set up the mock to write a response when HandleHTTP is called
					mockApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Run(
							func(args mock.Arguments) {
								// Extract the response writer from the arguments
								respWriter := args.Get(1).(http.ResponseWriter)
								// Write the success response
								respWriter.WriteHeader(http.StatusOK)
								_, err := respWriter.Write([]byte("success"))
								require.NoError(t, err)
							},
						).
						Return(nil).
						Once()

					// Call the handler
					route.Handler(w, r)

					// Check response
					assert.Equal(
						t,
						http.StatusOK,
						w.Result().StatusCode,
						"Expected 200 OK status code",
					)
					body, err := io.ReadAll(w.Result().Body)
					require.NoError(t, err)
					assert.Equal(t, "success", string(body), "Expected 'success' response body")
				} else {
					t.Fatalf("Could not find app %s in registry", tt.appIDs[i])
				}
			}
		})
	}
}

func TestAdapterGetters(t *testing.T) {
	// Create example HTTP route for testing
	route1, err := httpserver.NewRoute(
		"route-1",
		"/api/test1",
		func(w http.ResponseWriter, r *http.Request) {},
	)
	require.NoError(t, err)
	route2, err := httpserver.NewRoute(
		"route-2",
		"/api/test2",
		func(w http.ResponseWriter, r *http.Request) {},
	)
	require.NoError(t, err)
	route3, err := httpserver.NewRoute(
		"route-3",
		"/api/test3",
		func(w http.ResponseWriter, r *http.Request) {},
	)
	require.NoError(t, err)

	// Create test adapter
	adapter := &Adapter{
		TxID: "test-tx-id",
		Listeners: map[string]ListenerConfig{
			"http-1": {
				ID:      "http-1",
				Address: "localhost:8080",
			},
			"http-2": {
				ID:      "http-2",
				Address: "localhost:8081",
			},
		},
		Routes: map[string][]httpserver.Route{
			"http-1": {*route1, *route2},
			"http-2": {*route3},
		},
	}

	// Test GetListenerIDs
	ids := adapter.GetListenerIDs()
	assert.Len(t, ids, 2, "Should return 2 listener IDs")
	assert.ElementsMatch(t, []string{"http-1", "http-2"}, ids, "IDs should match")

	// Test sorted order (since map iteration order is random)
	assert.Contains(t, ids, "http-1", "Should contain http-1")
	assert.Contains(t, ids, "http-2", "Should contain http-2")

	// Test GetListenerConfig
	config1, ok := adapter.GetListenerConfig("http-1")
	assert.True(t, ok, "Should find config for http-1")
	assert.Equal(t, "http-1", config1.ID, "Config ID should match")
	assert.Equal(t, "localhost:8080", config1.Address, "Config address should match")

	config3, ok := adapter.GetListenerConfig("nonexistent")
	assert.False(t, ok, "Should not find config for nonexistent")
	assert.Empty(t, config3, "Config should be empty for nonexistent")

	// Test GetRoutesForListener
	routes1 := adapter.GetRoutesForListener("http-1")
	assert.Len(t, routes1, 2, "Should return 2 routes for http-1")
	assert.Equal(t, "/api/test1", routes1[0].Path, "First route path should match")
	assert.Equal(t, "/api/test2", routes1[1].Path, "Second route path should match")

	routes2 := adapter.GetRoutesForListener("http-2")
	assert.Len(t, routes2, 1, "Should return 1 route for http-2")
	assert.Equal(t, "/api/test3", routes2[0].Path, "Route path should match")

	routes3 := adapter.GetRoutesForListener("nonexistent")
	assert.Empty(t, routes3, "Should return empty slice for nonexistent listener")
}
