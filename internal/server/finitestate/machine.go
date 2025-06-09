package finitestate

import (
	"context"
	"log/slog"
	"time"

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

// WithSyncTimeout sets a timeout for synchronous broadcast operations
var WithSyncTimeout = fsm.WithSyncTimeout

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

// ServerFSM embeds fsm.Machine and overrides GetStateChan for sync broadcast
type ServerFSM struct {
	*fsm.Machine
}

// GetStateChan returns a sync broadcast channel with 5-second timeout to ensure state updates are delivered during shutdown
func (m *ServerFSM) GetStateChan(ctx context.Context) <-chan string {
	return m.GetStateChanWithOptions(ctx, WithSyncTimeout(5*time.Second))
}

// New creates a new finite state machine with the specified logger using "standard" state transitions.
func New(handler slog.Handler) (Machine, error) {
	machine, err := fsm.New(handler, StatusNew, TypicalTransitions)
	if err != nil {
		return nil, err
	}
	return &ServerFSM{Machine: machine}, nil
}
