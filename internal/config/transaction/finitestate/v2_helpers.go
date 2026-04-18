package finitestate

import (
	"context"
	"log/slog"

	"github.com/robbyt/go-fsm/v2"
	"github.com/robbyt/go-fsm/v2/hooks"
	"github.com/robbyt/go-fsm/v2/hooks/broadcast"
	"github.com/robbyt/go-fsm/v2/transitions"
)

func newMachine(handler slog.Handler, initialState string, allowedTransitions map[string][]string) (*fsm.Machine, *broadcast.Manager, error) {
	if handler == nil {
		handler = slog.Default().Handler()
	}

	trans, err := transitions.New(allowedTransitions)
	if err != nil {
		return nil, nil, err
	}

	stateManager := broadcast.NewManager(handler)
	registry, err := hooks.NewRegistry(
		hooks.WithLogHandler(handler),
		hooks.WithTransitions(trans),
	)
	if err != nil {
		return nil, nil, err
	}

	err = registry.RegisterPostTransitionHook(hooks.PostTransitionHookConfig{
		Name:   "transaction.finitestate.broadcast",
		From:   []string{hooks.WildcardStatePattern},
		To:     []string{hooks.WildcardStatePattern},
		Action: stateManager.BroadcastHook,
	})
	if err != nil {
		return nil, nil, err
	}

	machine, err := fsm.New(
		initialState,
		trans,
		fsm.WithLogHandler(handler),
		fsm.WithCallbackRegistry(registry),
	)
	if err != nil {
		return nil, nil, err
	}

	return machine, stateManager, nil
}

func getStateChan(ctx context.Context, stateManager *broadcast.Manager, initialState string, opts ...broadcast.Option) <-chan string {
	if ctx == nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	in, err := stateManager.GetStateChan(ctx, opts...)
	if err != nil {
		ch := make(chan string)
		close(ch)
		return ch
	}

	out := make(chan string, 1)
	out <- initialState

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
