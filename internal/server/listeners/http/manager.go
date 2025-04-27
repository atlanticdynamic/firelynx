// Package http provides HTTP listener implementation
package http

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/robbyt/go-supervisor/runnables/composite"
)

// Manager handles the lifecycle of multiple HTTP listeners based on configuration changes
type Manager struct {
	logger      *slog.Logger
	registry    apps.Registry
	routeMapper *RouteMapper
	runner      *composite.Runner[*Listener]
}

// ManagerOption configures a Manager
type ManagerOption func(*Manager)

// WithManagerLogger sets the logger for the Manager
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// NewManager creates a new Manager
func NewManager(
	registry apps.Registry,
	configCallback func() *config.Config,
	opts ...ManagerOption,
) (*Manager, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is required")
	}

	m := &Manager{
		logger:      slog.Default().With("component", "http.Manager"),
		registry:    registry,
		routeMapper: nil, // Will be initialized below
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Initialize route mapper
	m.routeMapper = NewRouteMapper(registry, m.logger)

	// Create composite runner config callback
	runnerConfigCallback := func() (*composite.Config[*Listener], error) {
		cfg := configCallback()
		if cfg == nil {
			return nil, fmt.Errorf("config callback returned nil")
		}

		// Validate configuration
		if err := m.validateConfig(cfg); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		return m.buildCompositeConfig(cfg)
	}

	// Create composite runner
	runner, err := composite.NewRunner[*Listener](
		runnerConfigCallback,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite HTTP listener runner: %w", err)
	}

	m.runner = runner
	return m, nil
}

// String returns a unique identifier for the manager
func (m *Manager) String() string {
	return "http.Manager"
}

// Run starts the composite HTTP runner
func (m *Manager) Run(ctx context.Context) error {
	m.logger.Info("Starting HTTP listener manager")
	return m.runner.Run(ctx)
}

// Stop terminates all HTTP listeners
func (m *Manager) Stop() {
	m.logger.Info("Stopping HTTP listener manager")
	m.runner.Stop()
}

// Reload triggers a configuration reload on the composite runner
func (m *Manager) Reload() {
	m.logger.Info("Reloading HTTP listener manager")
	m.runner.Reload()
}

// GetListenerStates returns the states of all managed HTTP listeners
func (m *Manager) GetListenerStates() map[string]string {
	return m.runner.GetChildStates()
}

// validateConfig validates the configuration
func (m *Manager) validateConfig(cfg *config.Config) error {
	// Validate listeners
	for _, l := range cfg.Listeners {
		if l.Type != config.ListenerTypeHTTP {
			continue
		}

		// Validate listener ID
		if l.ID == "" {
			return fmt.Errorf("listener ID cannot be empty")
		}

		// Validate listener address
		if l.Address == "" {
			return fmt.Errorf("listener address cannot be empty")
		}

		// Validate HTTP options
		httpOpts, ok := l.Options.(config.HTTPListenerOptions)
		if !ok {
			return fmt.Errorf("invalid options type for HTTP listener %s", l.ID)
		}

		// Validate timeouts
		if httpOpts.ReadTimeout != nil && httpOpts.ReadTimeout.AsDuration() < 0 {
			return fmt.Errorf("invalid read timeout for HTTP listener %s", l.ID)
		}
		if httpOpts.WriteTimeout != nil && httpOpts.WriteTimeout.AsDuration() < 0 {
			return fmt.Errorf("invalid write timeout for HTTP listener %s", l.ID)
		}
		if httpOpts.DrainTimeout != nil && httpOpts.DrainTimeout.AsDuration() < 0 {
			return fmt.Errorf("invalid drain timeout for HTTP listener %s", l.ID)
		}
	}

	// Validate endpoints
	for _, e := range cfg.Endpoints {
		// Validate endpoint ID
		if e.ID == "" {
			return fmt.Errorf("endpoint ID cannot be empty")
		}

		// Validate listener IDs
		if len(e.ListenerIDs) == 0 {
			return fmt.Errorf("endpoint %s has no listener IDs", e.ID)
		}

		// Validate routes
		if len(e.Routes) == 0 {
			return fmt.Errorf("endpoint %s has no routes", e.ID)
		}

		// Validate each route
		for _, r := range e.Routes {
			// Validate app ID
			if r.AppID == "" {
				return fmt.Errorf("route in endpoint %s has no app ID", e.ID)
			}

			// Validate condition
			if r.Condition == nil {
				return fmt.Errorf("route in endpoint %s has no condition", e.ID)
			}

			// Validate HTTP path condition
			if httpCond, ok := r.Condition.(config.HTTPPathCondition); ok {
				if httpCond.Path == "" {
					return fmt.Errorf("HTTP path condition in endpoint %s has empty path", e.ID)
				}
			}
		}
	}

	return nil
}

// buildCompositeConfig constructs the composite runner configuration from the server config
func (m *Manager) buildCompositeConfig(cfg *config.Config) (*composite.Config[*Listener], error) {
	if cfg == nil {
		m.logger.Warn("Received nil configuration")
		config, err := composite.NewConfig[*Listener]("http-listeners", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create empty config: %w", err)
		}
		return config, nil
	}

	m.logger.Debug("Building HTTP listener configuration", "listeners", len(cfg.Listeners))

	// Detailed info about the config
	for i, l := range cfg.Listeners {
		m.logger.Debug(
			"Found listener in config",
			"index",
			i,
			"id",
			l.ID,
			"type",
			l.Type,
			"address",
			l.Address,
		)
	}

	var entries []composite.RunnableEntry[*Listener]

	// Process each HTTP listener in the configuration
	for _, l := range cfg.Listeners {
		// Skip non-HTTP listeners
		if l.Type != config.ListenerTypeHTTP {
			m.logger.Debug("Skipping non-HTTP listener", "id", l.ID, "type", l.Type)
			continue
		}

		m.logger.Debug("Processing HTTP listener", "id", l.ID, "address", l.Address, "type", l.Type)

		// Map all routes for this listener
		routes := m.routeMapper.MapEndpointsForListener(cfg, l.ID)

		// Create handler for the routes
		handler := NewAppHandler(m.registry, routes, m.logger)

		// Create listener from config
		listener, err := FromConfig(&l, handler, WithListenerLogger(m.logger))
		if err != nil {
			m.logger.Error("Failed to create HTTP listener", "id", l.ID, "error", err)
			continue
		}

		// Create an entry with the listener-specific config
		entry := composite.RunnableEntry[*Listener]{
			Runnable: listener,
			Config:   nil, // No per-listener config needed for now
		}

		entries = append(entries, entry)
	}

	config, err := composite.NewConfig[*Listener]("http-listeners", entries)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}
	return config, nil
}
