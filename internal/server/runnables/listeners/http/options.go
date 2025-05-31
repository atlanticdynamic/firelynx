package http

import (
	"log/slog"
	"time"
)

type Option func(*Runner)

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

// WithSiphonTimeout sets the timeout for sending configs through the siphon channel.
func WithSiphonTimeout(timeout time.Duration) Option {
	return func(r *Runner) {
		r.siphonTimeout = timeout
	}
}
