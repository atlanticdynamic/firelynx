package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Default timeout values
const (
	// DefaultReloadTimeout is the timeout to wait for a component to become ready after reload
	DefaultReloadTimeout = 30 * time.Second
	// DefaultReloadRetryInterval is the interval to check component readiness
	DefaultReloadRetryInterval = 10 * time.Millisecond
)

// SagaParticipant defines the interface for components participating
// in configuration transactions. It extends Runnable and Stateable from supervisor
// to ensure components have the necessary lifecycle management capabilities.
// Note that SagaParticipant SHOULD NOT implement supervisor.Reloadable to avoid
// conflicts with the CommitConfig method.
type SagaParticipant interface {
	supervisor.Runnable
	supervisor.Stateable

	// StageConfig processes a validated configuration transaction
	// by preparing the component to apply the changes. This is called
	// during the execution phase of the saga.
	StageConfig(ctx context.Context, tx *transaction.ConfigTransaction) error

	// CompensateConfig reverts changes made during StageConfig
	// when a transaction fails. This is called for successful
	// participants when the saga needs to be rolled back.
	CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error

	// CommitConfig applies the pending configuration prepared during StageConfig.
	// This is called during the reload phase after all participants have successfully
	// executed their configurations.
	CommitConfig(ctx context.Context) error
}

// SagaOrchestrator coordinates configuration changes across multiple components
// using the saga pattern. It maintains participant state tracking and handles
// compensation if any component fails.
type SagaOrchestrator struct {
	// Transaction storage for persistent transaction state
	txStorage *txstorage.MemoryStorage

	// Registry of saga participants
	runnables map[string]SagaParticipant

	// Logger
	logger *slog.Logger

	// Internal state
	mutex sync.RWMutex
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(
	txStorage *txstorage.MemoryStorage,
	handler slog.Handler,
) *SagaOrchestrator {
	logger := slog.New(handler).WithGroup("sagaOrchestrator")

	return &SagaOrchestrator{
		txStorage: txStorage,
		runnables: make(map[string]SagaParticipant),
		logger:    logger,
	}
}

// RegisterParticipant registers a component as a saga participant.
// Returns an error if the participant also implements supervisor.Reloadable which would
// cause conflicts with CommitConfig.
func (o *SagaOrchestrator) RegisterParticipant(participant SagaParticipant) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	name := participant.String()

	// Check if the participant also implements supervisor.Reloadable
	// This is a conflict that would lead to double reloading
	if _, isReloadable := participant.(supervisor.Reloadable); isReloadable {
		return fmt.Errorf(
			"participant %s implements supervisor.Reloadable which conflicts with SagaParticipant - remove Reloadable implementation",
			name,
		)
	}

	o.runnables[name] = participant
	o.logger.Debug("Registered saga participant", "name", name)
	return nil
}

// ProcessTransaction processes a validated transaction through the saga lifecycle
// TODO: consider passing the transaction ID instead of the pointer to the transaction, then load it from storage
func (o *SagaOrchestrator) ProcessTransaction(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Only process validated transactions
	if tx.GetState() != finitestate.StateValidated {
		return fmt.Errorf("transaction is not in validated state: %s", tx.GetState())
	}

	// Begin execution phase
	if err := tx.BeginExecution(); err != nil {
		return fmt.Errorf("failed to begin execution: %w", err)
	}

	// Skip waiting for participants during initial startup
	// The channels exist and that's all that matters for communication
	// Components will process configs when they're ready
	o.logger.Debug("Processing transaction without waiting for participant states",
		"participantCount", len(o.runnables))

	// Register all known participants with the transaction
	o.mutex.RLock()
	for name := range o.runnables {
		if err := tx.RegisterParticipant(name); err != nil {
			o.mutex.RUnlock()
			return fmt.Errorf("failed to register participant %s with transaction: %w", name, err)
		}
	}
	participants := o.runnables
	o.mutex.RUnlock()

	// Get sorted participant names for deterministic ordering
	names := o.getSortedParticipantNames()

	// Process each participant
	for _, name := range names {
		participant := participants[name]

		// Wait for participant to be running before processing
		if err := o.waitForRunning(ctx, participant, name); err != nil {
			o.logger.Error("Participant not ready", "name", name, "error", err)
			// Get participant state tracker to mark as failed
			if participantState, err := tx.GetParticipants().GetOrCreate(name); err == nil {
				if markErr := participantState.MarkFailed(fmt.Errorf("not ready: %w", err)); markErr != nil {
					o.logger.Error(
						"Failed to mark participant as failed",
						"name",
						name,
						"error",
						markErr,
					)
				}
			}
			continue
		}

		// Get participant state tracker from the transaction
		participantState, err := tx.GetParticipants().GetOrCreate(name)
		if err != nil {
			o.logger.Error("Failed to create participant state", "name", name, "error", err)
			continue
		}

		// Start execution for this participant
		if err := participantState.Execute(); err != nil {
			o.logger.Error("Failed to start execution for participant",
				"name", name, "error", err)
			continue
		}

		// Execute the configuration on this participant
		err = participant.StageConfig(ctx, tx)
		if err != nil {
			// Mark participant as failed
			if markErr := participantState.MarkFailed(err); markErr != nil {
				o.logger.Error("Failed to mark participant as failed",
					"name", name, "error", markErr, "originalError", err)
			}
			// Mark transaction as failed
			if markErr := tx.MarkFailed(ctx, err); markErr != nil {
				o.logger.Error("Failed to mark transaction as failed",
					"error", markErr, "originalError", err)
			}
			// Begin compensation for successful participants
			o.compensateParticipants(ctx, tx)
			return err
		}

		// Mark participant as succeeded
		if err := participantState.MarkSucceeded(); err != nil {
			o.logger.Error("Failed to mark participant as succeeded", "name", name, "error", err)
		}
	}

	// Check if all participants succeeded
	if tx.GetParticipants().AllParticipantsSucceeded() {
		// Mark transaction as succeeded
		if err := tx.MarkSucceeded(); err != nil {
			return fmt.Errorf("failed to mark transaction as succeeded: %w", err)
		}

		// Set as current in transaction storage
		o.txStorage.SetCurrent(tx)

		// Trigger reload of all participants
		// If reload fails, the transaction will be marked as error by TriggerReload
		if err := o.TriggerReload(ctx); err != nil {
			o.logger.Error("Reload failed after transaction execution",
				"id", tx.ID, "error", err)
			return fmt.Errorf("transaction execution succeeded but reload failed: %w", err)
		}

		o.logger.Debug("Transaction and reload completed successfully", "id", tx.ID)
		return nil
	}

	// If we got here but not all participants succeeded, something went wrong
	return fmt.Errorf("not all participants succeeded, but no specific error was reported")
}

// compensateParticipants triggers compensation for all successful participants
func (o *SagaOrchestrator) compensateParticipants(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) {
	// Transition transaction to compensating state
	if err := tx.BeginCompensation(); err != nil {
		o.logger.Error("Failed to begin compensation", "error", err, "currentState", tx.GetState())
		return
	}

	// Begin compensation for all participants
	if err := tx.GetParticipants().BeginCompensation(); err != nil {
		o.logger.Error("Error starting compensation for participants", "error", err)
	}

	// Execute compensation for each component that supports it
	o.mutex.RLock()
	participants := o.runnables
	o.mutex.RUnlock()

	// Get sorted participant names for deterministic ordering
	names := o.getSortedParticipantNames()

	// Process each participant
	for _, name := range names {
		participant := participants[name]
		// Get participant state
		participantState, err := tx.GetParticipants().GetOrCreate(name)
		if err != nil {
			o.logger.Error("Failed to get participant state", "name", name, "error", err)
			continue
		}

		// Only compensate participants that succeeded
		if participantState.GetState() != finitestate.ParticipantSucceeded {
			continue
		}

		// Execute compensation
		if err := participant.CompensateConfig(ctx, tx); err != nil {
			o.logger.Error("Failed to compensate participant", "name", name, "error", err)
			continue
		}

		// Mark as compensated
		if err := participantState.MarkCompensated(); err != nil {
			o.logger.Error("Failed to mark participant as compensated", "name", name, "error", err)
		}
	}

	// Mark transaction as compensated
	if err := tx.MarkCompensated(); err != nil {
		o.logger.Error("Failed to mark transaction as compensated", "error", err)
	}
}

// getSortedParticipantNames returns a sorted slice of participant names for deterministic ordering.
// This makes components always process in the same order for reproducibility and testing.
func (o *SagaOrchestrator) getSortedParticipantNames() []string {
	var names []string
	for name := range o.runnables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetTransactionStatus returns the current status of a transaction
func (o *SagaOrchestrator) GetTransactionStatus(txID string) (map[string]any, error) {
	// Get transaction from storage
	tx := o.txStorage.GetByID(txID)
	if tx == nil {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}

	// Build status response
	status := map[string]any{
		"id":           tx.ID.String(),
		"state":        tx.GetState(),
		"source":       tx.Source,
		"sourceDetail": tx.SourceDetail,
		"createdAt":    tx.CreatedAt,
		"isValid":      tx.IsValid.Load(),
	}

	// Add participant states if available
	participantStates := tx.GetParticipantStates()
	if len(participantStates) > 0 {
		status["participants"] = participantStates
	}

	return status, nil
}

// waitForRunning waits for the given participant to return to running state
// or until the timeout is reached. Returns an error if the timeout expires.
func (o *SagaOrchestrator) waitForRunning(
	ctx context.Context,
	stateable supervisor.Stateable,
	name string,
) error {
	// Use default values
	reloadTimeout := DefaultReloadTimeout
	retryInterval := DefaultReloadRetryInterval

	// Add timeout to the context
	deadline := time.Now().Add(reloadTimeout)
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	o.logger.Debug("Waiting for participant to be running after reload",
		"name", name, "timeout", reloadTimeout)

	for {
		// Check if participant is running
		if stateable.IsRunning() {
			o.logger.Debug("Participant is now running after reload", "name", name)
			return nil
		}

		// Check if we've reached the deadline
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for participant to be running after reload")
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue waiting
		}
	}
}

// AddToStorage adds a transaction to the transaction storage.
// This implements part of the SagaProcessor interface.
func (o *SagaOrchestrator) AddToStorage(tx *transaction.ConfigTransaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}
	return o.txStorage.Add(tx)
}

// WaitForCompletion waits for the current transaction to reach a terminal state.
// Returns immediately if no transaction is in progress.
func (o *SagaOrchestrator) WaitForCompletion(ctx context.Context) error {
	o.mutex.RLock()
	currentTx := o.txStorage.GetCurrent()
	o.mutex.RUnlock()

	if currentTx == nil {
		o.logger.Debug("No current transaction to wait for")
		return nil
	}

	o.logger.Debug(
		"Waiting for current transaction to complete",
		"txID",
		currentTx.ID,
		"currentState",
		currentTx.GetState(),
	)
	return currentTx.WaitForCompletion(ctx)
}

// TriggerReload initiates a synchronous reload of all participants in the saga.
// This is called after a transaction has successfully completed execution
// (reaches the succeeded state).
//
// The reload is handled by calling CommitConfig() on each participant,
// which is part of the SagaParticipant interface. After each component's
// CommitConfig() call, we wait for it to return to the running state
// before proceeding to the next component.
//
// This method blocks until all components are reloaded and running again, or until
// a timeout occurs. If any component fails to return to running state after reload,
// the transaction will be marked as error, but all components will still have been reloaded.
func (o *SagaOrchestrator) TriggerReload(ctx context.Context) error {
	logger := o.logger.WithGroup("TriggerReload")
	logger.Debug("Triggering reload of all participants")
	o.mutex.RLock()
	participants := o.runnables
	currentTx := o.txStorage.GetCurrent()
	o.mutex.RUnlock()

	if currentTx == nil {
		logger.Debug("No current transaction to reload")
		return nil
	}

	// Mark transaction as reloading
	if err := currentTx.BeginReload(); err != nil {
		logger.Error("Failed to mark transaction as reloading", "error", err)
		return err
	}

	// If no participants are registered, we can complete immediately
	if len(participants) == 0 {
		logger.Debug("No participants registered for reload")
		// Mark as completed since there's nothing to reload
		if err := currentTx.MarkCompleted(); err != nil {
			logger.Error("Failed to mark transaction as completed (no participants)", "error", err)
			return err
		}
		logger.Debug("Transaction completed successfully (no participants)")
		return nil
	}

	// Get sorted participant names for deterministic ordering
	names := o.getSortedParticipantNames()

	logger.Debug("Starting reload of all participants",
		"transactionID", currentTx.ID,
		"participantCount", len(names))

	var reloadErrors []error

	// Apply pending configuration for each participant
	for _, name := range names {
		logger := logger.With("participant", name)
		participant := participants[name]

		// Log pre-reload state if available
		if stateable, ok := participant.(supervisor.Stateable); ok {
			logger.Debug("Pre-reload state", "state", stateable.GetState())
		}

		// Call CommitConfig on the participant
		logger.Debug("Applying pending configuration")
		if err := participant.CommitConfig(ctx); err != nil {
			logger.Error("Failed to apply pending configuration", "error", err)
			reloadErrors = append(
				reloadErrors,
				fmt.Errorf("participant %s failed to apply pending config: %w", name, err),
			)
			// Continue with other participants despite errors
			continue
		}

		// Log post-reload state if available
		if stateable, ok := participant.(supervisor.Stateable); ok {
			logger.Debug("Post-reload state", "state", stateable.GetState())

			// Wait for participant to be running again
			if err := o.waitForRunning(ctx, stateable, name); err != nil {
				logger.Error(
					"Participant failed to return to running state after reload",
					"error", err,
				)
				reloadErrors = append(
					reloadErrors,
					fmt.Errorf("participant %s failed to return to running state: %w", name, err),
				)
				// Continue with other participants despite errors
			}
		}
	}

	// Handle completion or errors
	if len(reloadErrors) > 0 {
		e := errors.Join(reloadErrors...)
		if err := currentTx.MarkError(e); err != nil {
			logger.Error(
				"Failed to mark transaction as error after reload failures",
				"error", err,
			)
		}
		return e
	}

	// If successful, mark as completed
	if err := currentTx.MarkCompleted(); err != nil {
		logger.Error(
			"Failed to mark transaction as completed after successful reload",
			"error", err,
		)
		return err
	}

	logger.Debug("Reload completed",
		"transactionID", currentTx.ID,
		"participantCount", len(names))

	return nil
}
