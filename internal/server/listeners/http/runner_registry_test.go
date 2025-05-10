package http

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunner_WithRouteRegistry tests the HTTP runner with the new route registry
func TestRunner_WithRouteRegistry(t *testing.T) {
	// Setup app registry
	appRegistry := &testAppRegistry{
		apps: map[string]apps.App{
			"app1": &testApp{id: "app1"},
			"app2": &testApp{id: "app2"},
		},
	}

	// Setup routing config
	routingConfig := &routing.RoutingConfig{
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

	// Create route registry
	routingCallback := func() (*routing.RoutingConfig, error) {
		return routingConfig, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	routeRegistry := routing.NewRegistry(appRegistry, routingCallback, logger)

	// Initialize registry
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Create HTTP config
	httpConfig := &Config{
		AppRegistry:   appRegistry,
		RouteRegistry: routeRegistry,
		Listeners: []ListenerConfig{
			{
				ID:           "listener1",
				Address:      "localhost:8080",
				EndpointID:   "endpoint1",
				ReadTimeout:  1 * time.Minute,
				WriteTimeout: 1 * time.Minute,
				IdleTimeout:  1 * time.Minute,
				DrainTimeout: 10 * time.Minute,
			},
		},
		logger: logger,
	}

	// Create HTTP config callback
	configCallback := func() (*Config, error) {
		return httpConfig, nil
	}

	// Create runner
	runner, err := NewRunner(configCallback, WithManagerLogger(logger))
	require.NoError(t, err)

	// Get runner config
	runnerConfig, err := runner.getRunnerConfig()
	require.NoError(t, err)

	// Check that we have one entry
	require.Equal(t, 1, len(runnerConfig.Entries))

	// Check server configuration
	server := runnerConfig.Entries[0].Runnable
	require.NotNil(t, server)

	// The server is a wrapper.HttpServer type
	httpServer := server

	// Since the field is unexported, we don't need to check its contents directly
	// Just verify that the server was created successfully
	require.NotNil(t, httpServer)

	// This is a functional test, so we primarily care that creation succeeded
	// rather than examining the private fields
}

// Test with both styles of routing (legacy and new)
func TestRunner_WithMixedRouting(t *testing.T) {
	// Setup app registry
	appRegistry := &testAppRegistry{
		apps: map[string]apps.App{
			"app1": &testApp{id: "app1"},
			"app2": &testApp{id: "app2"},
			"app3": &testApp{id: "app3"},
		},
	}

	// Setup routing config
	routingConfig := &routing.RoutingConfig{
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

	// Create route registry
	routingCallback := func() (*routing.RoutingConfig, error) {
		return routingConfig, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	routeRegistry := routing.NewRegistry(appRegistry, routingCallback, logger)

	// Initialize registry
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Create HTTP config with both types of listeners
	httpConfig := &Config{
		AppRegistry:   appRegistry,
		RouteRegistry: routeRegistry,
		Listeners: []ListenerConfig{
			{
				// New style with route registry
				ID:           "listener1",
				Address:      "localhost:8080",
				EndpointID:   "endpoint1",
				ReadTimeout:  1 * time.Minute,
				WriteTimeout: 1 * time.Minute,
				IdleTimeout:  1 * time.Minute,
				DrainTimeout: 10 * time.Minute,
			},
			{
				// Legacy style with direct routes
				ID:           "listener2",
				Address:      "localhost:8081",
				ReadTimeout:  1 * time.Minute,
				WriteTimeout: 1 * time.Minute,
				IdleTimeout:  1 * time.Minute,
				DrainTimeout: 10 * time.Minute,
				Routes: []RouteConfig{
					{
						Path:  "/api/v3",
						AppID: "app3",
						StaticData: map[string]any{
							"version": "3.0",
						},
					},
				},
			},
		},
		logger: logger,
	}

	// Create HTTP config callback
	configCallback := func() (*Config, error) {
		return httpConfig, nil
	}

	// Create runner
	runner, err := NewRunner(configCallback, WithManagerLogger(logger))
	require.NoError(t, err)

	// Get runner config
	runnerConfig, err := runner.getRunnerConfig()
	require.NoError(t, err)

	// Check that we have two entries
	assert.Equal(t, 2, len(runnerConfig.Entries))

	// Test that we can run the runner
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- runner.Run(ctx)
	}()

	// Run for a short time then cancel
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Check that Run completed without error
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Runner.Run did not complete in time")
	}
}

// Tests error handling for invalid configs
func TestRunner_WithInvalidConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test case 1: Missing both registry types
	invalidConfig1 := &Config{
		Listeners: []ListenerConfig{
			{
				ID:         "listener1",
				Address:    "localhost:8080",
				EndpointID: "endpoint1",
			},
		},
	}

	// Create HTTP config callback
	configCallback1 := func() (*Config, error) {
		return invalidConfig1, nil
	}

	// Create runner
	runner1, err := NewRunner(configCallback1, WithManagerLogger(logger))
	require.NoError(t, err)

	// Try to get runner config - should fail validation
	_, err = runner1.getRunnerConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either AppRegistry or RouteRegistry must be provided")

	// Test case 2: Listener with neither endpoint ID nor routes
	appRegistry := &testAppRegistry{
		apps: map[string]apps.App{
			"app1": &testApp{id: "app1"},
		},
	}

	invalidConfig2 := &Config{
		AppRegistry: appRegistry,
		Listeners: []ListenerConfig{
			{
				ID:      "listener1",
				Address: "localhost:8080",
				// No EndpointID or Routes
			},
		},
	}

	configCallback2 := func() (*Config, error) {
		return invalidConfig2, nil
	}

	runner2, err := NewRunner(configCallback2, WithManagerLogger(logger))
	require.NoError(t, err)

	// This should fail during buildCompositeConfig due to missing both endpoint and routes
	_, err = runner2.getRunnerConfig()
	require.Error(t, err) // Now we expect an error due to missing EndpointID and Routes
	assert.Contains(t, err.Error(), "either EndpointID or Routes must be provided")
}
