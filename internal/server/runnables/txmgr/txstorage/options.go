package txstorage

import (
	"log/slog"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// Option is a functional option for configuring the TransactionStorage
type Option func(*MemoryStorage)

// WithMaxTransactions sets the maximum number of transactions to store
func WithMaxTransactions(max int) Option {
	return func(s *MemoryStorage) {
		if max > 0 {
			s.maxTransactions = max
		}
	}
}

// WithCleanupFunc sets a custom cleanup function
func WithCleanupFunc(
	fn func([]*transaction.ConfigTransaction) []*transaction.ConfigTransaction,
) Option {
	return func(s *MemoryStorage) {
		if fn != nil {
			s.cleanupFunc = fn
		}
	}
}

// WithAsyncCleanup enables or disables async cleanup
func WithAsyncCleanup(enabled bool) Option {
	return func(s *MemoryStorage) {
		s.asyncCleanup = enabled
	}
}

// WithCleanupDebounceInterval sets the cleanup debounce interval
func WithCleanupDebounceInterval(d time.Duration) Option {
	return func(s *MemoryStorage) {
		if d > 0 {
			s.cleanupDebounceInterval = d
		}
	}
}

// WithLogHandler sets the log handler for the storage
func WithLogHandler(handler slog.Handler) Option {
	return func(s *MemoryStorage) {
		if handler != nil {
			s.logger = slog.New(handler)
		}
	}
}

// WithLogger sets the logger for the storage
func WithLogger(logger *slog.Logger) Option {
	return func(s *MemoryStorage) {
		s.logger = logger
	}
}
