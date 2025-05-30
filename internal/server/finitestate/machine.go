package finitestate

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-fsm"
)

const (
	StatusNew       = fsm.StatusNew
	StatusBooting   = fsm.StatusBooting
	StatusRunning   = fsm.StatusRunning
	StatusReloading = fsm.StatusReloading
	StatusStopping  = fsm.StatusStopping
	StatusStopped   = fsm.StatusStopped
	StatusError     = fsm.StatusError
	StatusUnknown   = fsm.StatusUnknown
)

// TypicalTransitions is a set of standard transitions for a finite state machine.
var TypicalTransitions = fsm.TypicalTransitions

// SubscriberOption is a functional option for configuring state channel behavior
type SubscriberOption = fsm.SubscriberOption

// WithSyncBroadcast is a channel option that blocks until message delivery instead of dropping on full channels
var WithSyncBroadcast = fsm.WithSyncBroadcast

// Machine defines the interface for the finite state machine that tracks
// the HTTP server's lifecycle states. This abstraction allows for different
// FSM implementations and simplifies testing.
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

	// GetStateChanWithOptions returns a channel with custom configuration options.
	// The channel is closed when the provided context is canceled.
	GetStateChanWithOptions(ctx context.Context, opts ...SubscriberOption) <-chan string
}

// New creates a new finite state machine with the specified logger using "standard" state transitions.
func New(handler slog.Handler) (*fsm.Machine, error) {
	return fsm.New(handler, StatusNew, TypicalTransitions)
}
