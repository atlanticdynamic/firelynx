package cfgservice

import (
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

// WithListenAddr sets the address for the gRPC server to listen on.
func WithListenAddr(addr string) Option {
	return func(r *Runner) {
		if addr != "" {
			r.listenAddr = addr
		}
	}
}

// WithConfigPath sets the path to the configuration file.
func WithConfigPath(path string) Option {
	return func(r *Runner) {
		if path != "" {
			r.configPath = path
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
func WithTransactionStorage(storage TransactionStorage) Option {
	return func(r *Runner) {
		if storage != nil {
			r.txStorage = storage
		}
	}
}
