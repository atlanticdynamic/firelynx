package wrapper

import "context"

// IsRunning implements the supervisor.Readiness interface
func (s *HttpServer) IsRunning() bool {
	return s.runner.IsRunning()
}

// GetState implements the supervisor.Stateable interface
func (s *HttpServer) GetState() string {
	return s.runner.GetState()
}

// GetStateChan implements the supervisor.Stateable interface
func (s *HttpServer) GetStateChan(ctx context.Context) <-chan string {
	return s.runner.GetStateChan(ctx)
}
