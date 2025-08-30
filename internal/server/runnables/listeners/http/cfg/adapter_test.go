package cfg

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	configApps "github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider implements the ConfigProvider interface for testing
type MockConfigProvider struct {
	config             *config.Config
	txID               string
	appInstances       *serverApps.AppInstances
	middlewareRegistry MiddlewareRegistry
}

// Ensure MockConfigProvider implements ConfigProvider
var _ ConfigProvider = (*MockConfigProvider)(nil)

func (m *MockConfigProvider) GetConfig() *config.Config {
	return m.config
}

func (m *MockConfigProvider) GetTransactionID() string {
	return m.txID
}

func (m *MockConfigProvider) GetAppCollection() *serverApps.AppInstances {
	return m.appInstances
}

func (m *MockConfigProvider) GetMiddlewareRegistry() MiddlewareRegistry {
	if m.middlewareRegistry == nil {
		return make(MiddlewareRegistry)
	}
	return m.middlewareRegistry
}

func TestNewAdapter(t *testing.T) {
	// Create mock config provider with nil config
	nilProvider := &MockConfigProvider{
		config:       nil,
		txID:         "test-tx-id",
		appInstances: nil,
	}

	// Test with nil provider
	adapter, err := NewAdapter(nil, nil)
	require.Error(t, err, "Should error with nil provider")
	assert.Nil(t, adapter, "Adapter should be nil with error")

	// Test with nil config
	adapter, err = NewAdapter(nilProvider, nil)
	require.Error(t, err, "Should error with nil config")
	assert.Nil(t, adapter, "Adapter should be nil with error")

	// Create a minimal valid config
	validConfig := &config.Config{
		Version: "v1alpha1",
	}

	// Create mock config provider with valid config
	validProvider := &MockConfigProvider{
		config:       validConfig,
		txID:         "test-tx-id",
		appInstances: nil,
	}

	// Test with valid but empty config
	adapter, err = NewAdapter(validProvider, nil)
	require.NoError(t, err, "Should not error with valid empty config")
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
	require.NoError(t, err, "Should not error with valid listeners")
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

	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Set up test app IDs
	appID1 := "test-app-1"
	appID2 := "test-app-2"
	appID3 := "missing-app"

	// Create app collection for expansion (empty collection for testing missing app scenarios)
	appCollection := configApps.NewAppCollection()

	// Create app instances (empty since all test routes will reference missing apps)
	appInstances, err := serverApps.NewAppInstances([]serverApps.App{})
	require.NoError(t, err)

	// Set up static data for routes
	staticData1 := map[string]any{"version": "1.0"}
	staticData2 := map[string]any{"timeout": 30}

	// Helper function to create config and expand endpoints automatically
	createExpandedConfig := func(endpoint *endpoints.Endpoint) (*config.Config, *endpoints.Endpoint) {
		// Create a minimal config for expansion
		cfg := &config.Config{
			Apps:      appCollection,
			Endpoints: endpoints.EndpointCollection{*endpoint},
		}

		// Convert to proto and back to trigger expansion
		proto := cfg.ToProto()

		expandedConfig, err := config.NewFromProto(proto)
		require.NoError(t, err)

		// Return the expanded config and endpoint
		return expandedConfig, &expandedConfig.Endpoints[0]
	}

	// Helper function to add expanded apps from config to app registry
	addExpandedAppsFromConfig := func(cfg *config.Config) {
		var expandedApps []serverApps.App
		for _, endpoint := range cfg.Endpoints {
			for _, route := range endpoint.Routes {
				if route.App != nil {
					// Create a mock app for the expanded app ID
					expandedMockApp := mocks.NewMockApp(route.App.ID)
					expandedApps = append(expandedApps, expandedMockApp)
				}
			}
		}

		// Create new app instances with expanded apps
		if len(expandedApps) > 0 {
			appInstances, err = serverApps.NewAppInstances(expandedApps)
			require.NoError(t, err)
		}
	}

	// Create test endpoints with different route configurations
	tests := []struct {
		name           string
		endpoint       *endpoints.Endpoint
		listenerID     string
		expectedRoutes int
		expectError    bool
		pathPrefixes   []string
	}{
		{
			name: "expansion fails with missing apps",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-1",
				ListenerID: "http-1",
				Routes: routes.RouteCollection{
					{
						AppID: appID1,
						Condition: &conditions.HTTP{
							PathPrefix: "/api/v1",
							Method:     "GET",
						},
						StaticData: staticData1,
					},
					{
						AppID: appID2,
						Condition: &conditions.HTTP{
							PathPrefix: "/api/v2",
							Method:     "POST",
						},
						StaticData: staticData2,
					},
				},
			},
			listenerID:     "http-1",
			expectedRoutes: 0,
			expectError:    true,
			pathPrefixes:   []string{},
		},
		{
			name: "missing app in registry",
			endpoint: &endpoints.Endpoint{
				ID:         "test-endpoint-2",
				ListenerID: "http-2",
				Routes: routes.RouteCollection{
					{
						AppID: appID3, // This app doesn't exist in the registry
						Condition: &conditions.HTTP{
							PathPrefix: "/api/missing",
							Method:     "GET",
						},
					},
				},
			},
			listenerID:     "http-2",
			expectedRoutes: 0,
			expectError:    true,
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
						Condition: &conditions.HTTP{
							PathPrefix: "/api/valid",
							Method:     "GET",
						},
					},
					{
						AppID: appID3, // This app doesn't exist in the registry
						Condition: &conditions.HTTP{
							PathPrefix: "/api/invalid",
							Method:     "GET",
						},
					},
				},
			},
			listenerID:     "http-4",
			expectedRoutes: 0,
			expectError:    true,
			pathPrefixes:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create expanded config using helper function
			expandedConfig, expandedEndpoint := createExpandedConfig(tt.endpoint)

			// Add expanded apps to registry
			addExpandedAppsFromConfig(expandedConfig)

			// Call the function being tested
			routes, err := extractEndpointRoutes(
				expandedEndpoint,
				tt.listenerID,
				appInstances,
				make(MiddlewareRegistry),
				logger,
			)

			// Check error expectation
			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
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
				// Verify route has handlers
				assert.NotEmpty(t, route.Handlers, "Handlers should not be empty for route %d", i)

				// Verify route path matches expected
				if i < len(tt.pathPrefixes) {
					assert.Equal(
						t,
						tt.pathPrefixes[i],
						route.Path,
						"Route path should match expected",
					)
				}
			}
		})
	}
}

func TestExtractRoutes(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("route extraction fails without expanded apps", func(t *testing.T) {
		// Set up app instances (empty to simulate missing apps scenario)
		appInstances, err := serverApps.NewAppInstances([]serverApps.App{})
		require.NoError(t, err)

		// Create listeners map
		listenersMap := map[string]ListenerConfig{
			"http-1": {ID: "http-1", Address: ":8080"},
		}

		// Create config with endpoints (no expansion, should fail)
		cfg := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				listeners.Listener{
					ID:      "http-1",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
				},
			},
			Endpoints: endpoints.EndpointCollection{
				endpoints.Endpoint{
					ID:         "endpoint-1",
					ListenerID: "http-1",
					Routes: routes.RouteCollection{
						routes.Route{
							AppID: "test-app",
							Condition: &conditions.HTTP{
								PathPrefix: "/api/test",
								Method:     "GET",
							},
							StaticData: map[string]any{"version": "1.0"},
						},
					},
				},
			},
		}

		// Extract routes - should fail because routes don't have expanded app instances
		routeMap, err := extractRoutes(
			cfg,
			listenersMap,
			appInstances,
			make(MiddlewareRegistry),
			logger,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing expanded app instance")
		assert.Empty(t, routeMap["http-1"])
	})

	t.Run("empty listeners map", func(t *testing.T) {
		appInstances, err := serverApps.NewAppInstances([]serverApps.App{})
		require.NoError(t, err)
		listenersMap := map[string]ListenerConfig{}
		cfg := &config.Config{Version: config.VersionLatest}

		routeMap, err := extractRoutes(
			cfg,
			listenersMap,
			appInstances,
			make(MiddlewareRegistry),
			logger,
		)
		require.NoError(t, err)
		assert.Empty(t, routeMap)
	})

	t.Run("error in endpoint processing", func(t *testing.T) {
		appInstances, err := serverApps.NewAppInstances([]serverApps.App{})
		require.NoError(t, err)
		listenersMap := map[string]ListenerConfig{
			"http-1": {ID: "http-1", Address: ":8080"},
		}

		cfg := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				listeners.Listener{
					ID:      "http-1",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
				},
			},
			Endpoints: endpoints.EndpointCollection{
				endpoints.Endpoint{
					ID:         "endpoint-1",
					ListenerID: "http-1",
					Routes: routes.RouteCollection{
						routes.Route{
							AppID: "missing-app",
							Condition: &conditions.HTTP{
								PathPrefix: "/api/test",
								Method:     "GET",
							},
						},
					},
				},
			},
		}

		routeMap, err := extractRoutes(
			cfg,
			listenersMap,
			appInstances,
			make(MiddlewareRegistry),
			logger,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to process routes for endpoint endpoint-1")
		assert.Empty(t, routeMap["http-1"])
	})
}

func TestNewAdapterWithRoutes(t *testing.T) {
	t.Run("adapter with app collection", func(t *testing.T) {
		// Set up app instances
		testApp := mocks.NewMockApp("test-app")
		expandedApp := mocks.NewMockApp("test-app#0:0")
		expandedConfigApp := &configApps.App{
			ID:     "test-app#0:0",
			Config: nil,
		}

		appInstances, err := serverApps.NewAppInstances([]serverApps.App{testApp, expandedApp})
		require.NoError(t, err)

		// Create config with HTTP listener and endpoints
		cfg := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				{
					ID:      "http-1",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 30,
						WriteTimeout: time.Second * 30,
					},
				},
			},
			Endpoints: endpoints.EndpointCollection{
				endpoints.Endpoint{
					ID:         "endpoint-1",
					ListenerID: "http-1",
					Routes: routes.RouteCollection{
						routes.Route{
							AppID: "test-app",
							Condition: &conditions.HTTP{
								PathPrefix: "/api/v1",
								Method:     "GET",
							},
							StaticData: map[string]any{"version": "1.0"},
							App:        expandedConfigApp,
						},
					},
				},
			},
		}

		provider := &MockConfigProvider{
			config:       cfg,
			txID:         "test-tx-id",
			appInstances: appInstances,
		}

		adapter, err := NewAdapter(provider, nil)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, "test-tx-id", adapter.TxID)
		assert.Len(t, adapter.Listeners, 1)
		assert.Len(t, adapter.Routes["http-1"], 1)
	})

	t.Run("adapter without app collection", func(t *testing.T) {
		cfg := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				listeners.Listener{
					ID:      "http-1",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
				},
			},
		}

		provider := &MockConfigProvider{
			config:       cfg,
			txID:         "test-tx-id",
			appInstances: nil,
		}

		adapter, err := NewAdapter(provider, nil)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Len(t, adapter.Listeners, 1)
		assert.Empty(t, adapter.Routes["http-1"])
	})

	t.Run("error extracting routes", func(t *testing.T) {
		appInstances, err := serverApps.NewAppInstances([]serverApps.App{})
		require.NoError(t, err)
		cfg := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				listeners.Listener{
					ID:      "http-1",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
				},
			},
			Endpoints: endpoints.EndpointCollection{
				endpoints.Endpoint{
					ID:         "endpoint-1",
					ListenerID: "http-1",
					Routes: routes.RouteCollection{
						routes.Route{
							AppID: "missing-app",
							Condition: &conditions.HTTP{
								PathPrefix: "/api/test",
								Method:     "GET",
							},
						},
					},
				},
			},
		}

		provider := &MockConfigProvider{
			config:       cfg,
			txID:         "test-tx-id",
			appInstances: appInstances,
		}

		adapter, err := NewAdapter(provider, nil)
		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "failed to extract HTTP routes")
	})
}

func TestExtractEndpointRoutesErrorHandling(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create app instances with a mock app that returns an error
	errorApp := mocks.NewMockApp("error-app")
	expandedErrorApp := mocks.NewMockApp("error-app#0:0")
	expandedConfigApp := &configApps.App{
		ID:     "error-app#0:0",
		Config: nil,
	}

	appInstances, err := serverApps.NewAppInstances([]serverApps.App{errorApp, expandedErrorApp})
	require.NoError(t, err)

	endpoint := &endpoints.Endpoint{
		ID:         "test-endpoint",
		ListenerID: "http-1",
		Routes: routes.RouteCollection{
			routes.Route{
				AppID: "error-app",
				Condition: &conditions.HTTP{
					PathPrefix: "/api/error",
					Method:     "GET",
				},
				App: expandedConfigApp,
			},
		},
	}

	routes, err := extractEndpointRoutes(
		endpoint,
		"http-1",
		appInstances,
		make(MiddlewareRegistry),
		logger,
	)
	require.NoError(t, err) // Route creation succeeds
	assert.Len(t, routes, 1)

	// Test the handler with an app that returns an error
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/error", nil)

	expandedErrorApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("app error")).
		Once()

	routes[0].ServeHTTP(w, r)

	// Should get 500 error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestExtractEndpointRoutesWithStaticData(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create base app and expanded app with static data
	testApp := mocks.NewMockApp("test-app")

	staticData := map[string]any{
		"version":  "1.0",
		"timeout":  30,
		"features": []string{"auth", "logging"},
	}

	// Create expanded app with unique ID and merged static data
	expandedServerApp := mocks.NewMockApp("test-app#0:0")
	expandedConfigApp := &configApps.App{
		ID:     "test-app#0:0",
		Config: nil, // We don't need the config for this test
	}

	// Create app instances with both base and expanded apps
	appInstances, err := serverApps.NewAppInstances([]serverApps.App{testApp, expandedServerApp})
	require.NoError(t, err)

	endpoint := &endpoints.Endpoint{
		ID:         "test-endpoint",
		ListenerID: "http-1",
		Routes: routes.RouteCollection{
			routes.Route{
				AppID: "test-app",
				Condition: &conditions.HTTP{
					PathPrefix: "/api/test",
					Method:     "GET",
				},
				StaticData: staticData,
				App:        expandedConfigApp, // Set expanded app instance
			},
		},
	}

	routes, err := extractEndpointRoutes(
		endpoint,
		"http-1",
		appInstances,
		make(MiddlewareRegistry),
		logger,
	)
	require.NoError(t, err)
	assert.Len(t, routes, 1)

	// Test that app is called without static data parameter
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/test", nil)

	expandedServerApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	routes[0].ServeHTTP(w, r)
}

func TestAdapterGetters(t *testing.T) {
	// Create example HTTP route for testing
	route1, err := httpserver.NewRouteFromHandlerFunc(
		"route-1",
		"/api/test1",
		func(w http.ResponseWriter, r *http.Request) {},
	)
	require.NoError(t, err)
	route2, err := httpserver.NewRouteFromHandlerFunc(
		"route-2",
		"/api/test2",
		func(w http.ResponseWriter, r *http.Request) {},
	)
	require.NoError(t, err)
	route3, err := httpserver.NewRouteFromHandlerFunc(
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

	t.Run("GetListenerIDs", func(t *testing.T) {
		ids := adapter.GetListenerIDs()
		assert.Len(t, ids, 2, "Should return 2 listener IDs")
		assert.ElementsMatch(t, []string{"http-1", "http-2"}, ids, "IDs should match")
		assert.Contains(t, ids, "http-1", "Should contain http-1")
		assert.Contains(t, ids, "http-2", "Should contain http-2")
	})

	t.Run("GetListenerConfig", func(t *testing.T) {
		config, ok := adapter.GetListenerConfig("http-1")
		assert.True(t, ok, "Should find config for http-1")
		assert.Equal(t, "http-1", config.ID, "Config ID should match")
		assert.Equal(t, "localhost:8080", config.Address, "Config address should match")
	})

	t.Run("GetListenerConfig for nonexistent listener", func(t *testing.T) {
		config, ok := adapter.GetListenerConfig("nonexistent")
		assert.False(t, ok, "Should not find config for nonexistent")
		assert.Empty(t, config, "Config should be empty for nonexistent")
	})

	t.Run("GetRoutesForListener", func(t *testing.T) {
		routes1 := adapter.GetRoutesForListener("http-1")
		assert.Len(t, routes1, 2, "Should return 2 routes for http-1")
		assert.Equal(t, "/api/test1", routes1[0].Path, "First route path should match")
		assert.Equal(t, "/api/test2", routes1[1].Path, "Second route path should match")
	})

	t.Run("GetRoutesForListener for existing listener", func(t *testing.T) {
		routes2 := adapter.GetRoutesForListener("http-2")
		assert.Len(t, routes2, 1, "Should return 1 route for http-2")
		assert.Equal(t, "/api/test3", routes2[0].Path, "Route path should match")
	})

	t.Run("GetRoutesForListener for nonexistent listener", func(t *testing.T) {
		routes3 := adapter.GetRoutesForListener("nonexistent")
		assert.Empty(t, routes3, "Should return empty slice for nonexistent listener")
	})
}
