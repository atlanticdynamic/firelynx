package finitestate

import (
	"context"
	"log/slog"
	"time"

	"github.com/robbyt/go-fsm/v2"
	"github.com/robbyt/go-fsm/v2/hooks"
	"github.com/robbyt/go-fsm/v2/transitions"
)

const (
	StatusNew       = transitions.StatusNew
	StatusBooting   = transitions.StatusBooting
	StatusRunning   = transitions.StatusRunning
	StatusReloading = transitions.StatusReloading
	StatusStopping  = transitions.StatusStopping
	StatusStopped   = transitions.StatusStopped
	StatusError     = transitions.StatusError
	StatusUnknown   = transitions.StatusUnknown
)

var typicalTransitions = transitions.Typical

const defaultBroadcastTimeout = 5 * time.Second

var ErrInvalidStateTransition = fsm.ErrInvalidStateTransition

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
}

type machine struct {
	fsm *fsm.Machine
}

// GetStateChan returns a channel that emits state updates and closes when the context is canceled.
func (m *machine) GetStateChan(ctx context.Context) <-chan string {
	if ctx == nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	in := make(chan string, 1)
	err := m.fsm.GetStateChan(ctx, in)
	if err != nil {
		slog.Error("failed to register finitestate state channel", "error", err)
		ch := make(chan string)
		close(ch)
		return ch
	}

	out := make(chan string, 1)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case state, ok := <-in:
				if !ok {
					return
				}

				select {
				case out <- state:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}

func (m *machine) Transition(state string) error {
	return m.fsm.Transition(state)
}

func (m *machine) TransitionBool(state string) bool {
	return m.fsm.TransitionBool(state)
}

func (m *machine) TransitionIfCurrentState(currentState, newState string) error {
	return m.fsm.TransitionIfCurrentState(currentState, newState)
}

func (m *machine) SetState(state string) error {
	return m.fsm.SetState(state)
}

func (m *machine) GetState() string {
	return m.fsm.GetState()
}

// New creates a new finite state machine with the specified logger using "standard" state transitions.
func New(handler slog.Handler) (Machine, error) {
	return newWithConfig(handler, StatusNew, typicalTransitions)
}

// NewWithTransitions creates a new finite state machine with a custom transition table.
func NewWithTransitions(
	handler slog.Handler,
	initialState string,
	allowedTransitions map[string][]string,
) (Machine, error) {
	trans, err := transitions.New(allowedTransitions)
	if err != nil {
		return nil, err
	}

	return newWithConfig(handler, initialState, trans)
}

func newWithConfig(
	handler slog.Handler,
	initialState string,
	trans *transitions.Config,
) (Machine, error) {
	if handler == nil {
		handler = slog.Default().Handler()
	}

	registry, err := hooks.NewRegistry(
		hooks.WithLogHandler(handler),
		hooks.WithTransitions(trans),
	)
	if err != nil {
		return nil, err
	}

	f, err := fsm.New(
		initialState,
		trans,
		fsm.WithLogHandler(handler),
		fsm.WithCallbackRegistry(registry),
		fsm.WithBroadcastTimeout(defaultBroadcastTimeout),
	)
	if err != nil {
		return nil, err
	}

	return &machine{fsm: f}, nil
}
