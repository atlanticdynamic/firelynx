// Package core provides the core functionality of the firelynx server
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/registry"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/robbyt/go-supervisor/supervisor"
)

const (
	VersionLatest  = config.VersionLatest  // Latest supported version
	VersionUnknown = config.VersionUnknown // Used when version is not specified
)

// Interface guards: ensure Runner implements these interfaces
var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
)

type configCallback func() config.Config

// Runner implements the supervisor.Runnable and supervisor.Reloadable interfaces.
type Runner struct {
	configCallback configCallback
	mutex          sync.Mutex

	parentCtx    context.Context
	parentCancel context.CancelFunc
	runCtx       context.Context
	runCancel    context.CancelFunc

	appRegistry   apps.Registry
	logger        *slog.Logger
	currentConfig *config.Config // Current domain configuration for use by GetListenersConfigCallback
	serverErrors  chan error     // Channel for async server errors
}

// New creates a new Runner instance.
//
// Parameters:
//   - configCallback: a function that returns a pre-validated copy of the domain configuration.
//     This callback is typically provided by a component such as the cfgservice package,
//     which receives protobuf messages, converts them into domain config objects, and
//     ensures the configuration is ready for use by the Runner.
//   - opts: optional functional options for customizing the Runner.
//
// Returns a pointer to the Runner and an error if initialization fails.
func New(configCallback configCallback, opts ...Option) (*Runner, error) {
	r := &Runner{
		configCallback: configCallback,
		logger:         slog.Default(),
		serverErrors:   make(chan error, 10), // Buffer for async errors
	}
	r.parentCtx, r.parentCancel = context.WithCancel(context.Background())

	// Apply functional options
	for _, opt := range opts {
		opt(r)
	}

	r.appRegistry = registry.New()
	if err := r.registerBuiltInApps(); err != nil {
		return nil, fmt.Errorf("failed to register built-in apps: %w", err)
	}

	return r, nil
}

func (r *Runner) String() string {
	return "core.Runner"
}

// Run implements the Runnable interface and starts the Runner.
//
// Parameters:
//   - ctx: the parent context for the Runner.
//
// Returns an error if the Runner fails to start.
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Info("Starting Runner")
	r.runCtx, r.runCancel = context.WithCancel(ctx)
	defer r.runCancel()

	r.logger.Debug("Booting Runner")
	if err := r.boot(); err != nil {
		return fmt.Errorf("failed to boot Runner: %w", err)
	}
	r.logger.Debug("Runner boot completed")

	// Block here until either runCtx or parentCtx is canceled or an error occurs
	var result error
	select {
	case <-r.runCtx.Done():
		r.logger.Debug("Run context canceled")
		result = nil
	case <-r.parentCtx.Done():
		r.logger.Debug("Parent context canceled")
		result = r.parentCtx.Err()
	case err := <-r.serverErrors:
		r.logger.Error("Received server error", "error", err)
		result = err
	}

	r.logger.Debug("Runner shutting down")
	return result
}

// Stop implements the Runnable interface and stops the Runner.
func (r *Runner) Stop() {
	r.logger.Debug("Stopping Runner")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.runCancel()
	r.logger.Debug("Runner stopped")
}

// boot handles the initialization phase of the Runner by loading the initial configuration.
func (r *Runner) boot() error {
	r.logger.Debug("Fetching initial configuration")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.configCallback == nil {
		return errors.New("config callback is nil")
	}

	// load the initial config from the callback provided by the cfg service
	domainConfig := r.configCallback()
	r.currentConfig = &domainConfig
	return nil
}

// Reload implements the Reloadable interface and reloads the Runner with the latest configuration.
func (r *Runner) Reload() {
	r.logger.Debug("Reloading Runner")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Get latest configuration
	if r.configCallback == nil {
		r.logger.Error("Cannot reload: config callback is nil")
		r.serverErrors <- errors.New("config callback is nil during reload")
		return
	}

	domainConfig := r.configCallback()

	// Update the current config - this will be picked up by GetListenersConfigCallback
	// when the composite runner calls it
	r.currentConfig = &domainConfig
}

// GetHTTPConfigCallback returns a configuration callback for the HTTP runner.
// This callback provides HTTP-specific configuration derived from the main domain config.
func (r *Runner) GetHTTPConfigCallback() http.ConfigCallback {
	return func() (*http.Config, error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		if r.currentConfig == nil {
			return nil, errors.New("no configuration available")
		}

		// Extract HTTP listeners from the domain config
		httpListeners := make([]http.ListenerConfig, 0)
		for _, l := range r.currentConfig.Listeners {
			// Only process HTTP listeners
			_, ok := l.GetHTTPOptions()
			if !ok {
				continue
			}

			// Find associated endpoint for this listener
			var endpointID string
			for _, e := range r.currentConfig.Endpoints {
				for _, id := range e.ListenerIDs {
					if id == l.ID {
						endpointID = e.ID
						break
					}
				}
				if endpointID != "" {
					break
				}
			}

			// Convert domain listener to HTTP-specific listener config
			httpListener := http.ListenerConfig{
				ID:           l.ID,
				Address:      l.Address,
				EndpointID:   endpointID,
				ReadTimeout:  l.GetReadTimeout(),
				WriteTimeout: l.GetWriteTimeout(),
				IdleTimeout:  l.GetIdleTimeout(),
				DrainTimeout: l.GetDrainTimeout(),
			}

			httpListeners = append(httpListeners, httpListener)
		}

		// Create HTTP config with app registry
		httpConfig := http.NewConfig(
			r.appRegistry,
			httpListeners,
		)

		return httpConfig, nil
	}
}
