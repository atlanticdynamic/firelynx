// Package finitestate provides a finite state machine to track the
// configuration transaction lifecycle.
//
// Configuration Lifecycle:
//  1. Received - Raw configuration received (from file, gRPC call, CLI, etc.)
//  2. Parsed - Successfully parsed into domain model objects
//  3. Validated - All validation rules passed
//  4. Adapted - Component-specific adapter views generated for consumption by runtime components
//     (This state allows multiple adapter creations for different components while
//     maintaining the same logical transaction state)
//  5. Activated - Configuration applied to running components
//
// Error states:
// - Rejected - Failed validation or semantic rules
// - Error - Processing error occurred
package finitestate

import (
	"context"
	"log/slog"
)

// State constants for the transaction lifecycle
const (
	StateReceived  = "Received"
	StateParsed    = "Parsed"
	StateValidated = "Validated"
	StateAdapted   = "Adapted"
	StateActivated = "Activated"
	StateRejected  = "Rejected"
	StateError     = "Error"
)

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

// New creates a new finite state machine with the specified logger
func New(handler slog.Handler) (Machine, error) {
	// This is just a skeleton for now
	// Will be implemented when we start the actual transaction implementation
	return nil, nil
}
