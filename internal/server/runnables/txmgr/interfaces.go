package txmgr

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// SagaProcessor processes configuration transactions through the saga pattern.
type SagaProcessor interface {
	// ProcessTransaction processes a validated transaction through the saga lifecycle.
	ProcessTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error

	// AddToStorage adds a transaction to the transaction storage.
	AddToStorage(tx *transaction.ConfigTransaction) error

	// WaitForCompletion waits for the current transaction to reach a terminal state.
	// Returns immediately if no transaction is in progress.
	WaitForCompletion(ctx context.Context) error
}
