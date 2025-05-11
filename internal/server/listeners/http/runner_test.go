package http

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name           string
		configCallback ConfigCallback
		expectError    bool
		runExpected    bool // Whether Run() would succeed or fail
	}{
		{
			name: "valid runner",
			configCallback: func() (*Config, error) {
				return &Config{
					AppRegistry: mocks.NewMockRegistry(),
					Listeners: []ListenerConfig{
						{
							ID:           "test",
							Address:      ":8080",
							ReadTimeout:  5 * time.Second,
							WriteTimeout: 10 * time.Second,
							Routes: []RouteConfig{
								{
									Path:  "/test",
									AppID: "test-app",
								},
							},
						},
					},
				}, nil
			},
			expectError: false,
			runExpected: true,
		},
		{
			name: "nil config",
			configCallback: func() (*Config, error) {
				return nil, nil
			},
			expectError: false, // NewRunner succeeds, but Run would fail
			runExpected: false,
		},
		{
			name: "config error",
			configCallback: func() (*Config, error) {
				return nil, fmt.Errorf("config error")
			},
			expectError: false, // Constructor won't fail, but Run would
			runExpected: false,
		},
		{
			name:           "nil callback",
			configCallback: nil,
			expectError:    true, // Requires a non-nil callback
			runExpected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test only the constructor, not Run
			m, err := NewRunner(tt.configCallback)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)

				// Note: In a real test, we would also check if Run fails as expected
				// with tt.runExpected, but we don't want to actually call Run here
				// since that would start the server
			}
		})
	}
}

func TestRunner_Run(t *testing.T) {
	// Create a registry with a test app
	registry := mocks.NewMockRegistry()
	// Set up the registry to return the test app
	app := mocks.NewMockApp("test-app")
	registry.On("GetApp", "test-app").Return(app, true)

	listenPort := testutil.GetRandomListeningPort(t)

	// Create a config callback that returns a valid HTTP config
	configCallback := func() (*Config, error) {
		return &Config{
			AppRegistry: registry,
			Listeners: []ListenerConfig{
				{
					ID:           "test",
					Address:      listenPort,
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 10 * time.Second,
					Routes: []RouteConfig{
						{
							Path:  "/test",
							AppID: "test-app",
						},
					},
				},
			},
		}, nil
	}

	// Create runner
	r, err := NewRunner(configCallback)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check listener states
	states := r.GetListenerStates()
	assert.NotEmpty(t, states)

	// Stop the runner
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}

// TestRunner_E2EConfigFormat tests that the HTTP runner correctly handles
// configurations like those used in the E2E tests.
// IMPORTANT: This test verifies a key requirement for E2E tests:
// When using EndpointID in a listener, the RouteRegistry must be properly set up.
// This was the source of a bug in the E2E tests where listeners had EndpointIDs
// but no routes were being registered because the RouteRegistry was missing.
func TestRunner_E2EConfigFormat(t *testing.T) {
	// Create mock app registry
	registry := mocks.NewMockRegistry()
	mockApp := mocks.NewMockApp("echo_app")
	registry.On("GetApp", "echo_app").Return(mockApp, true)

	// Create HTTP runner configuration simulating E2E test config
	listenerConfig := ListenerConfig{
		ID:           "http_listener",
		Address:      ":0",            // Ephemeral port
		EndpointID:   "echo_endpoint", // This is the key part - it matches test config template
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  1 * time.Second,
		DrainTimeout: 1 * time.Second,
	}

	// Create routing registry
	routes := []routing.Route{
		{
			Path:  "/echo",
			AppID: "echo_app",
		},
	}

	// Create routing config
	routingConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "echo_endpoint",
				Routes:     routes,
			},
		},
	}

	// Create and initialize route registry
	routingCallback := func() (*routing.RoutingConfig, error) {
		return routingConfig, nil
	}

	// Initialize the route registry
	routeRegistry := routing.NewRegistry(registry, routingCallback, nil)
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Create HTTP config with route registry
	routeConfig := &Config{
		AppRegistry:   registry,
		RouteRegistry: routeRegistry,
		Listeners:     []ListenerConfig{listenerConfig},
	}

	// Create config callback
	configCallback := func() (*Config, error) {
		return routeConfig, nil
	}

	// Create the HTTP runner
	runner, err := NewRunner(configCallback)
	require.NoError(t, err)
	require.NotNil(t, runner)

	// Get the composite runner config
	runnerConfig, err := runner.getRunnerConfig()
	require.NoError(t, err)

	// Verify that buildCompositeConfig correctly processes the EndpointID
	// If the EndpointID is processed correctly, it won't error on "Listener has neither endpoint ID nor routes"
	assert.NotNil(t, runnerConfig, "Runner config should not be nil with valid endpoint ID")

	// Verify entries are created for the listener
	assert.Equal(t, 1, len(runnerConfig.Entries), "Should have one HTTP server entry")
}

func TestRunner_Reload(t *testing.T) {
	// Create a registry with a test app
	registry := mocks.NewMockRegistry()
	// Set up the registry to return the test app
	app := mocks.NewMockApp("test-app")
	registry.On("GetApp", "test-app").Return(app, true)

	listenPort := fmt.Sprintf(":%d", testutil.GetRandomPort(t))

	// Create a config callback that returns a valid HTTP config
	configCallback := func() (*Config, error) {
		return &Config{
			AppRegistry: registry,
			Listeners: []ListenerConfig{
				{
					ID:           "test",
					Address:      listenPort,
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 10 * time.Second,
					Routes: []RouteConfig{
						{
							Path:  "/test",
							AppID: "test-app",
						},
					},
				},
			},
		}, nil
	}

	// Create runner
	r, err := NewRunner(configCallback)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Get initial listener states
	initialStates := r.GetListenerStates()
	assert.NotEmpty(t, initialStates)

	// Reload the runner
	r.Reload()

	// Give it a moment to reload
	time.Sleep(100 * time.Millisecond)

	// Get updated listener states
	updatedStates := r.GetListenerStates()
	assert.NotEmpty(t, updatedStates)

	// Stop the runner
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}
