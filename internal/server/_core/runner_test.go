package core

import (
	"context"
	"errors"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// buildTestConfig creates a minimal config with an HTTP listener and a route
// that directs to the echo app.
func buildTestConfig() *config.Config {
	cfg := &config.Config{
		Version: "test",
		Listeners: []listeners.Listener{
			{
				ID:      "test-listener",
				Address: ":0", // Use port 0 to get a random port
				Options: options.HTTP{
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
					IdleTimeout:  60 * time.Second,
					DrainTimeout: 30 * time.Second,
				},
			},
		},
		Endpoints: []endpoints.Endpoint{
			{
				ID:         "test-endpoint",
				ListenerID: "test-listener",
				Routes: []routes.Route{
					{
						AppID:     "echo", // This matches the echo app registered in Runner.New()
						Condition: conditions.NewHTTP("/echo", "GET"),
					},
				},
			},
		},
	}

	return cfg
}

func TestRunner_ConfigurationAccess(t *testing.T) {
	cfg := buildTestConfig()
	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	runner, err := NewRunner(configCallback)
	assert.NoError(t, err)

	// Boot the runner to initialize the config
	err = runner.boot()
	assert.NoError(t, err)

	assert.Equal(t, cfg, runner.currentConfig)
}

func TestRunner_New(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	runner, err := NewRunner(configCallback)

	// Verify setup
	assert.NoError(t, err)
	assert.NotNil(t, runner)
	assert.Equal(t, "core.Runner", runner.String())

	// Create a mock app for testing
	echoApp := mocks.NewMockApp("echo")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create app collection with the mock app
	appCollection, err := apps.NewAppCollection([]apps.App{echoApp})
	assert.NoError(t, err)

	// Set the app collection
	runner.appCollection = appCollection

	// Verify app collection has the echo app
	app, found := runner.appCollection.GetApp("echo")
	assert.True(t, found)
	assert.NotNil(t, app)
}

func TestRunner_Run(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	// Create a context with timeout for our test
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Create the runner and run it
	runner, err := NewRunner(configCallback)
	assert.NoError(t, err)

	err = runner.Run(ctx)

	// The run should eventually exit due to context cancellation
	// The error might be context.DeadlineExceeded or nil depending on how the runner handles timeout
	assert.True(t, err == nil || errors.Is(err, context.DeadlineExceeded))
}

func TestRunner_Stop(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	// Create the runner
	runner, err := NewRunner(configCallback)
	assert.NoError(t, err)

	// Run the runner in a goroutine
	go func() {
		ctx := context.Background()
		err := runner.Run(ctx)
		// It should exit with a canceled context error or nil
		assert.True(t, err == nil || errors.Is(err, context.Canceled))
	}()

	// Give it a moment to start up
	time.Sleep(10 * time.Millisecond)

	// Stop the runner
	runner.Stop()

	// Add a small sleep to allow background goroutines to finish
	time.Sleep(10 * time.Millisecond)
}

func TestRunner_WithApps(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	// Create the runner
	runner, err := NewRunner(configCallback)
	assert.NoError(t, err)

	// Create mock apps for testing
	specialApp := mocks.NewMockApp("special")
	specialApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	echoApp := mocks.NewMockApp("echo")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create app collection with both mock apps
	appCollection, err := apps.NewAppCollection([]apps.App{specialApp, echoApp})
	assert.NoError(t, err)

	// Set the app collection
	runner.appCollection = appCollection

	// Verify both apps are in the collection
	app1, found1 := runner.appCollection.GetApp("echo")
	assert.True(t, found1)
	assert.NotNil(t, app1)

	app2, found2 := runner.appCollection.GetApp("special")
	assert.True(t, found2)
	assert.NotNil(t, app2)
}

func TestRunner_EndpointListenerAssociation(t *testing.T) {
	// Create a configuration similar to the E2E test configuration
	cfg := &config.Config{
		Version: "test",
		Listeners: []listeners.Listener{
			{
				ID:      "http_listener",
				Address: ":0", // Use port 0 to get a random port
				Options: options.HTTP{
					ReadTimeout:  1 * time.Second,
					WriteTimeout: 1 * time.Second,
					IdleTimeout:  1 * time.Second,
					DrainTimeout: 1 * time.Second,
				},
			},
		},
		Endpoints: []endpoints.Endpoint{
			{
				ID:         "echo_endpoint",
				ListenerID: "http_listener",
				Routes: []routes.Route{
					{
						AppID:     "echo", // This matches the echo app registered in Runner.New()
						Condition: conditions.NewHTTP("/echo", "GET"),
					},
				},
			},
		},
	}

	// Create a configCallback that returns our test configuration
	configCallback := func() config.Config {
		return *cfg
	}

	// Create the runner
	runner, err := NewRunner(configCallback)
	assert.NoError(t, err)

	// Create a mock echo app and add it to the runner
	echoApp := mocks.NewMockApp("echo")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	appCollection, err := apps.NewAppCollection([]apps.App{echoApp})
	assert.NoError(t, err)
	runner.appCollection = appCollection

	// Boot the runner to initialize the config
	err = runner.boot()
	assert.NoError(t, err)

	// Test the HTTP config callback to ensure it properly processes the endpoint association
	httpConfigCallback := runner.GetHTTPConfigCallback()
	httpConfig, err := httpConfigCallback()
	assert.NoError(t, err)
	assert.NotNil(t, httpConfig)

	// Verify that there's one HTTP listener in the config
	assert.Equal(t, 1, len(httpConfig.Listeners))

	// Verify that the EndpointID is correctly set on the HTTP listener
	assert.Equal(
		t,
		"echo_endpoint",
		httpConfig.Listeners[0].EndpointID,
		"EndpointID should be set to 'echo_endpoint' based on the listener_ids in the endpoint config",
	)

	// Verify the ID matches what we expect
	assert.Equal(t, "http_listener", httpConfig.Listeners[0].ID)
}
