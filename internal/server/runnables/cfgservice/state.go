package cfgservice

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
)

// IsRunning returns true if the runner is in the Running state
func (r *Runner) IsRunning() bool {
	return r.fsm.GetState() == finitestate.StatusRunning
}

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.fsm.GetState()
}

// GetStateChan returns a channel that emits the runner's state whenever it changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.fsm.GetStateChan(ctx)
}
