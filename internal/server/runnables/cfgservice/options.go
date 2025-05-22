package cfgservice

import (
	"context"
	"log/slog"
)

// Option represents a functional option for configuring Runner.
type Option func(*Runner)

// WithLogHandler sets a custom slog handler for the Runner instance.
func WithLogHandler(handler slog.Handler) Option {
	return func(r *Runner) {
		if handler != nil {
			r.logger = slog.New(handler).WithGroup("cfgrpc.Runner")
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

// WithGRPCServer sets a custom GRPCServer instance for testing.
// This allows providing a mock implementation of the GRPCServer interface.
func WithGRPCServer(server GRPCServer) Option {
	return func(r *Runner) {
		if server != nil {
			r.grpcServer = server
		}
	}
}

// WithTransactionStorage sets a custom transaction storage implementation.
func WithTransactionStorage(storage transactionStorage) Option {
	return func(r *Runner) {
		if storage != nil {
			r.txStorage = storage
		}
	}
}

// WithContext sets a custom context for the Runner instance.
func WithContext(ctx context.Context) Option {
	return func(r *Runner) {
		r.parentCtx = ctx
	}
}
