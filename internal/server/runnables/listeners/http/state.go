package http

import "context"

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.cluster.GetState()
}

// IsReady returns whether the runner is running
func (r *Runner) IsReady() bool {
	return r.cluster.IsReady()
}

// GetStateChan returns a channel that emits state changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.cluster.GetStateChan(ctx)
}
