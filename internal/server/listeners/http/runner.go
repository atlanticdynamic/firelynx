// Package http provides HTTP listener implementation
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http/wrapper"
	"github.com/robbyt/go-supervisor/runnables/composite"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Interface guards: ensure Manager implements these interfaces
var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
	_ supervisor.Stateable  = (*Runner)(nil)
)

// Runner manages HTTP listener instances based on configuration
type Runner struct {
	logger         *slog.Logger
	configCallback ConfigCallback
	runner         *composite.Runner[*wrapper.HttpServer]
	mutex          sync.Mutex
}

// ManagerOption configures a Manager
type ManagerOption func(*Runner)

// WithManagerLogger sets the logger for the Manager
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(m *Runner) {
		m.logger = logger
	}
}

// NewRunner creates a new HTTP listeners manager
func NewRunner(
	callback ConfigCallback,
	opts ...ManagerOption,
) (*Runner, error) {
	if callback == nil {
		return nil, errors.New("config callback is required")
	}

	// Initialize with default options
	manager := &Runner{
		logger:         slog.Default().WithGroup("http.Runner"),
		configCallback: callback,
	}

	// Apply options
	for _, opt := range opts {
		opt(manager)
	}

	// Initialize the composite runner with a config callback
	runnerConfigCallback := func() (*composite.Config[*wrapper.HttpServer], error) {
		return manager.getRunnerConfig()
	}

	runner, err := composite.NewRunner[*wrapper.HttpServer](runnerConfigCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite runner: %w", err)
	}
	manager.runner = runner

	return manager, nil
}

// String returns a unique identifier for the manager
func (r *Runner) String() string {
	return "http.Manager"
}

// Run starts the HTTP listener manager and all configured listeners
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting HTTP listener manager")

	// Just delegate to the composite runner
	return r.runner.Run(ctx)
}

// Reload triggers reloading of all HTTP listeners
func (r *Runner) Reload() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Reloading HTTP listener manager")

	// The composite runner will automatically call our config callback
	// and reload with the latest configuration
	r.runner.Reload()
}

// Stop terminates all HTTP listeners
func (r *Runner) Stop() {
	r.logger.Info("Stopping HTTP listener manager")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.runner != nil {
		r.runner.Stop()
	}
}

// GetListenerStates returns the states of all managed HTTP listeners
func (r *Runner) GetListenerStates() map[string]string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.runner == nil {
		return make(map[string]string)
	}

	return r.runner.GetChildStates()
}

// getRunnerConfig returns the current configuration for the composite runner
func (r *Runner) getRunnerConfig() (*composite.Config[*wrapper.HttpServer], error) {
	// Get the current HTTP-specific configuration
	httpConfig, err := r.configCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP config: %w", err)
	}

	// Log the config we received
	if httpConfig != nil {
		r.logger.Debug("Received HTTP config from callback",
			"listeners", len(httpConfig.Listeners))

		// Log details of first listener
		if len(httpConfig.Listeners) > 0 {
			listener := httpConfig.Listeners[0]
			r.logger.Debug("First HTTP listener details",
				"id", listener.ID,
				"address", listener.Address,
				"endpoint_id", listener.EndpointID)
		}
	} else {
		r.logger.Warn("Received nil HTTP config from callback")
	}

	// Convert to composite runner config format
	return r.buildCompositeConfig(httpConfig)
}

// buildCompositeConfig constructs the composite runner configuration from the HTTP config
func (r *Runner) buildCompositeConfig(cfg *Config) (*composite.Config[*wrapper.HttpServer], error) {
	if cfg == nil {
		r.logger.Warn("Received nil configuration")
		config, err := composite.NewConfig[*wrapper.HttpServer]("http-listeners", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create empty config: %w", err)
		}
		return config, nil
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	r.logger.Debug("Building HTTP listener configuration", "listeners", len(cfg.Listeners))

	var entries []composite.RunnableEntry[*wrapper.HttpServer]

	// Map HTTP listeners to child runnables
	for _, listenerCfg := range cfg.Listeners {
		r.logger.Debug(
			"Processing HTTP listener",
			"id",
			listenerCfg.ID,
			"address",
			listenerCfg.Address,
		)

		var httpRoutes []httpserver.Route

		// Determine which type of routing to use:
		// 1. New style: Route registry with endpoints (preferred)
		// 2. Legacy style: App registry with direct route mapping
		if cfg.IsUsingRouteRegistry() && listenerCfg.EndpointID != "" {
			// New style routing with endpoint ID and route registry
			r.logger.Debug(
				"Using route registry for listener",
				"id",
				listenerCfg.ID,
				"endpoint",
				listenerCfg.EndpointID,
			)

			// Create a route handler that will resolve routes from the registry
			routeHandler := NewRouteHandler(
				cfg.RouteRegistry,
				listenerCfg.EndpointID,
				r.logger.With(
					"listener", listenerCfg.ID,
					"endpoint", listenerCfg.EndpointID,
				),
			)

			// Create a single route that captures all paths and delegates to the route handler
			rootRouteID := fmt.Sprintf("%s:root", listenerCfg.ID)
			rootRoute, err := httpserver.NewRoute(rootRouteID, "/", routeHandler.ServeHTTP)
			if err != nil {
				r.logger.Error("Failed to create root HTTP route", "error", err)
				continue
			}

			// The route handler will handle all routes for this endpoint
			httpRoutes = []httpserver.Route{*rootRoute}
		} else if len(listenerCfg.Routes) > 0 {
			// Legacy style routing with direct route mapping
			r.logger.Debug(
				"Using legacy direct route mapping for listener",
				"id",
				listenerCfg.ID,
				"routes",
				len(listenerCfg.Routes),
			)

			// Convert our HTTP routes to httpserver.Route objects
			httpRoutes = make([]httpserver.Route, 0, len(listenerCfg.Routes))
			for _, route := range listenerCfg.Routes {
				// Create a unique route ID
				routeID := fmt.Sprintf("%s:%s", listenerCfg.ID, route.Path)

				// Create handler for this route
				appHandler := NewAppHandler(cfg.AppRegistry, []RouteConfig{route}, r.logger)

				// Create httpserver route
				httpRoute, err := httpserver.NewRoute(routeID, route.Path, appHandler.ServeHTTP)
				if err != nil {
					r.logger.Error("Failed to create HTTP route", "error", err)
					continue
				}
				httpRoutes = append(httpRoutes, *httpRoute)
			}
		} else {
			r.logger.Error(
				"Listener has neither endpoint ID nor routes, or endpoint ID is set but RouteRegistry is missing",
				"id", listenerCfg.ID,
				"endpoint_id", listenerCfg.EndpointID,
				"routes", len(listenerCfg.Routes),
				"registry", cfg.RouteRegistry != nil,
			)
			continue
		}

		// Create a domain listener config based on our HTTP config
		domainListener := &listeners.Listener{
			ID:      listenerCfg.ID,
			Address: listenerCfg.Address,
			Options: options.HTTP{
				ReadTimeout:  listenerCfg.ReadTimeout,
				WriteTimeout: listenerCfg.WriteTimeout,
				IdleTimeout:  listenerCfg.IdleTimeout,
				DrainTimeout: listenerCfg.DrainTimeout,
			},
		}

		// Create wrapper server
		httpServer, err := wrapper.NewHttpServer(
			domainListener,
			httpRoutes,
			wrapper.WithLogger(r.logger.With("listener", listenerCfg.ID)),
		)
		if err != nil {
			r.logger.Error(
				"Failed to create HTTP server wrapper",
				"id",
				listenerCfg.ID,
				"error",
				err,
			)
			continue
		}

		// Add to entries
		entry := composite.RunnableEntry[*wrapper.HttpServer]{
			Runnable: httpServer,
			Config:   nil, // No additional config needed
		}
		entries = append(entries, entry)
	}

	// Create composite config for the runner
	return composite.NewConfig("http-listeners", entries)
}
