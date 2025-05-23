package routing

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/routing/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to create a mock registry with apps
func setupMockRegistry(appIDs ...string) *mocks.MockRegistry {
	mockRegistry := mocks.NewMockRegistry()

	for _, id := range appIDs {
		mockApp := mocks.NewMockApp(id)
		mockApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil)
		mockRegistry.On("GetApp", id).Return(mockApp, true)
	}

	return mockRegistry
}

func TestNewRegistry(t *testing.T) {
	// Setup
	appRegistry := setupMockRegistry()
	configCallback := func() (*RoutingConfig, error) {
		return &RoutingConfig{}, nil
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test
	registry := NewRegistry(appRegistry, configCallback, logger)

	// Verify
	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}
	if registry.appRegistry != appRegistry {
		t.Errorf("Expected appRegistry to be set")
	}
	if registry.logger != logger {
		t.Errorf("Expected logger to be set")
	}
	if registry.routeTable.Load() == nil {
		t.Errorf("Expected routeTable to be initialized")
	}
}

func TestRegistry_Run(t *testing.T) {
	// Setup
	appRegistry := setupMockRegistry("app1", "app2")

	configCallback := func() (*RoutingConfig, error) {
		return &RoutingConfig{
			EndpointRoutes: []EndpointRoutes{
				{
					EndpointID: "endpoint1",
					Routes: []Route{
						{Path: "/api/v1", AppID: "app1"},
						{Path: "/api/v2", AppID: "app2"},
					},
				},
			},
		}, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	registry := NewRegistry(appRegistry, configCallback, logger)

	// Create a context that cancels after a short time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Run the registry in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- registry.Run(ctx)
	}()

	// Wait for context to be canceled or error to occur
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Run() did not complete in time")
	}

	// Verify the registry is initialized
	if !registry.IsInitialized() {
		t.Errorf("Expected registry to be initialized")
	}
}

func TestRegistry_Run_ConfigError(t *testing.T) {
	// Setup with a failing config callback
	appRegistry := setupMockRegistry()
	configError := errors.New("config error")
	configCallback := func() (*RoutingConfig, error) {
		return nil, configError
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := NewRegistry(appRegistry, configCallback, logger)

	// Run should return an error
	err := registry.Run(context.Background())
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if !errors.Is(err, configError) {
		t.Errorf("Expected error to contain %v, got %v", configError, err)
	}
}

func TestRegistry_Reload(t *testing.T) {
	// Setup
	appRegistry := setupMockRegistry("app1")

	// Use a variable to change the config between calls
	var returnedConfig *RoutingConfig
	configCallback := func() (*RoutingConfig, error) {
		return returnedConfig, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := NewRegistry(appRegistry, configCallback, logger)

	// Initial config
	returnedConfig = &RoutingConfig{
		EndpointRoutes: []EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []Route{
					{Path: "/api/v1", AppID: "app1"},
				},
			},
		},
	}

	// Process initial config
	err := registry.reload()
	if err != nil {
		t.Errorf("reload() error = %v", err)
	}

	// Verify that routes are found
	request := createTestRequest(t, "/api/v1")
	result, err := registry.ResolveRoute("endpoint1", request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "app1", result.AppID)

	// Update config
	returnedConfig = &RoutingConfig{
		EndpointRoutes: []EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []Route{
					{Path: "/api/v1", AppID: "app1"},
					{Path: "/api/v2", AppID: "app1"},
				},
			},
		},
	}

	// Reload
	err = registry.Reload()
	if err != nil {
		t.Errorf("Reload() error = %v", err)
	}

	// Verify that new route is found
	request = createTestRequest(t, "/api/v2")
	result, err = registry.ResolveRoute("endpoint1", request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "app1", result.AppID)
}

func TestRegistry_ResolveRoute(t *testing.T) {
	// Setup app registry with mock apps
	appRegistry := mocks.NewMockRegistry()
	app1 := mocks.NewMockApp("app1")
	app2 := mocks.NewMockApp("app2")

	// Setup mock behaviors
	app1.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	app2.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Setup the registry's GetApp method to return the appropriate app
	appRegistry.On("GetApp", "app1").Return(app1, true)
	appRegistry.On("GetApp", "app2").Return(app2, true)

	// Setup registry with logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a new registry with proper initialization
	registry := NewRegistry(appRegistry, nil, logger)

	// Create route table manually
	routeTable := NewRouteTable()
	routeTable.routes = map[string][]MappedRoute{
		"endpoint1": {
			{
				Route: Route{
					Path:  "/api/v1",
					AppID: "app1",
					StaticData: map[string]any{
						"version": "1.0",
					},
				},
				App:     app1,
				Matcher: matcher.NewHTTPPathMatcher("/api/v1"),
			},
			{
				Route: Route{
					Path:  "/api/v2",
					AppID: "app2",
					StaticData: map[string]any{
						"version": "2.0",
					},
				},
				App:     app2,
				Matcher: matcher.NewHTTPPathMatcher("/api/v2"),
			},
		},
	}

	// Set the route table and mark as initialized
	registry.routeTable.Store(routeTable)
	registry.initialized.Store(true)

	// Test cases
	tests := []struct {
		name       string
		endpointID string
		path       string
		wantAppID  string
		wantErr    bool
		wantNil    bool
	}{
		{
			name:       "exact match for first route",
			endpointID: "endpoint1",
			path:       "/api/v1",
			wantAppID:  "app1",
			wantErr:    false,
			wantNil:    false,
		},
		{
			name:       "exact match for second route",
			endpointID: "endpoint1",
			path:       "/api/v2",
			wantAppID:  "app2",
			wantErr:    false,
			wantNil:    false,
		},
		{
			name:       "prefix match for first route",
			endpointID: "endpoint1",
			path:       "/api/v1/users",
			wantAppID:  "app1",
			wantErr:    false,
			wantNil:    false,
		},
		{
			name:       "no matching route",
			endpointID: "endpoint1",
			path:       "/api/v3",
			wantErr:    false,
			wantNil:    true,
		},
		{
			name:       "non-existent endpoint",
			endpointID: "unknown",
			path:       "/api/v1",
			wantErr:    false,
			wantNil:    true,
		},
		{
			name:       "nil request",
			endpointID: "endpoint1",
			path:       "",
			wantErr:    true,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tt.path != "" {
				req = createTestRequest(t, tt.path)
			}

			// Resolve route
			result, err := registry.ResolveRoute(tt.endpointID, req)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)

			// Check nil result
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			// Check result
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantAppID, result.AppID)

			// For app1, check static data is present
			if tt.wantAppID == "app1" {
				assert.Equal(t, "1.0", result.StaticData["version"])
			}

			// Check that we get a copy of static data
			if result.StaticData != nil {
				original := result.StaticData["version"]
				result.StaticData["version"] = "modified"

				// Resolve again
				newResult, err := registry.ResolveRoute(tt.endpointID, req)
				assert.NoError(t, err)
				assert.NotEqual(t, "modified", newResult.StaticData["version"])
				assert.Equal(t, original, newResult.StaticData["version"])
			}
		})
	}
}

func TestRegistry_ResolveRoute_NotInitialized(t *testing.T) {
	registry := NewRegistry(mocks.NewMockRegistry(), nil, nil)
	// Not initialized yet

	req := createTestRequest(t, "/api/v1")
	result, err := registry.ResolveRoute("endpoint1", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not initialized")
}

// Helper to create a test request
func createTestRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	u, err := url.Parse("http://example.com" + path)
	assert.NoError(t, err, "Failed to parse URL")
	return &http.Request{URL: u, Method: "GET"}
}

// TestRegistry_EndpointListenerConnection tests the routing registry with a
// configuration similar to what's used in the E2E tests.
func TestRegistry_EndpointListenerConnection(t *testing.T) {
	// Setup app registry with mock app
	appRegistry := setupMockRegistry("echo_app")

	// Create a route configuration like the one in E2E tests
	config := &RoutingConfig{
		EndpointRoutes: []EndpointRoutes{
			{
				EndpointID: "echo_endpoint",
				Routes: []Route{
					{
						Path:  "/echo",
						AppID: "echo_app",
					},
				},
			},
		},
	}

	// Create a configuration callback that returns our test config
	configCallback := func() (*RoutingConfig, error) {
		return config, nil
	}

	// Create the registry
	registry := NewRegistry(appRegistry, configCallback, nil)
	require.NotNil(t, registry)

	// Force reload to load our configuration
	err := registry.reload()
	require.NoError(t, err)

	// Verify the registry is initialized
	assert.True(t, registry.IsInitialized())

	// Get the route table and verify it has the expected data
	routeTable := registry.GetRouteTable()
	require.NotNil(t, routeTable)

	// Verify the endpoint routes are correctly registered
	routes := routeTable.GetRoutesForEndpoint("echo_endpoint")
	require.Len(t, routes, 1, "Expected 1 route for echo_endpoint")
	assert.Equal(t, "/echo", routes[0].Route.Path)
	assert.Equal(t, "echo_app", routes[0].Route.AppID)

	// Test route resolution with a request
	req := createTestRequest(t, "/echo")
	result, err := registry.ResolveRoute("echo_endpoint", req)
	require.NoError(t, err)
	require.NotNil(t, result, "Route resolution should succeed for /echo on echo_endpoint")
	assert.Equal(t, "echo_app", result.AppID)
}
