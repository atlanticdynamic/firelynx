// Package http provides wrappers around the go-supervisor HTTP server functionality
package wrapper

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
)

var (
	_ supervisor.Runnable   = (*HttpServer)(nil)
	_ supervisor.Reloadable = (*HttpServer)(nil)
)

// ServerOption configures a ServerWrapper
type ServerOption func(*HttpServer)

// WithLogger sets the logger for the ServerWrapper
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *HttpServer) {
		s.logger = logger
	}
}

// HttpServer wraps the go-supervisor httpserver.Runner.
// It adds configuration management, logging, and dynamic route updates.
type HttpServer struct {
	id      string
	address string
	runner  *httpserver.Runner
	logger  *slog.Logger
	routes  []httpserver.Route

	// Timeout options, loaded from the config.HTTPListenerOptions
	ReadTimeout  *time.Duration
	WriteTimeout *time.Duration
	DrainTimeout *time.Duration
	IdleTimeout  *time.Duration

	mutex sync.Mutex
}

// NewHttpServer creates a new wrapper for the go-supervisor httpserver.Runner from a listener configuration and routes
func NewHttpServer(
	listener *listeners.Listener,
	routes []httpserver.Route,
	opts ...ServerOption,
) (*HttpServer, error) {
	// Validate input parameters
	if err := validateListenerConfig(listener); err != nil {
		return nil, err
	}

	// Extract HTTP options from listener config
	httpOptions, err := extractHTTPOptions(listener)
	if err != nil {
		return nil, err
	}

	// Create the wrapper with default logger
	logger := slog.Default().WithGroup("http.ServerWrapper").With("id", listener.ID)

	// Initialize the wrapper with required fields
	wrapper := &HttpServer{
		id:      listener.ID,
		address: listener.Address,
		logger:  logger,
		routes:  routes,
	}

	// Get timeout values from options
	readTimeout := httpOptions.GetReadTimeout()
	if readTimeout > 0 {
		wrapper.ReadTimeout = &readTimeout
	}

	writeTimeout := httpOptions.GetWriteTimeout()
	if writeTimeout > 0 {
		wrapper.WriteTimeout = &writeTimeout
	}

	drainTimeout := httpOptions.GetDrainTimeout()
	if drainTimeout > 0 {
		wrapper.DrainTimeout = &drainTimeout
	}

	idleTimeout := httpOptions.GetIdleTimeout()
	if idleTimeout > 0 {
		wrapper.IdleTimeout = &idleTimeout
	}

	// Apply custom options
	for _, opt := range opts {
		opt(wrapper)
	}

	// Create the underlying runner
	if err := wrapper.initializeRunner(); err != nil {
		return nil, err
	}

	return wrapper, nil
}

// validateListenerConfig validates the listener configuration
func validateListenerConfig(listener *listeners.Listener) error {
	if listener == nil {
		return fmt.Errorf("listener config cannot be nil")
	}

	if listener.Options == nil {
		return fmt.Errorf("listener options cannot be nil")
	}

	if listener.ID == "" {
		return fmt.Errorf("listener ID cannot be empty")
	}

	if listener.Address == "" {
		return fmt.Errorf("listener address cannot be empty")
	}

	return nil
}

// extractHTTPOptions extracts HTTP options from listener configuration
func extractHTTPOptions(listener *listeners.Listener) (options.HTTP, error) {
	httpOptions, ok := listener.Options.(options.HTTP)
	if !ok {
		return options.HTTP{}, fmt.Errorf(
			"invalid listener options type: expected HTTPOptions",
		)
	}
	return httpOptions, nil
}

// initializeRunner creates and initializes the underlying httpserver.Runner
func (s *HttpServer) initializeRunner() error {
	// Create the configuration callback
	configCallback := func() (*httpserver.Config, error) {
		httpRoutes := s.routes
		options := s.buildConfigOptions()

		// Create httpserver config with the routes and options
		config, err := httpserver.NewConfig(
			s.address,
			httpRoutes,
			options...,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP server config: %w", err)
		}

		return config, nil
	}

	// Create the httpserver runner
	runner, err := httpserver.NewRunner(
		httpserver.WithConfigCallback(configCallback),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP server runner: %w", err)
	}

	s.runner = runner
	return nil
}

// buildConfigOptions creates a slice of configuration options based on the HTTP options
func (s *HttpServer) buildConfigOptions() []httpserver.ConfigOption {
	options := []httpserver.ConfigOption{}

	// Set timeout options if configured
	if s.ReadTimeout != nil {
		options = append(options, httpserver.WithReadTimeout(*s.ReadTimeout))
	}
	if s.WriteTimeout != nil {
		options = append(options, httpserver.WithWriteTimeout(*s.WriteTimeout))
	}
	if s.DrainTimeout != nil {
		options = append(options, httpserver.WithDrainTimeout(*s.DrainTimeout))
	}
	if s.IdleTimeout != nil {
		options = append(options, httpserver.WithIdleTimeout(*s.IdleTimeout))
	}

	return options
}

// String returns a unique identifier for the server
func (s *HttpServer) String() string {
	return fmt.Sprintf("HTTPServer[%s]", s.id)
}

// Run starts the HTTP server
func (s *HttpServer) Run(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", "id", s.id, "address", s.address)
	return s.runner.Run(ctx)
}

// Stop terminates the HTTP server
func (s *HttpServer) Stop() {
	s.logger.Info("Stopping HTTP server", "id", s.id)
	s.runner.Stop()
}

// Reload triggers reloading of the HTTP server configuration
func (s *HttpServer) Reload() {
	s.logger.Info("Reloading HTTP server", "id", s.id)
	s.runner.Reload()
}

// UpdateRoutes updates the routes for this server
func (s *HttpServer) UpdateRoutes(routes []httpserver.Route) {
	s.logger.Info("Updating HTTP server routes", "id", s.id, "routes", len(routes))
	s.routes = routes
	s.Reload()
}

// ReloadWithConfig implements the composite.ReloadableWithConfig interface.
// This allows the composite runner to pass configuration directly to this runnable.
func (s *HttpServer) ReloadWithConfig(config any) {
	s.logger.Info("Reloading HTTP server with config", "id", s.id)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if updated := s.processConfigUpdate(config); updated {
		// Trigger reload if config was updated
		s.Reload()
	}
}

// processConfigUpdate processes a configuration update and returns true if routes were updated
func (s *HttpServer) processConfigUpdate(config any) bool {
	// Handle different config types
	switch typedConfig := config.(type) {
	case map[string]any:
		return s.processMapConfig(typedConfig)

	case []httpserver.Route:
		// We received new routes directly
		s.logger.Debug("Updating routes directly", "id", s.id, "routeCount", len(typedConfig))
		s.routes = typedConfig
		return true

	default:
		// We received some other type of config - log it and return false
		s.logger.Debug("Received unknown config type during reload", "id", s.id, "configType", fmt.Sprintf("%T", config))
		return false
	}
}

// processMapConfig processes a map-based configuration update
func (s *HttpServer) processMapConfig(configMap map[string]any) bool {
	s.logger.Debug("Processing map config during reload", "id", s.id)

	// Check for routes update in the map
	if routesData, ok := configMap["routes"]; ok && routesData != nil {
		if routes, ok := routesData.([]httpserver.Route); ok {
			s.logger.Debug("Updating routes from map config", "id", s.id, "routeCount", len(routes))
			s.routes = routes
			return true
		}
		s.logger.Warn(
			"Routes data is not of expected type",
			"id",
			s.id,
			"type",
			fmt.Sprintf("%T", routesData),
		)
	}

	// Other config updates could be processed here
	// We don't modify listener address because that would require a completely new server

	return false
}

// GetID returns the ID of this HTTP server
func (s *HttpServer) GetID() string {
	return s.id
}

// GetAddress returns the address this server listens on
func (s *HttpServer) GetAddress() string {
	return s.address
}
