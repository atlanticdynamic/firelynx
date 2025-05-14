package core

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create a mock config callback function
func createTestConfigCallback(cfg *config.Config, err error) func() config.Config {
	return func() config.Config {
		if cfg == nil {
			return config.Config{}
		}
		return *cfg
	}
}

// TestRunnerReload tests the Reload method
func TestRunnerReload(t *testing.T) {
	// Setup a test config
	testConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "test-listener",
				Address: "localhost:8080",
				Options: options.HTTP{},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "test-endpoint",
				ListenerID: "test-listener",
			},
		},
	}

	// Test successful reload
	t.Run("successful reload", func(t *testing.T) {
		// Setup a runner with a mock callback that returns our test config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := NewRunner(createTestConfigCallback(testConfig, nil))
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// Verify the config was updated
		assert.Equal(t, testConfig, runner.currentConfig)

		// Verify no errors were sent to the channel
		select {
		case err := <-runner.serverErrors:
			t.Fatalf("Unexpected error from reload: %v", err)
		default:
			// No error, this is expected
		}
	})

	// Test reload with nil callback
	t.Run("nil callback", func(t *testing.T) {
		// Setup a runner with a nil callback
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := NewRunner(nil)
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Set configCallback to nil to simulate this error case
		runner.configCallback = nil

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// Verify an error was sent to the channel
		select {
		case err := <-runner.serverErrors:
			assert.Contains(t, err.Error(), "config callback is nil")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected error from reload, but none received")
		}
	})

	// Test reload with empty config
	t.Run("config error", func(t *testing.T) {
		// Setup a runner with a mock callback that returns an empty config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)

		emptyConfigCallback := func() config.Config {
			return config.Config{} // Return empty config
		}

		runner, err := NewRunner(emptyConfigCallback)
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// There's no error anymore, since we just return an empty config
		// instead of an error in the callback
		select {
		case err := <-runner.serverErrors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(100 * time.Millisecond):
			// No error expected anymore
		}
	})

	// Test reload with default config
	t.Run("default config", func(t *testing.T) {
		// Setup a runner with a mock callback that returns a default config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)

		defaultConfigCallback := func() config.Config {
			return config.Config{
				Version: config.VersionLatest,
			}
		}

		runner, err := NewRunner(defaultConfigCallback)
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// Verify no errors were sent to the channel
		select {
		case err := <-runner.serverErrors:
			t.Fatalf("Unexpected error from reload: %v", err)
		default:
			// No error, this is expected
		}
	})
}

// TestGetHTTPConfigCallback tests the GetHTTPConfigCallback method
func TestGetHTTPConfigCallback(t *testing.T) {
	// Setup a test config with HTTP listeners
	testConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http-listener",
				Address: "localhost:8080",
				Options: options.HTTP{
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
					IdleTimeout:  10 * time.Second,
					DrainTimeout: 30 * time.Second,
				},
			},
			{
				ID:      "grpc-listener", // Not HTTP
				Address: "localhost:9000",
				Options: options.GRPC{},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "http-endpoint",
				ListenerID: "http-listener",
				Routes: []routes.Route{
					{
						AppID:     "echo", // This matches the echo app registered in Runner.New()
						Condition: conditions.NewHTTP("/echo", "GET"),
					},
				},
			},
		},
	}

	// Test HTTP config callback
	t.Run("with valid config", func(t *testing.T) {
		// Setup a runner with our test config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := NewRunner(createTestConfigCallback(testConfig, nil))
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Set the current config manually since we're not calling Run
		runner.currentConfig = testConfig

		// Get the HTTP config callback
		callback := runner.GetHTTPConfigCallback()
		require.NotNil(t, callback)

		// Execute the callback
		httpConfig, err := callback()
		require.NoError(t, err)
		require.NotNil(t, httpConfig)

		// Verify the config has correct values
		assert.NotNil(t, httpConfig.AppRegistry)
		require.Len(t, httpConfig.Listeners, 1) // Only one HTTP listener

		// Verify listener properties
		assert.Equal(t, "http-listener", httpConfig.Listeners[0].ID)
		assert.Equal(t, "localhost:8080", httpConfig.Listeners[0].Address)
		assert.Equal(t, "http-endpoint", httpConfig.Listeners[0].EndpointID)
		assert.Equal(t, 10*time.Second, httpConfig.Listeners[0].ReadTimeout)
		assert.Equal(t, 10*time.Second, httpConfig.Listeners[0].WriteTimeout)
		assert.Equal(t, 10*time.Second, httpConfig.Listeners[0].IdleTimeout)
		assert.Equal(t, 30*time.Second, httpConfig.Listeners[0].DrainTimeout)
	})

	// Test with nil config
	t.Run("with nil config", func(t *testing.T) {
		// Setup a runner with no config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := NewRunner(createTestConfigCallback(nil, nil))
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// currentConfig should be nil initially
		assert.Nil(t, runner.currentConfig)

		// Get the HTTP config callback
		callback := runner.GetHTTPConfigCallback()
		require.NotNil(t, callback)

		// Execute the callback - should return error
		httpConfig, err := callback()
		assert.Error(t, err)
		assert.Nil(t, httpConfig)
		assert.Contains(t, err.Error(), "no configuration available")
	})
}
