package txmgr

import (
	"log/slog"
	"time"
)

// Option represents a functional option for configuring Runner.
type Option func(*Runner) error

// WithLogHandler sets a custom slog handler for the Runner instance.
// For example, to use a custom JSON handler with debug level:
//
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
func WithLogHandler(handler slog.Handler) Option {
	return func(r *Runner) error {
		if handler != nil {
			r.logger = slog.New(handler).WithGroup("txmgr.Runner")
		}
		return nil
	}
}

// WithLogger sets a logger for the Runner instance.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Runner) error {
		if logger != nil {
			r.logger = logger
		}
		return nil
	}
}

// WithSagaOrchestratorShutdownTimeout sets the timeout for shutting down the saga orchestrator.
func WithSagaOrchestratorShutdownTimeout(timeout time.Duration) Option {
	return func(r *Runner) error {
		if timeout <= 0 {
			return nil // No-op if timeout is not positive
		}
		r.sagaOrchestratorShutdownTimeout = timeout
		return nil
	}
}
