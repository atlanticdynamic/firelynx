package finitestate

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-fsm/v2"
	"github.com/robbyt/go-fsm/v2/hooks"
	"github.com/robbyt/go-fsm/v2/transitions"
)

func newMachine(handler slog.Handler, initialState string, allowedTransitions map[string][]string) (*fsm.Machine, error) {
	if handler == nil {
		handler = slog.Default().Handler()
	}

	trans, err := transitions.New(allowedTransitions)
	if err != nil {
		return nil, err
	}

	registry, err := hooks.NewRegistry(
		hooks.WithLogHandler(handler),
		hooks.WithTransitions(trans),
	)
	if err != nil {
		return nil, err
	}

	machine, err := fsm.New(
		initialState,
		trans,
		fsm.WithLogHandler(handler),
		fsm.WithCallbackRegistry(registry),
		fsm.WithBroadcastTimeout(defaultBroadcastTimeout),
	)
	if err != nil {
		return nil, err
	}

	return machine, nil
}

func getStateChan(ctx context.Context, machine *fsm.Machine) <-chan string {
	if ctx == nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	in := make(chan string, 1)
	err := machine.GetStateChan(ctx, in)
	if err != nil {
		slog.Error("failed to register transaction finitestate state channel", "error", err)
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
