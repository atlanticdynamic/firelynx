// Package core provides adapters between domain config and runtime components.
// This is the ONLY package that should import from internal/config.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/registry"
	http "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Interface guards: ensure Runner implements these interfaces
var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
)

// These are injected by goreleaser and correspond to the version of the build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

// Version returns a formatted string with version information.
func Version() string {
	return fmt.Sprintf("version %s (commit %s) built by %s on %s", version, commit, builtBy, date)
}

// Runner implements the core server coordinator that manages the HTTP server
// and its configuration lifecycle.
type Runner struct {
	// Required dependencies
	appRegistry apps.Registry
	logger      *slog.Logger

	// Internal state
	configCallback func() config.Config
	currentConfig  *config.Config

	// Control channels
	serverErrors chan error
	stopCh       chan struct{}

	// Synchronization
	mutex  sync.RWMutex
	wg     sync.WaitGroup
	cancel context.CancelFunc

	// Parent context handling
	parentCtx    context.Context
	parentCancel context.CancelFunc
}

// NewRunner creates a new core runner that coordinates configuration and services.
// It follows the functional options pattern for configuration.
func NewRunner(
	configCallback func() config.Config,
	opts ...Option,
) (*Runner, error) {
	// Initialize with default options
	runner := &Runner{
		appRegistry:    registry.New(),
		logger:         slog.Default().WithGroup("core.Runner"),
		configCallback: configCallback,
		serverErrors:   make(chan error, 10),
		stopCh:         make(chan struct{}),
		parentCtx:      context.Background(),
	}

	// Apply options
	for _, opt := range opts {
		opt(runner)
	}

	return runner, nil
}

// boot initializes the runner configuration without starting it.
// This is primarily used in tests to load the configuration without running services.
func (r *Runner) boot() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Skip if the config callback is nil
	if r.configCallback == nil {
		return errors.New("config callback is nil")
	}

	// Get the initial configuration
	domainConfig := r.configCallback()
	r.currentConfig = &domainConfig

	return nil
}

// Run implements the supervisor.Runnable interface.
// It initializes and starts all server components, blocking until
// the context is cancelled or Stop is called.
func (r *Runner) Run(ctx context.Context) error {
	// Create a cancellable context
	r.mutex.Lock()
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.mutex.Unlock()

	// Start monitoring for errors
	r.wg.Add(1)
	go r.monitorErrors(runCtx)

	// Load the initial configuration
	if err := r.boot(); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Block until the context is cancelled
	<-ctx.Done()
	return ctx.Err()
}

// monitorErrors watches the error channel and logs errors.
func (r *Runner) monitorErrors(ctx context.Context) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case err := <-r.serverErrors:
			if err != nil {
				r.logger.Error("Server error", "error", err)
			}
		}
	}
}

// String returns the name of this runnable component.
func (r *Runner) String() string {
	return "core.Runner"
}

// Stop gracefully stops all server components.
func (r *Runner) Stop() {
	r.logger.Info("Stopping core runner")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Signal stop to error monitor
	close(r.stopCh)

	// Trigger context cancellation
	if r.cancel != nil {
		r.cancel()
	}

	// Wait for all goroutines to exit
	r.wg.Wait()
}

// SetConfigProvider sets the callback used to get the current configuration.
func (r *Runner) SetConfigProvider(callback func() config.Config) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.configCallback = callback
}

// Reload reloads the configuration from the callback and updates all components.
// This implements the supervisor.Reloadable interface.
func (r *Runner) Reload() {
	r.logger.Debug("Reloading configuration...")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Get latest configuration
	if r.configCallback == nil {
		r.logger.Error("Cannot reload configuration", "error", "config callback is nil")
		r.serverErrors <- errors.New("config callback is nil during reload")
		return
	}

	domainConfig := r.configCallback()

	// Update the current config - this will be picked up by GetHTTPConfigCallback
	// when the composite runner calls it
	r.currentConfig = &domainConfig
	r.logger.Debug("Configuration reloaded successfully")
}

// GetHTTPConfigCallback returns a configuration callback for the HTTP runner.
// This callback provides HTTP-specific configuration derived from the main domain config.
func (r *Runner) GetHTTPConfigCallback() http.ConfigCallback {
	// Create a shared route registry for the HTTP server
	var routeRegistry *routing.Registry

	return func() (*http.Config, error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Handle the case where we don't have a configuration yet
		if r.currentConfig == nil {
			// For consistency with tests, return an error when configuration is nil
			// This ensures callers properly handle the case where configuration isn't ready
			return nil, fmt.Errorf("no configuration available")
		}

		// Create a config adapter to handle conversion between domain and runtime models
		adapter := NewConfigAdapter(r.currentConfig, r.appRegistry, r.logger)

		// Create or update the route registry if needed
		if routeRegistry == nil {
			// Create a routing callback using the adapter
			routingCallback := adapter.RoutingConfigCallback()

			// Create a new route registry
			routeRegistry = routing.NewRegistry(r.appRegistry, routingCallback, r.logger)

			// Initial load of route configuration
			if err := routeRegistry.Reload(); err != nil {
				return nil, fmt.Errorf("failed to load route registry: %w", err)
			}
		} else {
			// Reload the existing registry with new configuration
			if err := routeRegistry.Reload(); err != nil {
				return nil, fmt.Errorf("failed to reload route registry: %w", err)
			}
		}

		// Get HTTP configuration from adapter
		return adapter.HTTPConfigCallback(routeRegistry)()
	}
}

// PollConfig starts the background polling of configuration every interval.
// This is useful for file-based configurations that might change.
// TODO this is a mess, and it should be removed. Reload is triggered by go-supervisor's signal handler.
func (r *Runner) PollConfig(ctx context.Context, interval time.Duration) {
	// Exit early if interval is zero or less
	if interval <= 0 {
		return
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-r.stopCh:
				return
			case <-ticker.C:
				r.Reload() // Call Reload without checking return value since it's now void
			}
		}
	}()
}
