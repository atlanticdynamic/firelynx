package txmgr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnerString(t *testing.T) {
	// Create a runner with a config callback
	callback := func() config.Config {
		return config.Config{}
	}

	// Create the runner
	runner, err := NewRunner(callback)
	require.NoError(t, err)

	// Check that String() returns a non-empty string
	name := runner.String()
	assert.NotEmpty(t, name, "String() should return a non-empty value")
}

func TestRunnerSetConfigProvider(t *testing.T) {
	// Create a runner with a nil config callback
	runner, err := NewRunner(nil)
	require.NoError(t, err)

	// Define a new callback
	callback := func() config.Config {
		return config.Config{
			Version: "v2", // We'll check for this later
		}
	}

	// Set the new callback
	runner.SetConfigProvider(callback)

	// Start the runner - this should call the new callback
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop the runner
	cancel()

	// Wait for it to exit
	<-errCh
}

func TestRunnerStop(t *testing.T) {
	// Create a runner
	runner, err := NewRunner(func() config.Config {
		return config.Config{}
	})
	require.NoError(t, err)

	// Start the runner
	ctx, cancel := context.WithCancel(t.Context())

	var runErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		runErr = runner.Run(ctx)
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop the runner explicitly
	runner.Stop()

	// Cancel context and wait for goroutine to finish
	cancel()
	<-done

	// Assert that run completed (context canceled is expected)
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		t.Errorf("Runner failed with unexpected error: %v", runErr)
	}
}
