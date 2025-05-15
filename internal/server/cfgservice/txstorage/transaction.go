// Package txstorage provides storage interfaces and implementations for managing
// configuration transactions. It allows for storing, retrieving, and querying
// transaction objects throughout their lifecycle.
package txstorage

import (
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// Record represents a stored transaction with its ID and the transaction itself
type Record struct {
	// ID is the unique identifier for this transaction
	ID string

	// Transaction is the actual ConfigTransaction object
	Transaction *transaction.ConfigTransaction
}
