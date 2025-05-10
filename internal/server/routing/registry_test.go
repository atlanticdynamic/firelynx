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

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockApp implements the apps.App interface for testing
type mockApp struct {
	id string
}

func (m *mockApp) ID() string {
	return m.id
}

func (m *mockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	return nil
}

// mockAppRegistry implements apps.Registry for testing
type mockAppRegistry struct {
	apps map[string]apps.App
}

func newMockAppRegistry() *mockAppRegistry {
	return &mockAppRegistry{
		apps: make(map[string]apps.App),
	}
}

func (r *mockAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *mockAppRegistry) RegisterApp(app apps.App) error {
	r.apps[app.ID()] = app
	return nil
}

func (r *mockAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}

func (r *mockAppRegistry) addMockApp(id string) {
	r.apps[id] = &mockApp{id: id}
}

func TestNewRegistry(t *testing.T) {
	// Setup
	appRegistry := newMockAppRegistry()
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
	appRegistry := newMockAppRegistry()
	appRegistry.addMockApp("app1")
	appRegistry.addMockApp("app2")

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
	appRegistry := newMockAppRegistry()
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
	appRegistry := newMockAppRegistry()
	appRegistry.addMockApp("app1")

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
	// Setup app registry
	appRegistry := newMockAppRegistry()
	app1 := &mockApp{id: "app1"}
	app2 := &mockApp{id: "app2"}
	appRegistry.apps["app1"] = app1
	appRegistry.apps["app2"] = app2

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
	registry := NewRegistry(newMockAppRegistry(), nil, nil)
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
