// Configuration saga state machine implementation.
// Tracks the lifecycle of the overall configuration change process.
package finitestate

import (
	"log/slog"

	"github.com/robbyt/go-fsm"
)

// Saga state constants
const (
	// Initial states
	StateCreated    = "created"    // Initial state when transaction is created
	StateValidating = "validating" // Validation is in progress

	// Validation outcome states
	StateValidated = "validated" // Validation succeeded, ready for execution
	StateInvalid   = "invalid"   // Validation failed (terminal state)

	// Execution states
	StateExecuting = "executing" // Transaction is being executed across components
	StateSucceeded = "succeeded" // Execution succeeded, ready for reload
	StateReloading = "reloading" // Reloading components with new configuration
	StateCompleted = "completed" // Transaction fully completed (terminal state)

	// Failure and compensation states
	StateFailed       = "failed"       // Execution failed, ready for compensation
	StateCompensating = "compensating" // Compensation is in progress
	StateCompensated  = "compensated"  // Compensation completed (terminal state)

	// Error state (for all unrecoverable errors)
	StateError = "error" // Unrecoverable error occurred (terminal state)
)

// SagaTransitions defines the valid state transitions for a configuration saga.
var SagaTransitions = map[string][]string{
	// Initial validation flow
	StateCreated:    {StateValidating, StateError},
	StateValidating: {StateValidated, StateInvalid, StateError},
	StateValidated:  {StateExecuting, StateError},
	StateInvalid:    {}, // Invalid is a terminal state

	// Execution flow
	StateExecuting: {StateSucceeded, StateFailed, StateError},
	StateSucceeded: {StateReloading, StateFailed, StateError},
	StateReloading: {StateCompleted, StateError},
	StateCompleted: {}, // Completed is a terminal state

	// Compensation flow
	StateFailed:       {StateCompensating, StateError},
	StateCompensating: {StateCompensated, StateError},
	StateCompensated:  {}, // Compensated is a terminal state

	// Error state
	StateError: {}, // Error is a terminal state for unrecoverable errors
}

// SagaFactory creates saga state machines
type SagaFactory struct{}

// NewMachine creates a new saga state machine
func (f *SagaFactory) NewMachine(handler slog.Handler) (Machine, error) {
	return fsm.New(handler, StateCreated, SagaTransitions)
}

// NewSagaMachine creates a new saga state machine directly
func NewSagaMachine(handler slog.Handler) (Machine, error) {
	factory := &SagaFactory{}
	return factory.NewMachine(handler)
}
