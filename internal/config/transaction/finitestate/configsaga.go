// Configuration saga state machine implementation.
// Tracks the lifecycle of the overall configuration change process.
package finitestate

import (
	"context"
	"log/slog"
	"time"

	"github.com/robbyt/go-fsm"
)

// Error aliases from go-fsm for use in transaction handling
var (
	ErrInvalidStateTransition = fsm.ErrInvalidStateTransition
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

type SagaFSM struct {
	*fsm.Machine
}

func (s *SagaFSM) GetStateChan(ctx context.Context) <-chan string {
	return s.GetStateChanWithOptions(ctx, fsm.WithSyncTimeout(5*time.Second))
}

func NewSagaFSM(handler slog.Handler) (*SagaFSM, error) {
	machine, err := fsm.New(handler, StateCreated, SagaTransitions)
	if err != nil {
		return nil, err
	}
	return &SagaFSM{Machine: machine}, nil
}
