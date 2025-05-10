package core

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAppRegistry for testing
type testAppRegistry struct {
	apps map[string]apps.App
}

func newTestAppRegistry() *testAppRegistry {
	return &testAppRegistry{
		apps: make(map[string]apps.App),
	}
}

func (r *testAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *testAppRegistry) RegisterApp(app apps.App) error {
	r.apps[app.ID()] = app
	return nil
}

func (r *testAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}

// Create a mock config callback function
func createTestConfigCallback(cfg *config.Config, err error) func() (*config.Config, error) {
	return func() (*config.Config, error) {
		return cfg, err
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
				ID:          "test-endpoint",
				ListenerIDs: []string{"test-listener"},
			},
		},
	}

	// Create a mock app registry - not used in this test case but kept for reference
	_ = newTestAppRegistry()

	// Test successful reload
	t.Run("successful reload", func(t *testing.T) {
		// Setup a runner with a mock callback that returns our test config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := New(createTestConfigCallback(testConfig, nil))
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
		runner, err := New(nil)
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

	// Test reload with config error
	t.Run("config error", func(t *testing.T) {
		// Setup a runner with a mock callback that returns an error
		expectedErr := errors.New("config error")
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := New(createTestConfigCallback(nil, expectedErr))
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// Verify an error was sent to the channel
		select {
		case err := <-runner.serverErrors:
			assert.Contains(t, err.Error(), "failed to load configuration")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected error from reload, but none received")
		}
	})

	// Test reload with nil config
	t.Run("nil config", func(t *testing.T) {
		// Setup a runner with a mock callback that returns a nil config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := New(createTestConfigCallback(nil, nil))
		require.NoError(t, err)

		// Set the logger option
		WithLogger(logger)(runner)

		// Manually initialize serverErrors channel
		runner.serverErrors = make(chan error, 1)

		// Call Reload
		runner.Reload()

		// Verify no errors were sent to the channel (nil config is logged but not an error)
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
				ID:          "http-endpoint",
				ListenerIDs: []string{"http-listener"},
			},
		},
	}

	// Test HTTP config callback
	t.Run("with valid config", func(t *testing.T) {
		// Setup a runner with our test config
		logger := slog.New(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
		)
		runner, err := New(createTestConfigCallback(testConfig, nil))
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
		runner, err := New(createTestConfigCallback(nil, nil))
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
