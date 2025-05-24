package cfgfileloader

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
)

var _ supervisor.Stateable = (*Runner)(nil)

func (r *Runner) GetState() string {
	return r.fsm.GetState()
}

func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.fsm.GetStateChan(ctx)
}

func (r *Runner) IsRunning() bool {
	return r.fsm.GetState() == finitestate.StatusRunning
}
