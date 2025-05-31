package txmgr

import (
	"log/slog"
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
