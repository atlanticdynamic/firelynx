// Package finitestate provides finite state machines to track the
// lifecycle of configuration sagas and their participants.
package finitestate

import (
	"context"
	"log/slog"
)

// Machine defines the interface for any finite state machine in the system.
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

// Factory defines interface for creating state machines
type Factory interface {
	// NewMachine creates a new state machine instance
	NewMachine(handler slog.Handler) (Machine, error)
}
