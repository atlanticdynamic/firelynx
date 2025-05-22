// TODO: these are extra, remove them later
package cfgservice

import (
	"context"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice/server"
)

// createAPITransaction creates a new transaction from an API request.
// The transaction is added to storage but not yet validated.
// It extracts the request ID from the context or generates a new one.
func (r *Runner) createAPITransaction(
	ctx context.Context,
	cfg *config.Config,
) (*transaction.ConfigTransaction, error) {
	// Extract request ID from context or generate a new one
	requestID := server.ExtractRequestID(ctx)

	tx, err := transaction.FromAPI(requestID, cfg, r.logger.Handler())
	if err != nil {
		return nil, err
	}

	if err := r.txStorage.Add(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// processTransaction orchestrates the complete transaction lifecycle from validation
// through preparation, commitment, and completion. It delegates the transaction
// processing to the ConfigOrchestrator.
func (r *Runner) processTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error {
	logger := r.logger.With("id", tx.ID, "source", tx.Source)

	// 1. Validate the transaction
	if err := tx.RunValidation(); err != nil {
		return fmt.Errorf("failed to validate transaction: %w", err)
	}

	// 2. Delegate transaction processing to the orchestrator
	if err := r.orchestrator.ProcessTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to process transaction: %w", err)
	}

	logger.Debug(
		"Transaction processed successfully via orchestrator",
		"sourceDetail",
		tx.SourceDetail,
	)
	return nil
}
