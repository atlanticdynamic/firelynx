// Package transaction provides the core configuration transaction framework.
// It implements the Config Saga Pattern for managing configuration changes
// through a complete lifecycle with metadata tracking.
package transaction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/atlanticdynamic/firelynx/internal/logging/writers"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	httpCfg "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/gofrs/uuid/v5"
	"github.com/robbyt/go-loglater"
	"github.com/robbyt/go-loglater/storage"
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
	// Source indicates the general category of configuration source (file, API, test)
	Source Source

	// SourceDetail provides specific information about the origin of the configuration.
	// This field contains more detailed context about where the configuration came from:
	//   - For SourceFile: The absolute file path (e.g., "/etc/firelynx/config.toml")
	//   - For SourceAPI: The API service name (e.g., "gRPC API")
	//   - For SourceTest: The test name (e.g., "TestConfigReload")
	// This information is useful for auditing, debugging, and tracing configuration changes.
	SourceDetail string

	// RequestID contains a correlation ID for API requests or can be empty for file sources
	RequestID string

	// CreatedAt records when this transaction was created
	CreatedAt time.Time

	// State management
	fsm finitestate.Machine

	// Participant tracking
	participants *ParticipantCollection

	// Logging with history tracking
	logger       *slog.Logger
	logCollector *loglater.LogCollector

	// Domain configuration
	domainConfig *config.Config

	// App-related resources
	app struct {
		factory    appFactory
		collection *serverApps.AppInstances
	}

	// Middleware-related resources
	middleware struct {
		factory    middlewareFactory
		collection *httpCfg.MiddlewareCollection
	}

	// Validation state
	IsValid atomic.Bool
}

// New creates a new ConfigTransaction with the given source information.
//
// - source: General category of the configuration origin (file, API, test)
// - sourceDetail: Specific information about the configuration source:
//   - For SourceFile: The absolute file path (e.g., "/etc/firelynx/config.toml")
//   - For SourceAPI: The API service name (e.g., "gRPC API")
//   - For SourceTest: The test name (e.g., "TestConfigReload")
//
// - requestID: Correlation ID for API requests, can be empty for file/test sources
// - cfg: Domain configuration object to be managed by this transaction
// - handler: Logging handler to use for this transaction's logs
func New(
	source Source,
	sourceDetail, requestID string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if handler == nil {
		handler = slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler()
	}

	sm, err := finitestate.NewSagaFSM(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}

	txID := uuid.Must(uuid.NewV6())

	// Set up logger with the loglater history collector
	logCollector := loglater.NewLogCollector(handler)
	logger := slog.New(logCollector).With(
		"id", txID,
		"source", source,
		"sourceDetail", sourceDetail,
		"requestID", requestID)

	// Create participant collection
	participants := NewParticipantCollection(handler)

	tx := &ConfigTransaction{
		ID:           txID,
		Source:       source,
		SourceDetail: sourceDetail,
		RequestID:    requestID,
		CreatedAt:    time.Now(),
		fsm:          sm,
		participants: participants,
		logger:       logger,
		logCollector: logCollector,
		domainConfig: cfg,
		IsValid:      atomic.Bool{},
	}

	// Initialize factories and collections
	tx.app.factory = serverApps.NewAppFactory()
	tx.middleware.factory = httpCfg.NewMiddlewareFactory()

	// Log the transaction creation
	tx.logger.Debug("Transaction created")

	return tx, nil
}

func (tx *ConfigTransaction) String() string {
	return fmt.Sprintf("Transaction ID: %s, State: %s", tx.GetTransactionID(), tx.GetState())
}

func (tx *ConfigTransaction) GetTransactionID() string {
	return tx.ID.String()
}

// GetState returns the current state of the transaction
func (tx *ConfigTransaction) GetState() string {
	return tx.fsm.GetState()
}

// BeginValidation marks the transaction as being validated
func (tx *ConfigTransaction) BeginValidation() error {
	err := tx.fsm.Transition(finitestate.StateValidating)
	if err != nil {
		tx.logger.Error("Failed to transition to validating state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction validation started", "state", finitestate.StateValidating)
	return nil
}

// MarkValidated marks the transaction as validated and ready for execution
func (tx *ConfigTransaction) MarkValidated() error {
	if !tx.IsValid.Load() {
		return tx.MarkInvalid(errors.New("transaction validation failed"))
	}

	err := tx.fsm.Transition(finitestate.StateValidated)
	if err != nil {
		tx.logger.Error("Failed to transition to validated state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction validated successfully", "state", finitestate.StateValidated)
	return nil
}

// MarkInvalid marks the transaction as invalid due to validation errors
func (tx *ConfigTransaction) MarkInvalid(err error) error {
	fErr := tx.fsm.Transition(finitestate.StateInvalid)
	if fErr != nil {
		tx.logger.Error("Failed to transition to invalid state",
			"error", fErr,
			"originalError", err)
		return fErr
	}

	tx.logger.Warn("Transaction validation failed",
		"state", finitestate.StateInvalid,
		"error", err)
	return nil
}

// BeginExecution marks the transaction as being executed
func (tx *ConfigTransaction) BeginExecution() error {
	currentState := tx.GetState()
	if currentState != finitestate.StateValidated {
		tx.logger.Error("Cannot execute non validated transaction", "state", currentState)
		return ErrNotValidated
	}

	err := tx.fsm.Transition(finitestate.StateExecuting)
	if err != nil {
		tx.logger.Error("Failed to transition to executing state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction execution started", "state", finitestate.StateExecuting)
	return nil
}

// MarkSucceeded marks the transaction as successfully executed
func (tx *ConfigTransaction) MarkSucceeded() error {
	err := tx.fsm.Transition(finitestate.StateSucceeded)
	if err != nil {
		tx.logger.Error("Failed to transition to succeeded state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction executed successfully", "state", finitestate.StateSucceeded)
	return nil
}

// MarkCompleted marks the transaction as fully completed
func (tx *ConfigTransaction) MarkCompleted() error {
	err := tx.fsm.Transition(finitestate.StateCompleted)
	if err != nil {
		tx.logger.Error("Failed to transition to completed state", "error", err)
		return err
	}

	tx.logger.Debug(
		"Transaction completed successfully",
		"state", finitestate.StateCompleted,
		"duration", time.Since(tx.CreatedAt),
	)
	return nil
}

// BeginReload marks the transaction as being reloaded
func (tx *ConfigTransaction) BeginReload() error {
	err := tx.fsm.Transition(finitestate.StateReloading)
	if err != nil {
		tx.logger.Error("Failed to transition to reloading state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction reload started", "state", finitestate.StateReloading)
	return nil
}

// BeginCompensation marks the transaction as being compensated (rolled back)
func (tx *ConfigTransaction) BeginCompensation() error {
	// The FSM state transitions should enforce that only Failed state can transition to Compensating
	err := tx.fsm.Transition(finitestate.StateCompensating)
	if err != nil {
		tx.logger.Error("Failed to transition to compensating state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction compensation started", "state", finitestate.StateCompensating)
	return nil
}

// MarkCompensated marks the transaction as successfully compensated (rolled back)
func (tx *ConfigTransaction) MarkCompensated() error {
	err := tx.fsm.Transition(finitestate.StateCompensated)
	if err != nil {
		tx.logger.Error("Failed to transition to compensated state", "error", err)
		return err
	}

	tx.logger.Debug("Transaction compensated successfully", "state", finitestate.StateCompensated)
	return nil
}

// MarkError marks the transaction as in an unrecoverable error state
func (tx *ConfigTransaction) MarkError(err error) error {
	transErr := tx.fsm.Transition(finitestate.StateError)
	if transErr != nil {
		tx.logger.Error("Failed to transition to error state",
			"error", transErr,
			"originalError", err)
		return transErr
	}

	tx.logger.Error("Transaction encountered unrecoverable error",
		"state", finitestate.StateError,
		"error", err)
	return nil
}

// MarkFailed marks the transaction as failed
func (tx *ConfigTransaction) MarkFailed(ctx context.Context, err error) error {
	// Check if context is canceled before attempting state transition
	if ctx.Err() != nil {
		tx.logger.Debug(
			"Context canceled, skipping MarkFailed transition",
			"contextError",
			ctx.Err(),
		)
		return ctx.Err()
	}

	transErr := tx.fsm.Transition(finitestate.StateFailed)
	if transErr != nil {
		// Check if this is an invalid transition error (like from StateError to StateFailed)
		// Since StateError is already a terminal error state, attempting to transition
		// to StateFailed during shutdown is not a real problem
		if errors.Is(transErr, finitestate.ErrInvalidStateTransition) {
			tx.logger.Warn(
				"Invalid FSM transition for MarkFailed, transaction likely already in terminal state",
				"currentState",
				tx.fsm.GetState(),
				"transitionError",
				transErr,
				"originalError",
				err,
			)
			return nil
		}

		tx.logger.Error("Failed to transition to failed state",
			"error", transErr,
			"originalError", err)
		return transErr
	}

	// Only update state after successful transition
	tx.logger.Warn("Transaction failed", "state", finitestate.StateFailed, "error", err)
	return nil
}

// GetConfig returns the configuration associated with this transaction
func (tx *ConfigTransaction) GetConfig() *config.Config {
	return tx.domainConfig
}

// GetAppCollection returns the app collection associated with this transaction
func (tx *ConfigTransaction) GetAppCollection() *serverApps.AppInstances {
	return tx.app.collection
}

// PlaybackLogs plays back the transaction logs to the given handler
func (tx *ConfigTransaction) PlaybackLogs(handler slog.Handler) error {
	return tx.logCollector.PlayLogs(handler)
}

// GetLogs returns the raw log records from the transaction's log collector
func (tx *ConfigTransaction) GetLogs() []storage.Record {
	return tx.logCollector.GetLogs()
}

// GetTotalDuration returns the total duration of the transaction so far
func (tx *ConfigTransaction) GetTotalDuration() time.Duration {
	return time.Since(tx.CreatedAt)
}

// RegisterParticipant registers a new participant in this transaction
func (tx *ConfigTransaction) RegisterParticipant(name string) error {
	return tx.participants.AddParticipant(name)
}

// GetParticipants returns the participant collection for this transaction
func (tx *ConfigTransaction) GetParticipants() *ParticipantCollection {
	return tx.participants
}

// GetParticipantStates returns a map of participant names to their current states
func (tx *ConfigTransaction) GetParticipantStates() map[string]string {
	return tx.participants.GetParticipantStates()
}

// GetParticipantErrors returns a map of participant names to their errors
func (tx *ConfigTransaction) GetParticipantErrors() map[string]error {
	return tx.participants.GetParticipantErrors()
}

// WaitForCompletion waits for the transaction to reach a terminal state.
// Terminal states are: Completed, Compensated, Error, and Invalid.
// Returns immediately if already in a terminal state.
func (tx *ConfigTransaction) WaitForCompletion(ctx context.Context) error {
	// Check if already in a terminal state
	currentState := tx.GetState()
	if tx.isTerminalState(currentState) {
		tx.logger.Debug("Transaction already in terminal state",
			"currentState", currentState)
		return nil
	}

	tx.logger.Debug("Transaction not in terminal state, waiting",
		"currentState", currentState)

	// Get the FSM state channel
	stateChan := tx.fsm.GetStateChan(ctx)

	// Wait for terminal state
	for {
		select {
		case <-ctx.Done():
			tx.logger.Debug(
				"WaitForCompletion context cancelled",
				"finalState",
				tx.GetState(),
				"error",
				ctx.Err(),
			)
			return ctx.Err()
		case state, ok := <-stateChan:
			if !ok {
				return nil
			}
			if tx.isTerminalState(state) {
				return nil
			}
		}
	}
}

// isTerminalState returns true if the given state is a terminal state.
func (tx *ConfigTransaction) isTerminalState(state string) bool {
	return slices.Contains(finitestate.SagaTerminalStates, state)
}

// GetMiddlewareRegistry returns the middleware registry for use by adapters
func (tx *ConfigTransaction) GetMiddlewareRegistry() httpCfg.MiddlewareRegistry {
	if tx.middleware.collection == nil {
		return make(httpCfg.MiddlewareRegistry)
	}
	return tx.middleware.collection.GetRegistry()
}

// validateResourceConflicts validates that middleware instances don't conflict on shared resources
func validateResourceConflicts(allMiddlewares middleware.MiddlewareCollection) error {
	return validateConsoleLoggerFileConflicts(allMiddlewares)
}

// validateConsoleLoggerFileConflicts checks that console loggers don't use the same output file
func validateConsoleLoggerFileConflicts(allMiddlewares middleware.MiddlewareCollection) error {
	var consoleLoggers []*configLogger.ConsoleLogger

	// Extract all console loggers from the middleware collection
	for _, mw := range allMiddlewares {
		if consoleLogger, ok := mw.Config.(*configLogger.ConsoleLogger); ok {
			consoleLoggers = append(consoleLoggers, consoleLogger)
		}
	}

	if len(consoleLoggers) <= 1 {
		return nil
	}

	fileUsage := make(map[string]int)
	var errs []error

	for _, logger := range consoleLoggers {
		expandedOutput, err := expandMiddlewareOutput(logger.Output)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"failed to expand environment variables in logger output '%s': %w",
				logger.Output,
				err,
			))
			continue
		}

		// Only track file paths, not stdout/stderr
		writerType := writers.ParseWriterType(expandedOutput)
		if writerType == writers.WriterTypeFile {
			fileUsage[expandedOutput]++
		}
	}
	for filePath, count := range fileUsage {
		if count > 1 {
			errs = append(errs, fmt.Errorf(
				"%w: duplicate output file '%s' used by %d console logger instances",
				ErrResourceConflict,
				filePath,
				count,
			))
		}
	}

	return errors.Join(errs...)
}

// expandMiddlewareOutput expands environment variables in middleware output paths
func expandMiddlewareOutput(output string) (string, error) {
	return interpolation.ExpandEnvVars(output)
}
