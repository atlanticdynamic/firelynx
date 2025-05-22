package cfgservice

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRun tests all Run method functionality with different configurations
func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("basic_functionality", func(t *testing.T) {
		// Create a mock orchestrator
		mockOrchestrator := new(MockConfigOrchestrator)
		mockOrchestrator.On("ProcessTransaction", mock.Anything, mock.Anything).Return(nil)
		mockOrchestrator.On("RegisterParticipant", mock.Anything).Return(nil)

		// Create a Runner instance with a listen address
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
			mockOrchestrator,
		)
		require.NoError(t, err)

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		assert.NoError(t, chanErr)
	})

	t.Run("with_invalid_address", func(t *testing.T) {
		// Create a mock orchestrator
		mockOrchestrator := new(MockConfigOrchestrator)
		mockOrchestrator.On("ProcessTransaction", mock.Anything, mock.Anything).Return(nil)
		mockOrchestrator.On("RegisterParticipant", mock.Anything).Return(nil)

		// Create a Runner with an invalid listen address that will cause NewGRPCManager to fail
		listenAddr := "invalid:address:with:too:many:colons"
		r, err := NewRunner(
			listenAddr,
			mockOrchestrator,
		)
		require.NoError(t, err)

		// Run should return the error from NewGRPCManager
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = r.Run(ctx)
		assert.Error(
			t,
			err,
			"Run should return an error when NewGRPCManager fails with an invalid address",
		)
	})

	t.Run("with_custom_logger", func(t *testing.T) {
		// Create a mock orchestrator
		mockOrchestrator := new(MockConfigOrchestrator)
		mockOrchestrator.On("ProcessTransaction", mock.Anything, mock.Anything).Return(nil)
		mockOrchestrator.On("RegisterParticipant", mock.Anything).Return(nil)

		// Create a Runner instance with custom logger
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
			mockOrchestrator,
			WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		)
		require.NoError(t, err)

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		assert.NoError(t, chanErr)
	})

	t.Run("stop_before_run", func(t *testing.T) {
		// Create a mock orchestrator
		mockOrchestrator := new(MockConfigOrchestrator)

		// Create a Runner instance
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
			mockOrchestrator,
		)
		require.NoError(t, err)

		// Call Stop before Run
		r.Stop()

		// This should not panic or cause issues
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run should handle being stopped before starting
		err = r.Run(ctx)
		assert.NoError(t, err)
	})
}
