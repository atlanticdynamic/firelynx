// Package transaction provides the core configuration transaction framework.
// It implements the Config Saga Pattern for managing configuration changes
// through a complete lifecycle with metadata tracking.
package transaction

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/robbyt/go-loglater"
)

// Source describes the origin of a configuration
type Source string

const (
	// SourceFile indicates configuration sourced from a file
	SourceFile Source = "file"
	// SourceAPI indicates configuration sourced from an API request
	SourceAPI Source = "api"
	// SourceTest indicates configuration sourced from a test
	SourceTest Source = "test"
)

// TransactionID is a unique identifier for a configuration transaction
type TransactionID string

// ConfigTransaction represents a complete lifecycle of a configuration change
type ConfigTransaction struct {
	// ID is the unique identifier for this transaction
	ID TransactionID

	// Source metadata
	Source       Source
	SourceDetail string
	RequestID    string
	CreatedAt    time.Time

	// State management
	StateMachine finitestate.Machine

	// Logging with history tracking
	Logger       *slog.Logger
	LogCollector *loglater.LogCollector

	// Domain configuration
	Config *config.Config

	// Validation state
	ValidationErrors []error
	IsValid          bool
}

// New creates a new ConfigTransaction with the given source information
func New(
	source Source,
	sourceDetail, requestID string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	now := time.Now()
	txID := GenerateTransactionID()

	// Create state machine
	sm, err := finitestate.New(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction state machine: %w", err)
	}

	// Set up logger with history collector
	logCollector := loglater.NewLogCollector(handler)
	logger := slog.New(logCollector)

	tx := &ConfigTransaction{
		ID:               txID,
		Source:           source,
		SourceDetail:     sourceDetail,
		RequestID:        requestID,
		CreatedAt:        now,
		StateMachine:     sm,
		Logger:           logger,
		LogCollector:     logCollector,
		Config:           cfg,
		ValidationErrors: []error{},
		IsValid:          false,
	}

	// Log the transaction creation
	tx.Logger.Info("Transaction created",
		"id", tx.ID,
		"source", tx.Source,
		"sourceDetail", tx.SourceDetail,
		"requestID", tx.RequestID)

	return tx, nil
}

// GenerateTransactionID creates a new unique transaction ID
func GenerateTransactionID() TransactionID {
	// Simple implementation for now, will enhance later
	return TransactionID(time.Now().Format("20060102150405.000000"))
}

// CurrentState returns the current state of the transaction
func (tx *ConfigTransaction) CurrentState() string {
	return tx.StateMachine.GetState()
}

// BeginValidation marks the transaction as being validated
func (tx *ConfigTransaction) BeginValidation() error {
	err := tx.StateMachine.Transition(finitestate.StateValidating)
	if err != nil {
		tx.Logger.Error("Failed to transition to validating state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction validation started",
		"id", tx.ID,
		"state", finitestate.StateValidating)

	return nil
}

// MarkValid marks the transaction as valid after successful validation
func (tx *ConfigTransaction) MarkValid() error {
	err := tx.StateMachine.Transition(finitestate.StateValidated)
	if err != nil {
		tx.Logger.Error("Failed to transition to validated state", "error", err)
		return err
	}

	// Only update state after successful transition
	tx.IsValid = true

	tx.Logger.Info("Transaction validated successfully",
		"id", tx.ID,
		"state", finitestate.StateValidated)

	return nil
}

// MarkInvalid marks the transaction as invalid after failed validation
func (tx *ConfigTransaction) MarkInvalid(errs []error) error {
	err := tx.StateMachine.Transition(finitestate.StateInvalid)
	if err != nil {
		tx.Logger.Error("Failed to transition to invalid state", "error", err)
		return err
	}

	// Only update state after successful transition
	tx.IsValid = false
	tx.ValidationErrors = append(tx.ValidationErrors, errs...)

	tx.Logger.Error("Transaction validation failed",
		"id", tx.ID,
		"state", finitestate.StateInvalid,
		"errorCount", len(tx.ValidationErrors))

	return nil
}

// BeginPreparation marks the transaction as being prepared
func (tx *ConfigTransaction) BeginPreparation() error {
	if !tx.IsValid {
		tx.Logger.Error("Cannot prepare invalid transaction",
			"id", tx.ID,
			"state", tx.CurrentState())
		return ErrInvalidTransaction
	}

	err := tx.StateMachine.Transition(finitestate.StatePreparing)
	if err != nil {
		tx.Logger.Error("Failed to transition to preparing state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction preparation started",
		"id", tx.ID,
		"state", finitestate.StatePreparing)

	return nil
}

// MarkPrepared marks the transaction as prepared and ready for commit
func (tx *ConfigTransaction) MarkPrepared() error {
	err := tx.StateMachine.Transition(finitestate.StatePrepared)
	if err != nil {
		tx.Logger.Error("Failed to transition to prepared state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction prepared successfully",
		"id", tx.ID,
		"state", finitestate.StatePrepared)

	return nil
}

// BeginCommit marks the transaction as being committed
func (tx *ConfigTransaction) BeginCommit() error {
	err := tx.StateMachine.Transition(finitestate.StateCommitting)
	if err != nil {
		tx.Logger.Error("Failed to transition to committing state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction commit started",
		"id", tx.ID,
		"state", finitestate.StateCommitting)

	return nil
}

// MarkCommitted marks the transaction as successfully committed
func (tx *ConfigTransaction) MarkCommitted() error {
	err := tx.StateMachine.Transition(finitestate.StateCommitted)
	if err != nil {
		tx.Logger.Error("Failed to transition to committed state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction committed successfully",
		"id", tx.ID,
		"state", finitestate.StateCommitted)

	return nil
}

// MarkCompleted marks the transaction as fully completed
func (tx *ConfigTransaction) MarkCompleted() error {
	err := tx.StateMachine.Transition(finitestate.StateCompleted)
	if err != nil {
		tx.Logger.Error("Failed to transition to completed state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction completed successfully",
		"id", tx.ID,
		"state", finitestate.StateCompleted,
		"duration", time.Since(tx.CreatedAt))

	return nil
}

// BeginRollback marks the transaction as being rolled back
func (tx *ConfigTransaction) BeginRollback() error {
	err := tx.StateMachine.Transition(finitestate.StateRollingBack)
	if err != nil {
		tx.Logger.Error("Failed to transition to rolling back state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction rollback started",
		"id", tx.ID,
		"state", finitestate.StateRollingBack,
		"fromState", tx.CurrentState())

	return nil
}

// MarkRolledBack marks the transaction as successfully rolled back
func (tx *ConfigTransaction) MarkRolledBack() error {
	err := tx.StateMachine.Transition(finitestate.StateRolledBack)
	if err != nil {
		tx.Logger.Error("Failed to transition to rolled back state", "error", err)
		return err
	}

	tx.Logger.Info("Transaction rolled back successfully",
		"id", tx.ID,
		"state", finitestate.StateRolledBack)

	return nil
}

// MarkFailed marks the transaction as failed
func (tx *ConfigTransaction) MarkFailed(err error) error {
	transErr := tx.StateMachine.Transition(finitestate.StateFailed)
	if transErr != nil {
		tx.Logger.Error("Failed to transition to failed state",
			"error", transErr,
			"originalError", err)
		return transErr
	}

	// Only update state after successful transition
	tx.ValidationErrors = append(tx.ValidationErrors, err)

	tx.Logger.Error("Transaction failed",
		"id", tx.ID,
		"state", finitestate.StateFailed,
		"error", err)

	return nil
}

// GetErrors returns all validation errors for this transaction
func (tx *ConfigTransaction) GetErrors() []error {
	return tx.ValidationErrors
}

// GetConfig returns the configuration associated with this transaction
func (tx *ConfigTransaction) GetConfig() *config.Config {
	return tx.Config
}

// GetLogger returns the logger associated with this transaction
func (tx *ConfigTransaction) GetLogger() *slog.Logger {
	return tx.Logger
}

// PlaybackLogs plays back the transaction logs to the given handler
func (tx *ConfigTransaction) PlaybackLogs(handler slog.Handler) error {
	return tx.LogCollector.PlayLogs(handler)
}

// GetTotalDuration returns the total duration of the transaction so far
func (tx *ConfigTransaction) GetTotalDuration() time.Duration {
	return time.Since(tx.CreatedAt)
}
