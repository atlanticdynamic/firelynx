// Package txmgr implements the transaction manager for configuration updates
// and adapters between domain config and runtime components.
//
// HTTP Listener Rewrite Plan:
// According to the HTTP listener rewrite plan, HTTP-specific configuration logic
// in this package will be moved to the HTTP listener package where it will implement
// the SagaParticipant interface. This will allow each SagaParticipant to handle
// its own configuration extraction and management, keeping this package focused
// on orchestration rather than HTTP-specific details.
package txmgr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Interface guards
var (
	_ supervisor.Runnable  = (*Runner)(nil)
	_ supervisor.Stateable = (*Runner)(nil)
)

// Runner implements the transaction manager using a siphon pattern
// following the design of httpcluster for clean configuration handling.
type Runner struct {
	// Transaction siphon channel - UNBUFFERED for ordering and synchronization
	txSiphon chan *transaction.ConfigTransaction

	// Saga orchestrator for processing
	sagaOrchestrator SagaProcessor

	// State management
	fsm finitestate.Machine

	// Context management
	parentCtx context.Context
	runCtx    context.Context
	runCancel context.CancelFunc

	// Options
	logger *slog.Logger
}

// NewRunner creates a new transaction manager runner with siphon pattern.
func NewRunner(
	sagaOrchestrator SagaProcessor,
	opts ...Option,
) (*Runner, error) {
	if sagaOrchestrator == nil {
		return nil, errors.New("saga orchestrator cannot be nil")
	}

	r := &Runner{
		sagaOrchestrator: sagaOrchestrator,
		txSiphon:         make(chan *transaction.ConfigTransaction), // UNBUFFERED
		logger:           slog.Default().WithGroup("txmgr.Runner"),
		parentCtx:        context.Background(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create FSM
	fsmLogger := r.logger.WithGroup("fsm")
	machine, err := finitestate.New(fsmLogger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create FSM: %w", err)
	}
	r.fsm = machine

	return r, nil
}

// GetTransactionSiphon returns the transaction siphon for sending transactions.
// The channel is unbuffered, so sends will block until the receiver is ready.
func (r *Runner) GetTransactionSiphon() chan<- *transaction.ConfigTransaction {
	return r.txSiphon
}

// Run implements the supervisor.Runnable interface.
func (r *Runner) Run(ctx context.Context) error {
	logger := r.logger.WithGroup("Run")
	logger.Debug("Starting transaction manager")

	if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
		return fmt.Errorf("failed to transition to booting: %w", err)
	}

	runCtx, runCancel := context.WithCancel(ctx)
	r.runCtx = runCtx
	r.runCancel = runCancel
	defer runCancel()

	// Transition to running - we're ready to receive on the siphon
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running: %w", err)
	}

	logger.Info("Transaction manager ready")

	// Main event loop
	for {
		select {
		case <-runCtx.Done():
			logger.Debug("Run context cancelled")
			return r.shutdown(runCtx)

		case <-r.parentCtx.Done():
			logger.Debug("Parent context cancelled")
			return r.shutdown(runCtx)

		case tx, ok := <-r.txSiphon:
			if !ok {
				logger.Debug("Transaction siphon closed")
				return r.shutdown(runCtx)
			}

			logger.Debug("Received transaction", "id", tx.ID)
			if err := r.processTransaction(runCtx, tx); err != nil {
				logger.Error("Failed to process transaction",
					"id", tx.ID, "error", err)
				// Mark transaction as failed but continue running
				if markErr := tx.MarkFailed(err); markErr != nil {
					logger.Error("Failed to mark transaction as failed",
						"id", tx.ID, "error", markErr)
				}
			}
		}
	}
}

// Stop signals the transaction manager to stop.
func (r *Runner) Stop() {
	r.logger.Debug("Stop called")
	if r.runCancel != nil {
		r.runCancel()
	}
}

// shutdown performs graceful shutdown of the transaction manager.
func (r *Runner) shutdown(ctx context.Context) error {
	logger := r.logger.WithGroup("shutdown")
	logger.Info("Transaction manager shutting down")

	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		logger.Error("Failed to transition to stopping", "error", err)
	}

	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		logger.Error("Failed to transition to stopped", "error", err)
	}

	return nil
}

// processTransaction handles a configuration transaction through the saga orchestrator.
func (r *Runner) processTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error {
	logger := r.logger.WithGroup("processTransaction")

	if err := r.fsm.Transition(finitestate.StatusReloading); err != nil {
		return fmt.Errorf("failed to transition to reloading: %w", err)
	}

	if err := r.sagaOrchestrator.AddToStorage(tx); err != nil {
		if transErr := r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning); transErr != nil {
			logger.Error("Failed to return to running state", "error", transErr)
		}
		return fmt.Errorf("failed to store transaction: %w", err)
	}

	if err := r.sagaOrchestrator.ProcessTransaction(ctx, tx); err != nil {
		if transErr := r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning); transErr != nil {
			logger.Error("Failed to return to running state", "error", transErr)
		}
		return fmt.Errorf("saga processing failed: %w", err)
	}

	if err := r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning); err != nil {
		logger.Error("Failed to return to running state", "error", err)
	}

	logger.Info("Successfully processed transaction", "id", tx.ID)
	return nil
}

// String returns the name of this runnable component.
func (r *Runner) String() string {
	return "txmgr.Runner"
}
