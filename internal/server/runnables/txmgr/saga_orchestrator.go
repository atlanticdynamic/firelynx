package txmgr

import (
	"context"
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
	DefaultReloadRetryInterval = 100 * time.Millisecond
)

// SagaParticipant defines the interface for components participating
// in configuration transactions. It extends Runnable and Stateable from supervisor
// to ensure components have the necessary lifecycle management capabilities.
// Note that SagaParticipant SHOULD NOT implement supervisor.Reloadable to avoid
// conflicts with the ApplyPendingConfig method.
type SagaParticipant interface {
	supervisor.Runnable
	supervisor.Stateable

	// ExecuteConfig processes a validated configuration transaction
	// by preparing the component to apply the changes. This is called
	// during the execution phase of the saga.
	ExecuteConfig(ctx context.Context, tx *transaction.ConfigTransaction) error

	// CompensateConfig reverts changes made during ExecuteConfig
	// when a transaction fails. This is called for successful
	// participants when the saga needs to be rolled back.
	CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error

	// ApplyPendingConfig applies the pending configuration prepared during ExecuteConfig.
	// This is called during the reload phase after all participants have successfully
	// executed their configurations.
	ApplyPendingConfig(ctx context.Context) error
}

// SagaOrchestrator coordinates configuration changes across multiple components
// using the saga pattern. It maintains participant state tracking and handles
// compensation if any component fails.
type SagaOrchestrator struct {
	// Transaction storage for persistent transaction state
	txStorage *txstorage.TransactionStorage

	// Participant collection for tracking component states
	participants *transaction.ParticipantCollection

	// Registry of saga participants
	runnables map[string]SagaParticipant

	// Logger
	logger *slog.Logger

	// Internal state
	mutex sync.RWMutex
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(
	txStorage *txstorage.TransactionStorage,
	handler slog.Handler,
) *SagaOrchestrator {
	logger := slog.New(handler).WithGroup("sagaOrchestrator")
	participants := transaction.NewParticipantCollection(handler)

	return &SagaOrchestrator{
		txStorage:    txStorage,
		participants: participants,
		runnables:    make(map[string]SagaParticipant),
		logger:       logger,
	}
}

// RegisterParticipant registers a component as a saga participant.
// Returns an error if the participant also implements supervisor.Reloadable which would
// cause conflicts with ApplyPendingConfig.
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

	// Track execution status of participants
	o.mutex.RLock()
	participants := o.runnables
	o.mutex.RUnlock()

	// Get sorted participant names for deterministic ordering
	names := o.getSortedParticipantNames()

	// Process each participant
	for _, name := range names {
		participant := participants[name]
		// Get or create participant state tracker
		participantState, err := o.participants.GetOrCreate(name)
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
		err = participant.ExecuteConfig(ctx, tx)
		if err != nil {
			// Mark participant as failed
			if markErr := participantState.MarkFailed(err); markErr != nil {
				o.logger.Error("Failed to mark participant as failed",
					"name", name, "error", markErr, "originalError", err)
			}
			// Mark transaction as failed
			if markErr := tx.MarkFailed(err); markErr != nil {
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
	if o.participants.AllParticipantsSucceeded() {
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

		o.logger.Info("Transaction and reload completed successfully", "id", tx.ID)
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
	if err := o.participants.BeginCompensation(); err != nil {
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
		participantState, err := o.participants.GetOrCreate(name)
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
// This helps ensure that components are always processed in the same order, which is important
// for reproducibility and testing.
func (o *SagaOrchestrator) getSortedParticipantNames() []string {
	var names []string
	for name := range o.runnables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetTransactionStatus returns the current status of a transaction
func (o *SagaOrchestrator) GetTransactionStatus(txID string) (map[string]interface{}, error) {
	// Get transaction from storage
	tx := o.txStorage.GetByID(txID)
	if tx == nil {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}

	// Build status response
	status := map[string]interface{}{
		"id":           tx.ID.String(),
		"state":        tx.GetState(),
		"source":       tx.Source,
		"sourceDetail": tx.SourceDetail,
		"createdAt":    tx.CreatedAt,
		"isValid":      tx.IsValid.Load(),
	}

	// Add participant states if available
	participantStates := o.participants.GetParticipantStates()
	if len(participantStates) > 0 {
		status["participants"] = participantStates
	}

	return status, nil
}
