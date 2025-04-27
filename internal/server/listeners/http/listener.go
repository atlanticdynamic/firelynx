// Package http provides HTTP listener implementation
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
)

// Default timeouts for HTTP servers
const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
)

// Listener represents an HTTP server instance
type Listener struct {
	id      string
	address string
	handler http.Handler
	server  *http.Server
	options config.HTTPListenerOptions
	logger  *slog.Logger
}

// ListenerOption configures a Listener
type ListenerOption func(*Listener)

// WithListenerLogger sets the logger for the Listener
func WithListenerLogger(logger *slog.Logger) ListenerOption {
	return func(l *Listener) {
		l.logger = logger
	}
}

// FromConfig creates a new Listener from configuration
func FromConfig(
	cfg *config.Listener,
	handler http.Handler,
	opts ...ListenerOption,
) (*Listener, error) {
	if cfg == nil {
		return nil, fmt.Errorf("listener config cannot be nil")
	}

	if cfg.Type != config.ListenerTypeHTTP {
		return nil, fmt.Errorf(
			"invalid listener type: %s, expected: %s",
			cfg.Type,
			config.ListenerTypeHTTP,
		)
	}

	if cfg.ID == "" {
		return nil, fmt.Errorf("listener ID cannot be empty")
	}

	if cfg.Address == "" {
		return nil, fmt.Errorf("listener address cannot be empty")
	}

	// Type assertion for HTTP options
	httpOptions, ok := cfg.Options.(config.HTTPListenerOptions)
	if !ok {
		return nil, fmt.Errorf("invalid listener options type: expected HTTPListenerOptions")
	}

	// Convert options to appropriate HTTP server settings
	readTimeout := DefaultReadTimeout
	writeTimeout := DefaultWriteTimeout
	idleTimeout := DefaultIdleTimeout

	if httpOptions.ReadTimeout != nil && httpOptions.ReadTimeout.AsDuration() > 0 {
		readTimeout = httpOptions.ReadTimeout.AsDuration()
	}

	if httpOptions.WriteTimeout != nil && httpOptions.WriteTimeout.AsDuration() > 0 {
		writeTimeout = httpOptions.WriteTimeout.AsDuration()
	}

	// Create HTTP server with provided options
	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Default logger
	logger := slog.Default()

	listener := &Listener{
		id:      cfg.ID,
		address: cfg.Address,
		handler: handler,
		server:  server,
		options: httpOptions,
		logger:  logger,
	}

	// Apply options
	for _, opt := range opts {
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
	// Create a timeout for server shutdown based on the drain timeout
	drainTimeout := 30 * time.Second
	if l.options.DrainTimeout != nil && l.options.DrainTimeout.AsDuration() > 0 {
		drainTimeout = l.options.DrainTimeout.AsDuration()
	}

	// Create a context with timeout for server shutdown
	shutdownCtx, cancel := context.WithTimeout(parentCtx, drainTimeout)
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := l.server.Shutdown(shutdownCtx); err != nil {
		l.logger.Error("Error during HTTP server shutdown", "error", err, "id", l.id)
		return fmt.Errorf("error shutting down HTTP server: %w", err)
	}

	l.logger.Info("HTTP listener stopped", "id", l.id)
	return nil
}

// Stop terminates the HTTP server
func (l *Listener) Stop() {
	l.logger.Info("Stopping HTTP listener", "id", l.id)
	err := l.shutdown(context.Background())
	if err != nil {
		l.logger.Error("Error during HTTP server shutdown", "error", err, "id", l.id)
	}
}

// UpdateHandler updates the HTTP handler used by the listener
func (l *Listener) UpdateHandler(handler http.Handler) {
	l.handler = handler
	l.server.Handler = handler
}
