package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/stretchr/testify/assert"
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
				ID:          "test-endpoint",
				ListenerIDs: []string{"test-listener"},
				Routes: []routes.Route{
					{
						AppID:     "echo", // This matches the echo app registered in Runner.New()
						Condition: conditions.NewHTTP("/echo"),
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
	configCallback := func() (*config.Config, error) {
		return cfg, nil
	}

	runner, err := New(configCallback)
	assert.NoError(t, err)

	// Boot the runner to initialize the config
	err = runner.boot()
	assert.NoError(t, err)

	assert.Equal(t, cfg, runner.currentConfig)
}

func TestRunner_New(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() (*config.Config, error) {
		return cfg, nil
	}

	runner, err := New(configCallback)

	// Verify setup
	assert.NoError(t, err)
	assert.NotNil(t, runner)
	assert.Equal(t, "core.Runner", runner.String())

	// Verify app registry has the echo app
	app, found := runner.appRegistry.GetApp("echo")
	assert.True(t, found)
	assert.NotNil(t, app)
}

func TestRunner_Run(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() (*config.Config, error) {
		return cfg, nil
	}

	// Create a context with timeout for our test
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Create the runner and run it
	runner, err := New(configCallback)
	assert.NoError(t, err)

	err = runner.Run(ctx)

	// The run should eventually exit due to context cancellation
	// The error might be context.DeadlineExceeded or nil depending on how the runner handles timeout
	assert.True(t, err == nil || errors.Is(err, context.DeadlineExceeded))
}

func TestRunner_Stop(t *testing.T) {
	cfg := buildTestConfig()

	// Create a configCallback that returns our test configuration
	configCallback := func() (*config.Config, error) {
		return cfg, nil
	}

	// Create the runner
	runner, err := New(configCallback)
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
	configCallback := func() (*config.Config, error) {
		return cfg, nil
	}

	// Create the runner
	runner, err := New(configCallback)
	assert.NoError(t, err)

	// Create a specialized echo app
	specialApp := echo.New("special")

	// Register the app manually
	err = runner.appRegistry.RegisterApp(specialApp)
	assert.NoError(t, err)

	// Verify both apps are registered
	app1, found1 := runner.appRegistry.GetApp("echo")
	assert.True(t, found1)
	assert.NotNil(t, app1)

	app2, found2 := runner.appRegistry.GetApp("special")
	assert.True(t, found2)
	assert.NotNil(t, app2)
}
