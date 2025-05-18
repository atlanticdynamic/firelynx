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

	// 2. Begin execution phase
	if err := tx.BeginExecution(); err != nil {
		r.logger.Error("Failed to begin transaction execution",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// Future: Here we would distribute the configuration to all subscribers
	// for their execution phase. For now, we'll just mark it succeeded immediately.

	if err := tx.MarkSucceeded(); err != nil {
		r.logger.Error("Failed to mark transaction as succeeded",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// Perform the actual configuration update in a single atomic operation
	r.configMu.Lock()
	r.config = tx.GetConfig()
	r.txStorage.SetCurrent(tx)
	r.configMu.Unlock()

	// Begin the reload phase
	if err := tx.BeginReload(); err != nil {
		r.logger.Error("Failed to begin transaction reload",
			"id", tx.ID,
			"source", tx.Source,
			"error", err)
		return err
	}

	// Notify subscribers about the config change AFTER releasing the lock to avoid deadlocks
	r.triggerReload()

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
