package http

import "context"

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.cluster.GetState()
}

// IsRunning returns whether the runner is running
func (r *Runner) IsRunning() bool {
	return r.cluster.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.cluster.GetStateChan(ctx)
}
