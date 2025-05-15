// Package transaction provides the core configuration transaction framework.
// It implements the Config Saga Pattern for managing configuration changes
// through a complete lifecycle with metadata tracking.
package transaction

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/gofrs/uuid/v5"
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

// ConfigTransaction represents a complete lifecycle of a configuration change
type ConfigTransaction struct {
	// ID is the unique identifier for this transaction
	ID uuid.UUID

	// Source metadata
	Source       Source
	SourceDetail string
	RequestID    string
	CreatedAt    time.Time

	// State management
	fsm finitestate.Machine

	// Logging with history tracking
	logger       *slog.Logger
	logCollector *loglater.LogCollector

	// Domain configuration
	domainConfig *config.Config

	// Validation state
	validationErrors []error
	IsValid          atomic.Bool
}

// New creates a new ConfigTransaction with the given source information
func New(
	source Source,
	sourceDetail, requestID string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	txID := uuid.Must(uuid.NewV6())

	// Create state machine for this transaction
	sm, err := finitestate.New(handler)
	if err != nil {
		return nil, fmt.Errorf("%s failed to create state machine: %w", txID, err)
	}

	// Set up logger with the loglater history collector and additional metadata
	logCollector := loglater.NewLogCollector(handler)
	logger := slog.New(logCollector).With(
		"id", txID,
		"source", source,
		"sourceDetail", sourceDetail,
		"requestID", requestID)

	tx := &ConfigTransaction{
		ID:               txID,
		Source:           source,
		SourceDetail:     sourceDetail,
		RequestID:        requestID,
		CreatedAt:        time.Now(),
		fsm:              sm,
		logger:           logger,
		logCollector:     logCollector,
		domainConfig:     cfg,
		validationErrors: []error{},
		IsValid:          atomic.Bool{},
	}

	// Log the transaction creation
	tx.logger.Info("Transaction created")

	return tx, nil
}

// GetState returns the current state of the transaction
func (tx *ConfigTransaction) GetState() string {
	return tx.fsm.GetState()
}

// BeginPreparation marks the transaction as being prepared
func (tx *ConfigTransaction) BeginPreparation() error {
	if !tx.IsValid.Load() {
		tx.logger.Error("Cannot prepare invalid transaction", "state", tx.GetState())
		return ErrInvalidTransaction
	}

	err := tx.fsm.Transition(finitestate.StatePreparing)
	if err != nil {
		tx.logger.Error("Failed to transition to preparing state", "error", err)
		return err
	}

	tx.logger.Info("Transaction preparation started", "state", finitestate.StatePreparing)
	return nil
}

// MarkPrepared marks the transaction as prepared and ready for commit
func (tx *ConfigTransaction) MarkPrepared() error {
	err := tx.fsm.Transition(finitestate.StatePrepared)
	if err != nil {
		tx.logger.Error("Failed to transition to prepared state", "error", err)
		return err
	}

	tx.logger.Info("Transaction prepared successfully", "state", finitestate.StatePrepared)
	return nil
}

// BeginCommit marks the transaction as being committed
func (tx *ConfigTransaction) BeginCommit() error {
	err := tx.fsm.Transition(finitestate.StateCommitting)
	if err != nil {
		tx.logger.Error("Failed to transition to committing state", "error", err)
		return err
	}

	tx.logger.Info("Transaction commit started", "state", finitestate.StateCommitting)
	return nil
}

// MarkCommitted marks the transaction as successfully committed
func (tx *ConfigTransaction) MarkCommitted() error {
	err := tx.fsm.Transition(finitestate.StateCommitted)
	if err != nil {
		tx.logger.Error("Failed to transition to committed state", "error", err)
		return err
	}

	tx.logger.Info("Transaction committed successfully", "state", finitestate.StateCommitted)
	return nil
}

// MarkCompleted marks the transaction as fully completed
func (tx *ConfigTransaction) MarkCompleted() error {
	err := tx.fsm.Transition(finitestate.StateCompleted)
	if err != nil {
		tx.logger.Error("Failed to transition to completed state", "error", err)
		return err
	}

	tx.logger.Info(
		"Transaction completed successfully",
		"state",
		finitestate.StateCompleted,
		"duration",
		time.Since(tx.CreatedAt),
	)
	return nil
}

// BeginRollback marks the transaction as being rolled back
func (tx *ConfigTransaction) BeginRollback() error {
	err := tx.fsm.Transition(finitestate.StateRollingBack)
	if err != nil {
		tx.logger.Error("Failed to transition to rolling back state", "error", err)
		return err
	}

	tx.logger.Info(
		"Transaction rollback started",
		"state",
		finitestate.StateRollingBack,
		"fromState",
		tx.GetState(),
	)
	return nil
}

// MarkRolledBack marks the transaction as successfully rolled back
func (tx *ConfigTransaction) MarkRolledBack() error {
	err := tx.fsm.Transition(finitestate.StateRolledBack)
	if err != nil {
		tx.logger.Error("Failed to transition to rolled back state", "error", err)
		return err
	}

	tx.logger.Info("Transaction rolled back successfully", "state", finitestate.StateRolledBack)
	return nil
}

// MarkFailed marks the transaction as failed
func (tx *ConfigTransaction) MarkFailed(err error) error {
	transErr := tx.fsm.Transition(finitestate.StateFailed)
	if transErr != nil {
		tx.logger.Error("Failed to transition to failed state",
			"error", transErr,
			"originalError", err)
		return transErr
	}

	// Only update state after successful transition
	tx.validationErrors = append(tx.validationErrors, err)

	tx.logger.Error("Transaction failed", "state", finitestate.StateFailed, "error", err)
	return nil
}

// GetErrors returns all validation errors for this transaction
func (tx *ConfigTransaction) GetErrors() []error {
	return tx.validationErrors
}

// GetConfig returns the configuration associated with this transaction
func (tx *ConfigTransaction) GetConfig() *config.Config {
	return tx.domainConfig
}

// PlaybackLogs plays back the transaction logs to the given handler
func (tx *ConfigTransaction) PlaybackLogs(handler slog.Handler) error {
	return tx.logCollector.PlayLogs(handler)
}

// GetTotalDuration returns the total duration of the transaction so far
func (tx *ConfigTransaction) GetTotalDuration() time.Duration {
	return time.Since(tx.CreatedAt)
}
