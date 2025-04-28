// filepath: /Users/rterhaar/Dropbox/research/golang/firelynx/internal/server/cfgrpc/options.go
package cfgrpc

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
		r.listenAddr = addr
	}
}

// WithConfigPath sets the path to the configuration file.
func WithConfigPath(path string) Option {
	return func(r *Runner) {
		r.configPath = path
	}
}

// WithGRPCServerStarter sets a custom function to start the gRPC server.
// This is primarily used for testing.
func WithGRPCServerStarter(starter StartGRPCServerFunc) Option {
	return func(r *Runner) {
		if starter != nil {
			r.startGRPCServer = starter
		}
	}
}
