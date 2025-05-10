package http_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test demonstrates how the route registry handles dynamic configuration
// updates and how HTTP handlers properly resolve routes after updates.
// It focuses specifically on immutability and atomic updates.
func TestDynamicUpdateWithRouteHandler(t *testing.T) {
	// Create a variable to hold the current routing config
	var currentConfig atomic.Pointer[routing.RoutingConfig]

	// Setup app registry with test apps
	app1 := &dynamicTestApp{id: "app1", response: "Response from App1"}
	app2 := &dynamicTestApp{id: "app2", response: "Response from App2"}
	appRegistry := &dynamicTestAppRegistry{
		apps: map[string]apps.App{
			"app1": app1,
			"app2": app2,
		},
	}

	// Initial configuration with just app1 routes
	initialConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []routing.Route{
					{
						Path:  "/api/v1",
						AppID: "app1",
						StaticData: map[string]any{
							"version": "1.0",
						},
					},
				},
			},
		},
	}

	currentConfig.Store(initialConfig)

	// Config callback that returns the current config
	configCallback := func() (*routing.RoutingConfig, error) {
		return currentConfig.Load(), nil
	}

	// Create logger and registry
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := routing.NewRegistry(appRegistry, configCallback, logger)

	// Initialize registry
	err := registry.Reload()
	require.NoError(t, err)

	// Now registry should be initialized
	assert.True(t, registry.IsInitialized())

	// Create a handler using a function that directly calls the registry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the route for this request
		resolved, err := registry.ResolveRoute("endpoint1", r)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no route matched, return 404
		if resolved == nil {
			http.NotFound(w, r)
			return
		}

		// Merge static data and params into a single map
		data := make(map[string]any)
		if resolved.StaticData != nil {
			for k, v := range resolved.StaticData {
				data[k] = v
			}
		}

		// Add path parameters as special "params" entry
		if len(resolved.Params) > 0 {
			data["params"] = resolved.Params
		}

		// Handle the request with the resolved app
		if err := resolved.App.HandleHTTP(r.Context(), w, r, data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Setup test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// --- Test initial configuration ---

	// App1 request should succeed
	resp1, err := http.Get(server.URL + "/api/v1/resource")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp1.Body.Close()) }()

	// Check that app1 handled the request
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "Response from App1", resp1.Header.Get("X-Response"))
	assert.Equal(t, "1.0", resp1.Header.Get("X-Version"))
	assert.Equal(t, int64(1), app1.requestCount.Load())

	// App2 route doesn't exist yet
	resp2, err := http.Get(server.URL + "/api/v2/resource")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp2.Body.Close()) }()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
	assert.Equal(t, int64(0), app2.requestCount.Load())

	// --- Update configuration ---

	// Reset app state for next test
	app1.requestCount.Store(0)
	app1.lastDataMu.Lock()
	app1.lastData = nil
	app1.lastDataMu.Unlock()

	app2.requestCount.Store(0)
	app2.lastDataMu.Lock()
	app2.lastData = nil
	app2.lastDataMu.Unlock()

	// New config adds a route for app2
	updatedConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []routing.Route{
					{
						Path:  "/api/v1",
						AppID: "app1",
						StaticData: map[string]any{
							"version": "1.1", // Updated version
						},
					},
					{
						Path:  "/api/v2",
						AppID: "app2",
						StaticData: map[string]any{
							"version": "2.0",
						},
					},
				},
			},
		},
	}

	// Update the current config
	currentConfig.Store(updatedConfig)

	// Reload the registry
	err = registry.Reload()
	require.NoError(t, err)

	// --- Test updated configuration ---

	// App1 request with updated route config
	resp3, err := http.Get(server.URL + "/api/v1/resource")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp3.Body.Close()) }()

	// Check that app1 handled the request with updated data
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.Equal(t, "Response from App1", resp3.Header.Get("X-Response"))
	assert.Equal(t, "1.1", resp3.Header.Get("X-Version")) // Should have the updated version
	assert.Equal(t, int64(1), app1.requestCount.Load())

	// App2 request should now succeed
	resp4, err := http.Get(server.URL + "/api/v2/resource")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp4.Body.Close()) }()

	// Check that app2 handled the request
	assert.Equal(t, http.StatusOK, resp4.StatusCode)
	assert.Equal(t, "Response from App2", resp4.Header.Get("X-Response"))
	assert.Equal(t, "2.0", resp4.Header.Get("X-Version"))
	assert.Equal(t, int64(1), app2.requestCount.Load())
}

// TestAtomicRouteTableUpdates verifies that route table updates are atomic
// and that in-flight requests are handled properly during updates.
func TestAtomicRouteTableUpdates(t *testing.T) {
	// Create app with delayed response
	app1 := &dynamicTestApp{
		id:       "app1",
		response: "Response from App1",
		delay:    100 * time.Millisecond, // Delay to simulate processing
	}

	appRegistry := &dynamicTestAppRegistry{
		apps: map[string]apps.App{
			"app1": app1,
		},
	}

	// Create a variable to hold the current routing config
	var currentConfig atomic.Pointer[routing.RoutingConfig]

	// Initial config
	initialConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []routing.Route{
					{
						Path:  "/api/v1",
						AppID: "app1",
						StaticData: map[string]any{
							"version": "1.0",
						},
					},
				},
			},
		},
	}
	currentConfig.Store(initialConfig)

	// Config callback
	configCallback := func() (*routing.RoutingConfig, error) {
		return currentConfig.Load(), nil
	}

	// Create registry and handler
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := routing.NewRegistry(appRegistry, configCallback, logger)

	// Initialize registry
	err := registry.Reload()
	require.NoError(t, err)

	// Create a handler using a function that directly calls the registry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the route for this request
		resolved, err := registry.ResolveRoute("endpoint1", r)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no route matched, return 404
		if resolved == nil {
			http.NotFound(w, r)
			return
		}

		// Merge static data and params into a single map
		data := make(map[string]any)
		if resolved.StaticData != nil {
			for k, v := range resolved.StaticData {
				data[k] = v
			}
		}

		// Add path parameters as special "params" entry
		if len(resolved.Params) > 0 {
			data["params"] = resolved.Params
		}

		// Handle the request with the resolved app
		if err := resolved.App.HandleHTTP(r.Context(), w, r, data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Start a long-running request that will still be processing during config update
	reqStarted := make(chan struct{})
	reqHandling := make(chan struct{})

	go func() {
		// Signal that we're starting the request
		close(reqStarted)

		// Make request to app1
		resp, err := http.Get(server.URL + "/api/v1/long-operation")
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()

		// Signal that request is being handled
		close(reqHandling)

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Response from App1", resp.Header.Get("X-Response"))
		assert.Equal(t, "1.0", resp.Header.Get("X-Version"))
	}()

	// Wait for request to start
	<-reqStarted

	// Allow a little time for the request to reach the app
	time.Sleep(10 * time.Millisecond)

	// Update config while the first request is still being processed
	updatedConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []routing.Route{
					{
						Path:  "/api/v1",
						AppID: "app1",
						StaticData: map[string]any{
							"version": "1.1", // Updated version
						},
					},
				},
			},
		},
	}
	currentConfig.Store(updatedConfig)

	// Reload the registry
	err = registry.Reload()
	require.NoError(t, err)

	// Make a new request that should use the updated config
	resp2, err := http.Get(server.URL + "/api/v1/new-request")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp2.Body.Close()) }()

	// Verify that the new request uses the updated config
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "Response from App1", resp2.Header.Get("X-Response"))
	assert.Equal(t, "1.1", resp2.Header.Get("X-Version")) // Should have updated version

	// Wait for the first request to complete
	<-reqHandling

	// Verify that both requests were processed
	assert.Equal(t, int64(2), app1.requestCount.Load())
}

// dynamicTestApp implements apps.App for testing with request counting
type dynamicTestApp struct {
	id           string
	response     string
	requestCount atomic.Int64
	delay        time.Duration
	lastData     map[string]any
	lastDataMu   sync.Mutex // Protects lastData from concurrent access
}

func (a *dynamicTestApp) ID() string {
	return a.id
}

func (a *dynamicTestApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	// Increment request counter using atomic (thread-safe)
	a.requestCount.Add(1)

	// Safely store the last data received
	a.lastDataMu.Lock()
	// Make a deep copy of the data map to avoid race conditions
	dataCopy := make(map[string]any, len(data))
	for k, v := range data {
		dataCopy[k] = v
	}
	a.lastData = dataCopy
	a.lastDataMu.Unlock()

	// Optional delay to simulate processing time
	if a.delay > 0 {
		time.Sleep(a.delay)
	}

	// Set response headers
	w.Header().Set("X-Response", a.response)

	// Add any static data as headers
	for k, v := range data {
		if strVal, ok := v.(string); ok {
			w.Header().Set("X-"+k, strVal)
		}
	}

	// Write response
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(a.response))
	return err
}

// dynamicTestAppRegistry implements apps.Registry for testing
type dynamicTestAppRegistry struct {
	apps map[string]apps.App
}

func (r *dynamicTestAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *dynamicTestAppRegistry) RegisterApp(app apps.App) error {
	r.apps[app.ID()] = app
	return nil
}

func (r *dynamicTestAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}
