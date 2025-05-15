// Package txstorage provides an implementation of the TransactionStorage interface
// for storing and retrieving configuration transactions.
package txstorage

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// DefaultMaxTransactions is the default number of transactions to keep in history
const DefaultMaxTransactions = 20

// DefaultCleanupDebounceInterval is the default time to wait before cleaning up old transactions
const DefaultCleanupDebounceInterval = 10 * time.Second

// Option is a functional option for configuring the TransactionStorage
type Option func(*TransactionStorage)

// WithMaxTransactions sets the maximum number of transactions to store
func WithMaxTransactions(max int) Option {
	return func(s *TransactionStorage) {
		if max > 0 {
			s.maxTransactions = max
		}
	}
}

// WithCleanupFunc sets a custom cleanup function
func WithCleanupFunc(
	fn func([]*transaction.ConfigTransaction) []*transaction.ConfigTransaction,
) Option {
	return func(s *TransactionStorage) {
		if fn != nil {
			s.cleanupFunc = fn
		}
	}
}

// WithAsyncCleanup enables or disables async cleanup
func WithAsyncCleanup(enabled bool) Option {
	return func(s *TransactionStorage) {
		s.asyncCleanup = enabled
	}
}

// WithCleanupDebounceInterval sets the cleanup debounce interval
func WithCleanupDebounceInterval(d time.Duration) Option {
	return func(s *TransactionStorage) {
		if d > 0 {
			s.cleanupDebounceInterval = d
		}
	}
}

// TransactionStorage provides a thread-safe storage for transactions
type TransactionStorage struct {
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
	// 0 = not running, 1 = running
	cleanupRunning int32
}

// NewTransactionStorage creates a new transaction storage with the given options
func NewTransactionStorage(opts ...Option) *TransactionStorage {
	s := &TransactionStorage{
		transactions:            make([]*transaction.ConfigTransaction, 0, 10),
		maxTransactions:         DefaultMaxTransactions,
		cleanupDebounceInterval: DefaultCleanupDebounceInterval,
		cleanupSignal:           make(chan struct{}, 1),
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
func (s *TransactionStorage) Add(tx *transaction.ConfigTransaction) error {
	if tx == nil {
		return nil
	}

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
func (s *TransactionStorage) SetCurrent(tx *transaction.ConfigTransaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = tx
}

// GetAll returns all transactions in storage
func (s *TransactionStorage) GetAll() []*transaction.ConfigTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the transactions slice to prevent modification
	result := make([]*transaction.ConfigTransaction, len(s.transactions))
	copy(result, s.transactions)

	return result
}

// GetByID returns a transaction by ID or nil if not found
func (s *TransactionStorage) GetByID(id string) *transaction.ConfigTransaction {
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
func (s *TransactionStorage) GetCurrent() *transaction.ConfigTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// signalCleanup signals the cleanup worker to run
func (s *TransactionStorage) signalCleanup() {
	// Start cleanup worker if not running
	if atomic.CompareAndSwapInt32(&s.cleanupRunning, 0, 1) {
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
func (s *TransactionStorage) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleanupFunc != nil {
		s.transactions = s.cleanupFunc(s.transactions)
	}
}

// cleanupWorker runs cleanup operations asynchronously
func (s *TransactionStorage) cleanupWorker() {
	defer atomic.StoreInt32(&s.cleanupRunning, 0)

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
