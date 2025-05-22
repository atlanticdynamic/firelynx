package txstorage

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTransaction(t *testing.T) *transaction.ConfigTransaction {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, nil)
	cfg := &config.Config{}
	tx, err := transaction.FromTest("test_transaction", cfg, handler)
	require.NoError(t, err)
	require.NotNil(t, tx)

	return tx
}

func TestNewTransactionStorage(t *testing.T) {
	t.Parallel()

	t.Run("creates storage with default options", func(t *testing.T) {
		storage := NewTransactionStorage()
		assert.NotNil(t, storage)
		assert.Equal(t, DefaultMaxTransactions, storage.maxTransactions)
		assert.Equal(t, DefaultCleanupDebounceInterval, storage.cleanupDebounceInterval)
		assert.NotNil(t, storage.cleanupFunc)
		assert.False(t, storage.asyncCleanup)
	})

	t.Run("applies custom options", func(t *testing.T) {
		maxTransactions := 50
		interval := 30 * time.Second
		asyncCleanup := true

		storage := NewTransactionStorage(
			WithMaxTransactions(maxTransactions),
			WithCleanupDebounceInterval(interval),
			WithAsyncCleanup(asyncCleanup),
		)

		assert.Equal(t, maxTransactions, storage.maxTransactions)
		assert.Equal(t, interval, storage.cleanupDebounceInterval)
		assert.Equal(t, asyncCleanup, storage.asyncCleanup)
	})

	t.Run("ignores invalid options", func(t *testing.T) {
		storage := NewTransactionStorage(
			WithMaxTransactions(-1),
			WithCleanupDebounceInterval(-1*time.Second),
		)

		assert.Equal(t, DefaultMaxTransactions, storage.maxTransactions)
		assert.Equal(t, DefaultCleanupDebounceInterval, storage.cleanupDebounceInterval)
	})
}

func TestTransactionStorageOperations(t *testing.T) {
	t.Parallel()

	t.Run("add and retrieve transactions", func(t *testing.T) {
		storage := NewTransactionStorage()
		tx := createTestTransaction(t)

		err := storage.Add(tx)
		require.NoError(t, err)

		txs := storage.GetAll()
		assert.Len(t, txs, 1)
		assert.Equal(t, tx, txs[0])

		retrieved := storage.GetByID(tx.ID.String())
		assert.Equal(t, tx, retrieved)
	})

	t.Run("set and get current transaction", func(t *testing.T) {
		storage := NewTransactionStorage()
		tx := createTestTransaction(t)

		// Initially current is nil
		assert.Nil(t, storage.GetCurrent())

		// Set current
		storage.SetCurrent(tx)
		assert.Equal(t, tx, storage.GetCurrent())

		// Current should be returned by GetByID even if not in history
		retrieved := storage.GetByID(tx.ID.String())
		assert.Equal(t, tx, retrieved)
	})

	t.Run("cleanup respects max transactions", func(t *testing.T) {
		maxTransactions := 3
		storage := NewTransactionStorage(WithMaxTransactions(maxTransactions))

		// Add more transactions than the max
		for i := 0; i < maxTransactions+2; i++ {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Should only keep the most recent maxTransactions
		txs := storage.GetAll()
		assert.Len(t, txs, maxTransactions)
	})

	t.Run("custom cleanup function works", func(t *testing.T) {
		customCleanupCalled := false
		customCleanup := func(txs []*transaction.ConfigTransaction) []*transaction.ConfigTransaction {
			customCleanupCalled = true
			// Just return the last one
			if len(txs) > 0 {
				return txs[len(txs)-1:]
			}
			return txs
		}

		storage := NewTransactionStorage(WithCleanupFunc(customCleanup))

		// Add a few transactions
		for i := 0; i < 3; i++ {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Custom cleanup should have been called
		assert.True(t, customCleanupCalled)

		// Should only keep the last one
		txs := storage.GetAll()
		assert.Len(t, txs, 1)
	})

	t.Run("handles nil transaction gracefully", func(t *testing.T) {
		storage := NewTransactionStorage()

		err := storage.Add(nil)
		assert.NoError(t, err)

		storage.SetCurrent(nil)
		assert.Nil(t, storage.GetCurrent())
	})
}

// This is a simple test for async cleanup - more comprehensive tests would use
// a context with timeout and channels for coordination
func TestAsyncCleanup(t *testing.T) {
	t.Parallel()

	t.Run("async cleanup runs after a delay", func(t *testing.T) {
		// Use a very short debounce interval
		interval := 10 * time.Millisecond
		storage := NewTransactionStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(interval),
			WithMaxTransactions(1),
		)

		// Add more than max transactions
		for i := 0; i < 3; i++ {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Should be cleaned up to just one transaction
		assert.Eventually(t, func() bool {
			return len(storage.GetAll()) == 1
		}, interval*5, interval)
	})
}
