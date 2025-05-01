// Package http provides HTTP listener implementation
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ListenerOptions contains configuration for a HTTP server instance
type ListenerOptions struct {
	ID           string
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	DrainTimeout time.Duration
}

// Listener represents an HTTP server instance
type Listener struct {
	id           string
	address      string
	handler      http.Handler
	server       *http.Server
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	drainTimeout time.Duration
	logger       *slog.Logger
}

// ListenerOption configures a Listener
type ListenerOption func(*Listener)

// WithListenerLogger sets the logger for the Listener
func WithListenerLogger(logger *slog.Logger) ListenerOption {
	return func(l *Listener) {
		l.logger = logger
	}
}

// NewListener creates a new Listener with the given handler and options
func NewListener(
	handler http.Handler,
	opts ListenerOptions,
	listenerOpts ...ListenerOption,
) (*Listener, error) {
	if opts.ID == "" {
		return nil, fmt.Errorf("listener ID cannot be empty")
	}

	if opts.Address == "" {
		return nil, fmt.Errorf("listener address cannot be empty")
	}

	// Use default timeouts if not specified
	readTimeout := DefaultReadTimeout
	if opts.ReadTimeout > 0 {
		readTimeout = opts.ReadTimeout
	}

	writeTimeout := DefaultWriteTimeout
	if opts.WriteTimeout > 0 {
		writeTimeout = opts.WriteTimeout
	}

	idleTimeout := DefaultIdleTimeout
	if opts.IdleTimeout > 0 {
		idleTimeout = opts.IdleTimeout
	}

	drainTimeout := DefaultDrainTimeout
	if opts.DrainTimeout > 0 {
		drainTimeout = opts.DrainTimeout
	}

	// Create HTTP server with provided options
	server := &http.Server{
		Addr:         opts.Address,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Default logger
	logger := slog.Default()

	// Finish creating the listener instance
	listener := &Listener{
		id:           opts.ID,
		address:      opts.Address,
		handler:      handler,
		server:       server,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		idleTimeout:  idleTimeout,
		drainTimeout: drainTimeout,
		logger:       logger,
	}

	// Apply options to listener
	for _, opt := range listenerOpts {
		opt(listener)
	}

	return listener, nil
}

// String returns a unique identifier for the listener
func (l *Listener) String() string {
	return fmt.Sprintf("http.Listener[%s]", l.id)
}

// Run starts the HTTP server and blocks until the context is cancelled
func (l *Listener) Run(ctx context.Context) error {
	l.logger.Info("Starting HTTP listener", "id", l.id, "address", l.address)

	errCh := make(chan error, 1)
	go func() {
		if err := l.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		l.logger.Info("Stopping HTTP listener", "id", l.id)
		return l.shutdown(ctx)
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	}
}

// shutdown gracefully shuts down the HTTP server with a timeout
func (l *Listener) shutdown(parentCtx context.Context) error {
	l.logger.Debug("Gracefully shutting down HTTP server", "id", l.id, "address", l.address)

	// Create a timeout for graceful shutdown
	shutdownTimeout := l.drainTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = DefaultDrainTimeout
	}

	shutdownCtx, cancel := context.WithTimeout(parentCtx, shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := l.server.Shutdown(shutdownCtx); err != nil {
		l.logger.Error("Failed to gracefully shutdown server", "id", l.id, "error", err)
		return err
	}

	l.logger.Info("HTTP server shutdown complete", "id", l.id)
	return nil
}

// Stop terminates the HTTP server
func (l *Listener) Stop() {
	l.logger.Info("Stopping HTTP listener", "id", l.id)
	if err := l.shutdown(context.Background()); err != nil {
		l.logger.Error("Failed to stop HTTP listener", "id", l.id, "error", err)
	}
}

// UpdateHandler updates the HTTP handler used by the listener
func (l *Listener) UpdateHandler(handler http.Handler) {
	l.handler = handler
	l.server.Handler = handler
}
