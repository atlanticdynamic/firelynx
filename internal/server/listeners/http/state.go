package http

import "context"

// IsRunning returns true if the runner is in the Running state
func (r *Runner) IsRunning() bool {
	return r.runner.IsRunning()
}

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.runner.GetState()
}

// GetStateChan returns a channel that emits the runner's state whenever it changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.runner.GetStateChan(ctx)
}
