package http

import (
	"context"
	"log/slog"
)

type Option func(*Runner)

// WithContext sets a custom context for the Runner instance.
func WithContext(ctx context.Context) Option {
	return func(r *Runner) {
		r.parentCtx = ctx
	}
}

// WithLogHandler sets a custom slog handler for the Runner instance.
func WithLogHandler(handler slog.Handler) Option {
	return func(r *Runner) {
		if handler != nil {
			r.logger = slog.New(handler).WithGroup("http.Runner")
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
