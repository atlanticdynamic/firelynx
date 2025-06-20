package txmgr

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSagaProcessor is a test implementation of SagaProcessor
type mockSagaProcessor struct {
	tb               testing.TB
	txStorage        *txstorage.MemoryStorage
	waitDuration     time.Duration
	waitForCompleted chan struct{}
}

func newMockSagaProcessor(
	tb testing.TB,
	txStorage *txstorage.MemoryStorage,
	waitDuration time.Duration,
) *mockSagaProcessor {
	tb.Helper()
	return &mockSagaProcessor{
		tb:               tb,
		txStorage:        txStorage,
		waitDuration:     waitDuration,
		waitForCompleted: make(chan struct{}),
	}
}

func (m *mockSagaProcessor) ProcessTransaction(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	m.tb.Helper()
	if err := tx.MarkSucceeded(); err != nil {
		return err
	}
	m.txStorage.SetCurrent(tx)
	return nil
}

func (m *mockSagaProcessor) AddToStorage(tx *transaction.ConfigTransaction) error {
	m.tb.Helper()
	err := m.txStorage.Add(tx)
	require.NoError(m.tb, err, "failed to add transaction to storage")
	return nil
}

func (m *mockSagaProcessor) WaitForCompletion(ctx context.Context) error {
	m.tb.Helper()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.waitDuration):
		close(m.waitForCompleted)
		return nil
	}
}

func TestRunnerShutdownTimeout(t *testing.T) {
	t.Run("shutdown respects custom timeout", func(t *testing.T) {
		txStorage := txstorage.NewMemoryStorage()
		// Create a mock that waits 2 seconds
		mockSaga := newMockSagaProcessor(t, txStorage, 2*time.Second)

		// Create runner with 100ms timeout
		runner, err := NewRunner(
			mockSaga,
			WithSagaOrchestratorShutdownTimeout(100*time.Millisecond),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		// Wait for runner to start
		assert.Eventually(t, func() bool {
			return runner.IsRunning()
		}, 5*time.Second, 10*time.Millisecond)

		// Cancel context to trigger shutdown
		cancel()

		// Assert that WaitForCompletion was NOT completed within 1 second
		assert.Never(t, func() bool {
			select {
			case <-mockSaga.waitForCompleted:
				return true
			default:
				return false
			}
		}, 1*time.Second, 10*time.Millisecond, "WaitForCompletion should not complete due to timeout")

		// Verify shutdown completed with timeout error
		select {
		case err := <-errCh:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "context deadline exceeded")
		case <-time.After(1 * time.Second):
			t.Fatal("Runner did not complete within expected time")
		}
	})

	t.Run("shutdown completes when saga finishes quickly", func(t *testing.T) {
		txStorage := txstorage.NewMemoryStorage()
		// Create a mock that completes immediately
		mockSaga := newMockSagaProcessor(t, txStorage, 0)

		// Create runner with default timeout
		runner, err := NewRunner(mockSaga)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		// Wait for runner to start
		assert.Eventually(t, func() bool {
			return runner.IsRunning()
		}, 5*time.Second, 10*time.Millisecond)

		// Cancel context to trigger shutdown
		cancel()

		// Verify shutdown completed successfully
		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("Runner did not complete within expected time")
		}

		// Verify WaitForCompletion was completed
		select {
		case <-mockSaga.waitForCompleted:
			// Success
		default:
			t.Fatal("WaitForCompletion should have completed")
		}
	})
}
