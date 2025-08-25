package txmgr

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkFailedOnErrorStateTransaction reproduces the CI issue where
// a transaction in StateError gets MarkFailed called on it, which should fail
// because StateError is a terminal state with no transitions allowed
func TestMarkFailedOnErrorStateTransaction(t *testing.T) {
	t.Parallel()

	// Create minimal config for testing
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	cfg.Version = config.VersionLatest

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})

	// Create a transaction
	tx, err := transaction.New(
		transaction.SourceTest,
		"TestMarkFailedOnErrorStateTransaction",
		"test-request-id",
		cfg,
		handler,
	)
	require.NoError(t, err)

	// Simulate transaction reaching StateError (like in CI logs)
	err = tx.MarkError(context.Canceled)
	require.NoError(t, err)
	assert.Equal(t, finitestate.StateError, tx.GetState(), "Transaction should be in StateError")

	// Now try to call MarkFailed with a canceled context (like during shutdown)
	// This should return early due to context cancellation, not attempt the invalid transition
	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	err = tx.MarkFailed(canceledCtx, errors.New("saga processing failed: context canceled"))
	require.Error(t, err, "MarkFailed should return context error when context is canceled")
	require.ErrorIs(t, err, context.Canceled, "Should return context.Canceled error")

	// With non-canceled context, should handle invalid transition gracefully
	err = tx.MarkFailed(
		t.Context(),
		errors.New("saga processing failed: context canceled"),
	)
	require.NoError(
		t,
		err,
		"MarkFailed should handle invalid transition gracefully when transaction is already in StateError",
	)

	// Transaction should still be in StateError
	assert.Equal(
		t,
		finitestate.StateError,
		tx.GetState(),
		"Transaction should remain in StateError",
	)
}

// TestTransactionManagerWithErroredTransaction tests the runner behavior
// when it tries to process an already-errored transaction during context cancellation
func TestTransactionManagerWithErroredTransaction(t *testing.T) {
	t.Parallel()

	// Create minimal config for testing
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	cfg.Version = config.VersionLatest

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})

	// Create a transaction and put it in StateError first
	tx, err := transaction.New(
		transaction.SourceTest,
		"TestTransactionManagerWithErroredTransaction",
		"test-request-id",
		cfg,
		handler,
	)
	require.NoError(t, err)

	// Put transaction in error state (simulating participant failure)
	err = tx.MarkError(
		errors.New("participant HTTPRunner failed to apply pending config: context canceled"),
	)
	require.NoError(t, err)
	assert.Equal(t, finitestate.StateError, tx.GetState())

	// Create a failing orchestrator that returns context canceled
	failingOrchestrator := &errorReturningOrchestrator{}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(t.Context())

	// Create transaction manager
	logger := slog.New(handler)
	runner, err := NewRunner(failingOrchestrator, WithLogger(logger))
	require.NoError(t, err)

	// Start the runner in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	// Wait for runner to be ready
	time.Sleep(100 * time.Millisecond)

	// Send the transaction that's already in StateError
	// The orchestrator will fail, triggering MarkFailed on an already-errored transaction
	select {
	case runner.GetTransactionSiphon() <- tx:
		// Transaction sent
	case <-time.After(1 * time.Second):
		t.Fatal("Failed to send transaction")
	}

	// Let the transaction get processed and fail
	time.Sleep(200 * time.Millisecond)

	// Cancel context to simulate shutdown
	cancel()

	// The test passes if the runner shuts down without hanging
	// (the original CI issue was shutdown timeouts)
	select {
	case err := <-errCh:
		require.NoError(t, err, "Runner should shutdown cleanly")
	case <-time.After(2 * time.Second):
		t.Fatal("Runner shutdown timed out - this reproduces the CI issue")
	}
}

// errorReturningOrchestrator always fails ProcessTransaction
type errorReturningOrchestrator struct{}

func (e *errorReturningOrchestrator) AddToStorage(tx *transaction.ConfigTransaction) error {
	return nil
}

func (e *errorReturningOrchestrator) ProcessTransaction(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return errors.New("saga processing failed: context canceled")
}

func (e *errorReturningOrchestrator) WaitForCompletion(ctx context.Context) error {
	return nil
}
