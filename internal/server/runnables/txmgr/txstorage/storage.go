// Package txstorage provides an implementation of the TransactionStorage interface
// for storing and retrieving configuration transactions.
package txstorage

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
)

// DefaultMaxTransactions is the default number of transactions to keep in history
const DefaultMaxTransactions = 20

// DefaultCleanupDebounceInterval is the default time to wait before cleaning up old transactions
const DefaultCleanupDebounceInterval = 10 * time.Second

// MemoryStorage provides a thread-safe storage for transactions
type MemoryStorage struct {
	// Stored transactions
	transactions []*transaction.ConfigTransaction

	// Current active transaction
	current *transaction.ConfigTransaction

	// Mutex to protect access to transactions and current
	mu sync.RWMutex

	// Maximum number of transactions to store
	maxTransactions int

	// Function to clean up transactions (e.g., remove old ones)
	cleanupFunc func([]*transaction.ConfigTransaction) []*transaction.ConfigTransaction

	// Whether to use async cleanup
	asyncCleanup bool

	// Time to wait before cleaning up
	cleanupDebounceInterval time.Duration

	// Channel to signal cleanup
	cleanupSignal chan struct{}

	// Indicates if cleanup worker is running
	cleanupRunning atomic.Bool

	logger *slog.Logger
}

// NewMemoryStorage creates a new transaction storage with the given options
func NewMemoryStorage(opts ...Option) *MemoryStorage {
	s := &MemoryStorage{
		transactions:            make([]*transaction.ConfigTransaction, 0, 10),
		maxTransactions:         DefaultMaxTransactions,
		cleanupDebounceInterval: DefaultCleanupDebounceInterval,
		cleanupSignal:           make(chan struct{}, 1),
		logger:                  slog.Default().WithGroup("txstorage"),
	}

	// Default cleanup function: keep only the last maxTransactions
	s.cleanupFunc = func(txs []*transaction.ConfigTransaction) []*transaction.ConfigTransaction {
		if len(txs) <= s.maxTransactions {
			return txs
		}
		return txs[len(txs)-s.maxTransactions:]
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Add adds a transaction to storage
func (s *MemoryStorage) Add(tx *transaction.ConfigTransaction) error {
	if tx == nil {
		return nil
	}
	s.logger.WithGroup("Add").Debug("Adding transaction", "id", tx.ID.String())

	s.mu.Lock()
	s.transactions = append(s.transactions, tx)
	s.mu.Unlock()

	// Schedule cleanup if needed
	if s.asyncCleanup {
		s.signalCleanup()
	} else {
		s.cleanup()
	}

	return nil
}

// SetCurrent sets the current active transaction
func (s *MemoryStorage) SetCurrent(tx *transaction.ConfigTransaction) {
	logger := s.logger.WithGroup("SetCurrent")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = tx

	if tx != nil {
		logger.Debug("Setting current transaction", "id", tx.ID.String())
	} else {
		logger.Debug("Clearing current transaction")
	}
}

// GetAll returns all transactions in storage
func (s *MemoryStorage) GetAll() []*transaction.ConfigTransaction {
	s.logger.Debug("Getting all transactions")
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the transactions slice to prevent modification
	result := make([]*transaction.ConfigTransaction, len(s.transactions))
	copy(result, s.transactions)

	return result
}

// GetByID returns a transaction by ID or nil if not found
func (s *MemoryStorage) GetByID(id string) *transaction.ConfigTransaction {
	s.logger.Debug("Getting transaction by ID", "id", id)
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check current first
	if s.current != nil && s.current.ID.String() == id {
		return s.current
	}

	// Check history
	for _, tx := range s.transactions {
		if tx.ID.String() == id {
			return tx
		}
	}

	return nil
}

// GetCurrent returns the current active transaction
func (s *MemoryStorage) GetCurrent() *transaction.ConfigTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// signalCleanup signals the cleanup worker to run
func (s *MemoryStorage) signalCleanup() {
	// Start cleanup worker if not running
	if s.cleanupRunning.CompareAndSwap(false, true) {
		go s.cleanupWorker()
	}

	// Signal cleanup non-blocking
	select {
	case s.cleanupSignal <- struct{}{}:
	default:
		// Channel full, ignore
	}
}

// cleanup applies the cleanup function to the transactions
func (s *MemoryStorage) cleanup() {
	logger := s.logger.WithGroup("cleanup")
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Debug("Starting cleanup", "transactions", len(s.transactions))
	if s.cleanupFunc != nil {
		s.transactions = s.cleanupFunc(s.transactions)
	}
	logger.Debug("Finished cleanup", "transactions", len(s.transactions))
}

// cleanupWorker runs cleanup operations asynchronously
func (s *MemoryStorage) cleanupWorker() {
	defer s.cleanupRunning.Store(false)

	// Create timer for debounce
	timer := time.NewTimer(s.cleanupDebounceInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.cleanupSignal:
			// Reset timer when a new signal comes in
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.cleanupDebounceInterval)

		case <-timer.C:
			// Timer expired, run cleanup
			s.cleanup()
			return
		}
	}
}

// Clear removes transactions from storage that are in terminal states,
// keeping at least the last N transactions total.
// Returns the number of transactions cleared.
func (s *MemoryStorage) Clear(keepLast int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if keepLast < 0 {
		return 0, fmt.Errorf("keepLast must be non-negative, got %d", keepLast)
	}

	total := len(s.transactions)
	if total <= keepLast {
		s.logger.Debug("No transactions to clear", "total", total, "keepLast", keepLast)
		return 0, nil
	}

	// Number of transactions we need to delete
	toDelete := total - keepLast
	deleted := 0

	// Build new list, deleting old terminal transactions
	newTransactions := make([]*transaction.ConfigTransaction, 0, keepLast)

	for _, tx := range s.transactions {
		// Delete if we still need to delete more and it's terminal
		if deleted < toDelete && slices.Contains(finitestate.SagaTerminalStates, tx.GetState()) {
			deleted++
			continue
		}
		// Keep everything else
		newTransactions = append(newTransactions, tx)
	}

	s.transactions = newTransactions

	s.logger.WithGroup("Clear").
		Info("Cleared transactions", "cleared", deleted, "remaining", len(s.transactions))
	return deleted, nil
}
