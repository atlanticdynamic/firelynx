package finitestate

import (
	"context"
	"log/slog"
	"time"

	"github.com/robbyt/go-fsm/v2"
	"github.com/robbyt/go-fsm/v2/hooks"
	"github.com/robbyt/go-fsm/v2/hooks/broadcast"
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

// TypicalTransitions is a set of standard transitions for a finite state machine.
var TypicalTransitions = transitions.Typical

// SubscriberOption is a functional option for configuring state channel behavior
type SubscriberOption = broadcast.Option

// WithSyncTimeout sets a timeout for synchronous broadcast operations
var WithSyncTimeout = broadcast.WithTimeout

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
	stateManager *broadcast.Manager
}

// GetStateChan returns a sync broadcast channel with 5-second timeout to ensure state updates are delivered during shutdown
func (m *ServerFSM) GetStateChan(ctx context.Context) <-chan string {
	return m.GetStateChanWithOptions(ctx, WithSyncTimeout(5*time.Second))
}

// GetStateChanWithOptions returns a channel with custom broadcast options.
func (m *ServerFSM) GetStateChanWithOptions(ctx context.Context, opts ...SubscriberOption) <-chan string {
	if ctx == nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	in, err := m.stateManager.GetStateChan(ctx, opts...)
	if err != nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	out := make(chan string, 1)
	out <- m.GetState()

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

// New creates a new finite state machine with the specified logger using "standard" state transitions.
func New(handler slog.Handler) (Machine, error) {
	if handler == nil {
		handler = slog.Default().Handler()
	}

	stateManager := broadcast.NewManager(handler)
	registry, err := hooks.NewRegistry(
		hooks.WithLogHandler(handler),
		hooks.WithTransitions(TypicalTransitions),
	)
	if err != nil {
		return nil, err
	}

	err = registry.RegisterPostTransitionHook(hooks.PostTransitionHookConfig{
		Name:   "finitestate.broadcast",
		From:   []string{hooks.WildcardStatePattern},
		To:     []string{hooks.WildcardStatePattern},
		Action: stateManager.BroadcastHook,
	})
	if err != nil {
		return nil, err
	}

	machine, err := fsm.New(
		StatusNew,
		TypicalTransitions,
		fsm.WithLogHandler(handler),
		fsm.WithCallbackRegistry(registry),
	)
	if err != nil {
		return nil, err
	}

	return &ServerFSM{Machine: machine, stateManager: stateManager}, nil
}
