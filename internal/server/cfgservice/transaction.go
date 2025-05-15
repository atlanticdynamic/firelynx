package cfgservice

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/cfgservice/server"
)

// createFileTransaction creates a new transaction from a file configuration.
// The transaction is added to storage but not yet validated.
func (r *Runner) createFileTransaction(
	path string,
	cfg *config.Config,
) (*transaction.ConfigTransaction, error) {
	tx, err := transaction.FromFile(path, cfg, r.logger.Handler())
	if err != nil {
		return nil, err
	}

	if err := r.txStorage.Add(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

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
// through preparation, commitment, and completion. It ensures atomic updates
// and proper error handling for configuration changes.
func (r *Runner) processTransaction(tx *transaction.ConfigTransaction) error {
	// 1. Validate the transaction
	if err := tx.RunValidation(); err != nil {
		r.logger.Error("Transaction validation failed",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// 2. Begin preparation phase
	if err := tx.BeginPreparation(); err != nil {
		r.logger.Error("Failed to begin transaction preparation",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// Future: Here we would distribute the configuration to all subscribers
	// for their preparation phase. For now, we'll just mark it prepared immediately.

	if err := tx.MarkPrepared(); err != nil {
		r.logger.Error("Failed to mark transaction as prepared",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// 3. Begin commit phase
	if err := tx.BeginCommit(); err != nil {
		r.logger.Error("Failed to begin transaction commit",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// Perform the actual configuration update in a single atomic operation
	r.configMu.Lock()
	r.config = tx.GetConfig()
	r.txStorage.SetCurrent(tx)
	// Notify subscribers about the config change while still holding the lock
	r.triggerReload()
	r.configMu.Unlock()

	// 4. Mark the transaction as committed
	if err := tx.MarkCommitted(); err != nil {
		r.logger.Error("Failed to mark transaction as committed",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		// Note: At this point we're in a tricky situation since we've already
		// updated the configuration. In a true saga, we would need to roll back.
		// For now, we'll just log the error but not attempt to roll back.
		return err
	}

	// 5. Mark the transaction as completed
	if err := tx.MarkCompleted(); err != nil {
		r.logger.Error("Failed to mark transaction as completed",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	r.logger.Info("Transaction completed successfully",
		"id", tx.ID,
		"source", tx.Source,
		"sourceDetail", tx.SourceDetail)
	return nil
}
