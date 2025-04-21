package core

import (
	"context"
	"log/slog"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// Option represents a functional option for configuring Runner.
type Option func(*Runner)

// WithLogHandler sets a custom slog handler for the Runner instance.
// For example, to use a custom JSON handler with debug level:
//
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
func WithLogHandler(handler slog.Handler) Option {
	return func(r *Runner) {
		if handler != nil {
			r.logger = slog.New(handler.WithGroup("core.Runner"))
		}
	}
}

// WithLogger sets a logger for the Runner instance.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Runner) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// WithContext sets a custom context for the Runner instance.
// This allows for more granular control over cancellation and timeouts.
func WithContext(ctx context.Context) Option {
	return func(r *Runner) {
		if ctx != nil {
			r.ctx, r.cancel = context.WithCancel(ctx)
		}
	}
}

// WithConfigCallback sets the function that will be called to load or reload configuration.
// This option is required when creating a new Runner.
func WithConfigCallback(callback func() *pb.ServerConfig) Option {
	return func(r *Runner) {
		r.configCallback = callback
	}
}
