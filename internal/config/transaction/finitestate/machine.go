// Package finitestate provides a finite state machine to track the
// configuration transaction lifecycle.
//
// Configuration Transaction Lifecycle:
//  1. Created - Configuration transaction is initialized
//  2. Validating - Validation is in progress
//  3. Validated - Successfully passed validation
//  4. Invalid - Failed validation
//  5. Preparing - Components are preparing for the configuration change
//  6. Prepared - All components are ready for the configuration change
//  7. Committing - Configuration is being committed
//  8. Committed - Configuration has been successfully committed
//  9. Completed - Transaction is fully completed (terminal state)
//
// Error and rollback states:
//   - Rolling Back - Transaction is being rolled back
//   - Rolled Back - Transaction has been successfully rolled back (terminal state)
//   - Failed - Processing error occurred (terminal state)
package finitestate

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-fsm"
)

// Transaction state constants
const (
	StateCreated     = "created"
	StateValidating  = "validating"
	StateValidated   = "validated"
	StateInvalid     = "invalid"
	StatePreparing   = "preparing"
	StatePrepared    = "prepared"
	StateCommitting  = "committing"
	StateCommitted   = "committed"
	StateCompleted   = "completed"
	StateRollingBack = "rolling_back"
	StateRolledBack  = "rolled_back"
	StateFailed      = "failed"
)

// TransactionTransitions defines the valid state transitions for a configuration transaction.
var TransactionTransitions = map[string][]string{
	StateCreated:     {StateValidating, StateFailed},
	StateValidating:  {StateValidated, StateInvalid, StateFailed},
	StateValidated:   {StatePreparing, StateFailed},
	StateInvalid:     {},
	StatePreparing:   {StatePrepared, StateRollingBack, StateFailed},
	StatePrepared:    {StateCommitting, StateRollingBack, StateFailed},
	StateCommitting:  {StateCommitted, StateRollingBack, StateFailed},
	StateCommitted:   {StateCompleted, StateRollingBack, StateFailed},
	StateCompleted:   {},
	StateRollingBack: {StateRolledBack, StateFailed},
	StateRolledBack:  {},
	StateFailed:      {},
}

// Machine defines the interface for the finite state machine that tracks
// the configuration transaction lifecycle.
type Machine interface {
	// Transition attempts to transition the state machine to the specified state.
	Transition(state string) error

	// TransitionBool attempts to transition the state machine to the specified state.
	TransitionBool(state string) bool

	// TransitionIfCurrentState attempts to transition the state machine to the specified state
	TransitionIfCurrentState(currentState, newState string) error

	// SetState sets the state of the state machine to the specified state.
	SetState(state string) error

	// GetState returns the current state of the state machine.
	GetState() string

	// GetStateChan returns a channel that emits the state machine's state whenever it changes.
	// The channel is closed when the provided context is canceled.
	GetStateChan(ctx context.Context) <-chan string
}

// New creates a new finite state machine with the specified logger using transaction state transitions.
func New(handler slog.Handler) (Machine, error) {
	return fsm.New(handler, StateCreated, TransactionTransitions)
}
