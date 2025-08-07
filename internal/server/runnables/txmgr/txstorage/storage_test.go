package txstorage

import (
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTransaction(t *testing.T) *transaction.ConfigTransaction {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, nil)
	// Create config using proper constructor
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	tx, err := transaction.FromTest("test_transaction", cfg, handler)
	require.NoError(t, err)
	require.NotNil(t, tx)

	return tx
}

func TestNewTransactionStorage(t *testing.T) {
	t.Parallel()

	t.Run("creates storage with default options", func(t *testing.T) {
		storage := NewMemoryStorage()
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

		storage := NewMemoryStorage(
			WithMaxTransactions(maxTransactions),
			WithCleanupDebounceInterval(interval),
			WithAsyncCleanup(asyncCleanup),
		)

		assert.Equal(t, maxTransactions, storage.maxTransactions)
		assert.Equal(t, interval, storage.cleanupDebounceInterval)
		assert.Equal(t, asyncCleanup, storage.asyncCleanup)
	})

	t.Run("ignores invalid options", func(t *testing.T) {
		storage := NewMemoryStorage(
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
		storage := NewMemoryStorage()
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
		storage := NewMemoryStorage()
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
		storage := NewMemoryStorage(WithMaxTransactions(maxTransactions))

		// Add more transactions than the max
		for range maxTransactions + 2 {
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

		storage := NewMemoryStorage(WithCleanupFunc(customCleanup))

		// Add a few transactions
		for range 3 {
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
		storage := NewMemoryStorage()

		err := storage.Add(nil)
		assert.NoError(t, err)

		storage.SetCurrent(nil)
		assert.Nil(t, storage.GetCurrent())
	})
}

func TestAsyncCleanup(t *testing.T) {
	t.Parallel()

	t.Run("async cleanup runs after a delay", func(t *testing.T) {
		// Use a very short debounce interval
		interval := 10 * time.Millisecond
		storage := NewMemoryStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(interval),
			WithMaxTransactions(1),
		)

		// Add more than max transactions
		for range 3 {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Should be cleaned up to just one transaction
		assert.Eventually(t, func() bool {
			return len(storage.GetAll()) == 1
		}, interval*5, interval)
	})

	t.Run("cleanup worker only starts once", func(t *testing.T) {
		interval := 50 * time.Millisecond
		storage := NewMemoryStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(interval),
			WithMaxTransactions(1),
		)

		// Initially cleanup worker is not running
		assert.False(t, storage.cleanupRunning.Load())

		// Add transaction to trigger cleanup
		tx1 := createTestTransaction(t)
		err := storage.Add(tx1)
		require.NoError(t, err)

		// Cleanup worker should be running
		assert.True(t, storage.cleanupRunning.Load())

		// Adding more transactions shouldn't start another worker
		tx2 := createTestTransaction(t)
		err = storage.Add(tx2)
		require.NoError(t, err)

		// Wait for cleanup to complete
		assert.Eventually(t, func() bool {
			return !storage.cleanupRunning.Load()
		}, interval*3, time.Millisecond)

		// Should only have one transaction after cleanup
		assert.Len(t, storage.GetAll(), 1)
	})

	t.Run("cleanup worker can be restarted after completion", func(t *testing.T) {
		interval := 20 * time.Millisecond
		storage := NewMemoryStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(interval),
			WithMaxTransactions(2),
		)

		// First batch
		for range 3 {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Wait for first cleanup to complete
		assert.Eventually(t, func() bool {
			return !storage.cleanupRunning.Load() && len(storage.GetAll()) == 2
		}, interval*3, time.Millisecond)

		// Second batch - should be able to start worker again
		for range 2 {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Wait for second cleanup to complete
		assert.Eventually(t, func() bool {
			return !storage.cleanupRunning.Load() && len(storage.GetAll()) == 2
		}, interval*3, time.Millisecond)
	})

	t.Run("debounce mechanism works correctly", func(t *testing.T) {
		interval := 100 * time.Millisecond
		storage := NewMemoryStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(interval),
			WithMaxTransactions(1),
		)

		// Add multiple transactions rapidly
		for range 5 {
			tx := createTestTransaction(t)
			err := storage.Add(tx)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Small delay between additions
		}

		// Should have more than max before cleanup
		assert.Greater(t, len(storage.GetAll()), 1)

		// Wait for debounce and cleanup
		assert.Eventually(t, func() bool {
			return len(storage.GetAll()) == 1
		}, interval*2, 10*time.Millisecond)
	})
}

func TestCleanupWorkerLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("signal cleanup with full channel", func(t *testing.T) {
		storage := NewMemoryStorage(
			WithAsyncCleanup(true),
			WithCleanupDebounceInterval(50*time.Millisecond),
		)

		// Fill the cleanup signal channel
		storage.cleanupSignal <- struct{}{}

		// Should still be able to signal cleanup without blocking
		tx := createTestTransaction(t)
		err := storage.Add(tx)
		require.NoError(t, err)

		// Cleanup should still work
		assert.Eventually(t, func() bool {
			return !storage.cleanupRunning.Load()
		}, 200*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("cleanup function is nil safe", func(t *testing.T) {
		storage := NewMemoryStorage(WithCleanupFunc(nil))

		tx := createTestTransaction(t)
		err := storage.Add(tx)
		require.NoError(t, err)

		// Should not panic
		storage.cleanup()

		// Transaction should still be there
		assert.Len(t, storage.GetAll(), 1)
	})
}

func TestConcurrentOperations(t *testing.T) {
	t.Parallel()

	t.Run("concurrent add and get operations", func(t *testing.T) {
		// Use a large max to avoid cleanup during concurrent operations
		storage := NewMemoryStorage(WithMaxTransactions(1000))

		// Add transactions concurrently
		const numGoroutines = 10
		const transactionsPerGoroutine = 5

		done := make(chan struct{}, numGoroutines)

		for range numGoroutines {
			go func() {
				defer func() { done <- struct{}{} }()
				for range transactionsPerGoroutine {
					tx := createTestTransaction(t)
					err := storage.Add(tx)
					assert.NoError(t, err)
				}
			}()
		}

		// Wait for all goroutines to complete
		for range numGoroutines {
			<-done
		}

		// Should have all transactions
		txs := storage.GetAll()
		assert.Len(t, txs, numGoroutines*transactionsPerGoroutine)
	})

	t.Run("concurrent current transaction operations", func(t *testing.T) {
		storage := NewMemoryStorage()

		const numGoroutines = 10
		done := make(chan struct{}, numGoroutines)

		tx := createTestTransaction(t)

		// Set and get current transaction concurrently
		for range numGoroutines {
			go func() {
				defer func() { done <- struct{}{} }()
				storage.SetCurrent(tx)
				current := storage.GetCurrent()
				assert.Equal(t, tx, current)
			}()
		}

		// Wait for all goroutines to complete
		for range numGoroutines {
			<-done
		}
	})
}

func TestDefaultCleanupBehavior(t *testing.T) {
	t.Parallel()

	t.Run("default cleanup keeps most recent transactions", func(t *testing.T) {
		maxTx := 3
		storage := NewMemoryStorage(WithMaxTransactions(maxTx))

		var transactions []*transaction.ConfigTransaction

		// Add more transactions than max
		for range 5 {
			tx := createTestTransaction(t)
			transactions = append(transactions, tx)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Should keep only the last maxTx transactions
		result := storage.GetAll()
		assert.Len(t, result, maxTx)

		// Should be the most recent ones
		for i, tx := range result {
			expected := transactions[len(transactions)-maxTx+i]
			assert.Equal(t, expected.ID.String(), tx.ID.String())
		}
	})

	t.Run("cleanup with fewer than max transactions", func(t *testing.T) {
		storage := NewMemoryStorage(WithMaxTransactions(5))

		// Add fewer transactions than max
		var transactions []*transaction.ConfigTransaction
		for range 3 {
			tx := createTestTransaction(t)
			transactions = append(transactions, tx)
			err := storage.Add(tx)
			require.NoError(t, err)
		}

		// Should keep all transactions
		result := storage.GetAll()
		assert.Len(t, result, 3)

		for i, tx := range result {
			assert.Equal(t, transactions[i].ID.String(), tx.ID.String())
		}
	})
}

func TestTransactionRetrieval(t *testing.T) {
	t.Parallel()

	t.Run("get by id returns current transaction even if not in history", func(t *testing.T) {
		storage := NewMemoryStorage()

		tx := createTestTransaction(t)
		storage.SetCurrent(tx)

		// Should find current transaction even though it's not in history
		result := storage.GetByID(tx.ID.String())
		assert.Equal(t, tx, result)
	})

	t.Run("get by id returns nil for non-existent transaction", func(t *testing.T) {
		storage := NewMemoryStorage()

		result := storage.GetByID("non-existent-id")
		assert.Nil(t, result)
	})

	t.Run("get by id prefers current over history", func(t *testing.T) {
		storage := NewMemoryStorage()

		// Add transaction to history
		tx1 := createTestTransaction(t)
		err := storage.Add(tx1)
		require.NoError(t, err)

		// Create new transaction with same ID logic won't work due to UUID
		// Instead test that current is returned when both exist
		tx2 := createTestTransaction(t)
		storage.SetCurrent(tx2)

		// Should return current
		result := storage.GetByID(tx2.ID.String())
		assert.Equal(t, tx2, result)

		// Should also return history transaction by its ID
		result = storage.GetByID(tx1.ID.String())
		assert.Equal(t, tx1, result)
	})

	t.Run("get all returns copy of transactions", func(t *testing.T) {
		storage := NewMemoryStorage()

		tx := createTestTransaction(t)
		err := storage.Add(tx)
		require.NoError(t, err)

		result := storage.GetAll()

		// Modifying result should not affect storage
		result[0] = nil

		storageResult := storage.GetAll()
		assert.Equal(t, tx, storageResult[0])
	})
}
