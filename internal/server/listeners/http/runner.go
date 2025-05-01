// Package http provides HTTP listener implementation
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http/wrapper"
	"github.com/robbyt/go-supervisor/runnables/composite"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Interface guards: ensure Manager implements these interfaces
var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
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

	// Run the composite runner which will handle all HTTP listeners
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

	// Convert to composite runner config format
	return r.buildCompositeConfig(httpConfig)
}

// validateConfig ensures that the configuration is valid
func (r *Runner) validateConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("configuration is nil")
	}

	if cfg.Registry == nil {
		return errors.New("registry is nil")
	}

	// At least one listener is required
	if len(cfg.Listeners) == 0 {
		return errors.New("no listeners defined")
	}

	// Validate listeners
	for _, listener := range cfg.Listeners {
		if listener.ID == "" {
			return errors.New("listener id is required")
		}
		if listener.Address == "" {
			return errors.New("listener address is required")
		}
		if listener.DrainTimeout < 0 {
			return fmt.Errorf("invalid drain timeout for HTTP listener %s", listener.ID)
		}
		if listener.IdleTimeout < 0 {
			return fmt.Errorf("invalid idle timeout for HTTP listener %s", listener.ID)
		}

		// Validate routes
		for _, route := range listener.Routes {
			// Validate route path
			if route.Path == "" {
				return fmt.Errorf("route in listener %s has empty path", listener.ID)
			}

			// Validate app ID
			if route.AppID == "" {
				return fmt.Errorf(
					"route %s in listener %s has empty app ID",
					route.Path,
					listener.ID,
				)
			}
		}
	}

	return nil
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
	if err := r.validateConfig(cfg); err != nil {
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

		// Convert our HTTP routes to httpserver.Route objects
		httpRoutes := make([]httpserver.Route, 0, len(listenerCfg.Routes))
		for _, route := range listenerCfg.Routes {
			// Create a unique route ID
			routeID := fmt.Sprintf("%s:%s", listenerCfg.ID, route.Path)

			// Create handler for this route
			appHandler := NewAppHandler(cfg.Registry, []Route{route}, r.logger)

			// Create httpserver route
			httpRoute, err := httpserver.NewRoute(routeID, route.Path, appHandler.ServeHTTP)
			if err != nil {
				r.logger.Error("Failed to create HTTP route", "error", err)
				continue
			}
			httpRoutes = append(httpRoutes, *httpRoute)
		}

		// Create a domain listener config based on our HTTP config
		domainListener := &config.Listener{
			ID:      listenerCfg.ID,
			Type:    config.ListenerTypeHTTP,
			Address: listenerCfg.Address,
			Options: config.HTTPListenerOptions{
				ReadTimeout:  convertToDurationPb(listenerCfg.ReadTimeout),
				WriteTimeout: convertToDurationPb(listenerCfg.WriteTimeout),
				IdleTimeout:  convertToDurationPb(listenerCfg.IdleTimeout),
				DrainTimeout: convertToDurationPb(listenerCfg.DrainTimeout),
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

// convertToDurationPb converts a Go duration to a protobuf duration
func convertToDurationPb(d time.Duration) *durationpb.Duration {
	if d <= 0 {
		return nil
	}
	return durationpb.New(d)
}
