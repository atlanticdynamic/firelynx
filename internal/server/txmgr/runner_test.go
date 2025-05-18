package txmgr

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
)

func TestNewRunnerMinimalOptions(t *testing.T) {
	// Create a minimal runner with just a config callback
	callback := func() config.Config {
		return config.Config{}
	}

	// Create the runner
	runner, err := NewRunner(callback)
	assert.NoError(t, err)
	assert.NotNil(t, runner)
}

func TestRunnerOptionsFull(t *testing.T) {
	// Create a minimal runner with just a config callback
	callback := func() config.Config {
		return config.Config{}
	}

	// Create a custom logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create the runner with all options
	runner, err := NewRunner(
		callback,
		WithLogger(logger),
	)
	assert.NoError(t, err)
	assert.NotNil(t, runner)
}

func TestRunnerBoot(t *testing.T) {
	// Create a test config
	testConfig := config.Config{
		Version: "v1",
	}

	// Create a config callback that returns our test config
	callback := func() config.Config {
		return testConfig
	}

	// Create the runner
	runner, err := NewRunner(callback)
	assert.NoError(t, err)
	assert.NotNil(t, runner)

	// Call boot to initialize the runner
	err = runner.boot()
	assert.NoError(t, err)
}

func TestRunnerBootConfigError(t *testing.T) {
	// Create a runner with a nil config callback
	runner, err := NewRunner(nil)
	assert.NoError(t, err)
	assert.NotNil(t, runner)

	// Call boot - should fail because config callback is nil
	err = runner.boot()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config callback is nil")
}

func TestRunnerRunLifecycle(t *testing.T) {
	// Create a test config
	testConfig := config.Config{
		Version: "v1",
	}

	// Create a config callback that returns our test config
	callback := func() config.Config {
		return testConfig
	}

	// Create the runner
	runner, err := NewRunner(callback)
	assert.NoError(t, err)
	assert.NotNil(t, runner)

	// Run the runner with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start the runner in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	// Allow time for the runner to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context to stop the runner
	cancel()

	// Wait for the runner to exit
	err = <-errCh
	assert.Equal(t, context.Canceled, err)
}

func TestRunnerReload(t *testing.T) {
	// Create an initial test config
	initialConfig := config.Config{
		Version: "v1",
	}

	// Create a new config that will be used after reload
	newConfig := config.Config{
		Version: "v2",
	}

	// We'll switch configs when reload is called - with mutex protection
	var configMutex sync.Mutex
	currentConfig := initialConfig
	callback := func() config.Config {
		configMutex.Lock()
		defer configMutex.Unlock()
		return currentConfig
	}

	// Create the runner
	runner, err := NewRunner(callback)
	assert.NoError(t, err)

	ctx := t.Context()

	// Start the runner in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	// Allow time for the runner to start
	time.Sleep(50 * time.Millisecond)

	// Change the current config and reload - with mutex protection
	configMutex.Lock()
	currentConfig = newConfig
	configMutex.Unlock()
	runner.Reload()
}

func TestRunnerPollConfig(t *testing.T) {
	// Create an initial test config
	initialConfig := config.Config{
		Version: "v1",
	}

	// Create a new config that will be used after the poll interval
	newConfig := config.Config{
		Version: "v2",
	}

	// We'll switch configs after a delay - with mutex protection
	var configMutex sync.Mutex
	currentConfig := initialConfig
	callback := func() config.Config {
		configMutex.Lock()
		defer configMutex.Unlock()
		return currentConfig
	}

	// Create the runner
	runner, err := NewRunner(callback)
	assert.NoError(t, err)

	// Start the runner
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = runner.Run(ctx)
	}()

	// Start polling with a short interval
	runner.PollConfig(ctx, 100*time.Millisecond)

	// Wait to ensure polling starts
	time.Sleep(50 * time.Millisecond)

	// Change the current config - with mutex protection
	configMutex.Lock()
	currentConfig = newConfig
	configMutex.Unlock()

	// Wait for the poll interval to trigger
	time.Sleep(200 * time.Millisecond)
}

// Helper function to create a test config with HTTP listeners
func createTestConfig() *config.Config {
	return &config.Config{
		Version: "v1",
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http_listener",
				Address: "localhost:8080",
				Options: options.HTTP{},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "echo_endpoint",
				ListenerID: "http_listener",
			},
		},
	}
}

// Simple helper that returns a function that returns the provided config
func createTestConfigCallback(cfg *config.Config, err error) func() config.Config {
	return func() config.Config {
		if cfg == nil {
			return config.Config{}
		}
		return *cfg
	}
}
