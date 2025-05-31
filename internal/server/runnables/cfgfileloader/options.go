package cfgfileloader

import (
	"log/slog"
)

type Option func(*Runner)

// WithLogger sets a custom logger for the Runner instance.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Runner) {
		r.logger = logger
	}
}

// WithLogHandler sets a custom log handler for the Runner instance.
func WithLogHandler(handler slog.Handler) Option {
	return func(r *Runner) {
		r.logger = slog.New(handler)
	}
}
