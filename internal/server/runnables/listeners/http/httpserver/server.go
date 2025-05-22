// Package httpserver provides the HTTP server implementation for the firelynx HTTP listener.
package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Ensure HTTPServer implements the required interfaces but NOT the reload interfaces
var (
	_ supervisor.Runnable  = (*HTTPServer)(nil)
	_ supervisor.Stateable = (*HTTPServer)(nil)
	// Deliberately NOT implementing supervisor.Reloadable or composite.ReloadableWithConfig
)

// HTTPTimeoutOptions contains timeout configuration for the HTTP server
type HTTPTimeoutOptions struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	DrainTimeout time.Duration
}

// serverImplementation is an interface for abstracting the underlying HTTP server sub-runnable implementation
type serverImplementation interface {
	Run(ctx context.Context) error
	Stop()
	GetState() string
	IsRunning() bool
	GetStateChan(ctx context.Context) <-chan string
}

// HTTPServer wraps the go-supervisor's httpserver.Runner
// It deliberately does NOT implement ReloadableWithConfig or Reloadable
// to prevent direct reloading outside the saga pattern.
type HTTPServer struct {
	id      string
	address string
	server  serverImplementation

	logger   *slog.Logger
	routes   []httpserver.Route
	timeouts HTTPTimeoutOptions
	mutex    sync.Mutex
}

// NewHTTPServer creates a new HTTP server with the specified configuration
func NewHTTPServer(
	id, address string,
	routes []httpserver.Route,
	timeouts HTTPTimeoutOptions,
	logger *slog.Logger,
) (*HTTPServer, error) {
	if logger == nil {
		logger = slog.Default().WithGroup("httpserver").With("id", id)
	}

	server := &HTTPServer{
		id:       id,
		address:  address,
		routes:   routes,
		timeouts: timeouts,
		logger:   logger,
	}

	if err := server.initializeRunner(); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP server runner: %w", err)
	}

	return server, nil
}

// initializeRunner creates and initializes the underlying httpserver.Runner
func (s *HTTPServer) initializeRunner() error {
	configCallback := func() (*httpserver.Config, error) {
		s.mutex.Lock()
		address := s.address
		routes := make([]httpserver.Route, len(s.routes))
		copy(routes, s.routes)
		readTimeout := s.timeouts.ReadTimeout
		writeTimeout := s.timeouts.WriteTimeout
		idleTimeout := s.timeouts.IdleTimeout
		drainTimeout := s.timeouts.DrainTimeout
		s.mutex.Unlock()

		options := []httpserver.ConfigOption{}

		if readTimeout > 0 {
			options = append(options, httpserver.WithReadTimeout(readTimeout))
		}
		if writeTimeout > 0 {
			options = append(options, httpserver.WithWriteTimeout(writeTimeout))
		}
		if idleTimeout > 0 {
			options = append(options, httpserver.WithIdleTimeout(idleTimeout))
		}
		if drainTimeout > 0 {
			options = append(options, httpserver.WithDrainTimeout(drainTimeout))
		}

		config, err := httpserver.NewConfig(address, routes, options...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP server config: %w", err)
		}

		return config, nil
	}

	runner, err := httpserver.NewRunner(
		httpserver.WithConfigCallback(configCallback),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP server runner: %w", err)
	}

	s.server = runner
	return nil
}

// String returns a unique identifier for this server
func (s *HTTPServer) String() string {
	return fmt.Sprintf("HTTPServer[%s]", s.id)
}

// Run starts the HTTP server
func (s *HTTPServer) Run(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", "address", s.address, "routes", len(s.routes))
	return s.server.Run(ctx)
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() {
	s.logger.Info("Stopping HTTP server", "address", s.address)
	s.server.Stop()
}

// GetState returns the current state of the server
func (s *HTTPServer) GetState() string {
	if s.server == nil {
		return "unknown"
	}
	return s.server.GetState()
}

// IsRunning returns whether the server is running
func (s *HTTPServer) IsRunning() bool {
	if s.server == nil {
		return false
	}
	return s.server.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (s *HTTPServer) GetStateChan(ctx context.Context) <-chan string {
	if s.server == nil {
		ch := make(chan string)
		go func() {
			<-ctx.Done()
			close(ch)
		}()
		return ch
	}
	return s.server.GetStateChan(ctx)
}

// UpdateRoutes updates the routes for this server
// This doesn't immediately reload - that will happen when the parent composite runner
// recreates this server instance (since we don't implement Reloadable)
func (s *HTTPServer) UpdateRoutes(routes []httpserver.Route) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Debug("Updating HTTP server routes", "routeCount", len(routes))
	s.routes = routes
}

// GetID returns the ID of this HTTP server
func (s *HTTPServer) GetID() string {
	return s.id
}

// GetAddress returns the address this server listens on
func (s *HTTPServer) GetAddress() string {
	return s.address
}
