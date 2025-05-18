// Package txmgr implements the transaction manager for configuration updates
// and adapters between domain config and runtime components.
//
// HTTP Listener Rewrite Plan:
// According to the HTTP listener rewrite plan, HTTP-specific configuration logic
// in this package will be moved to the HTTP listener package where it will implement 
// the SagaParticipant interface. This will allow each SagaParticipant to handle 
// its own configuration extraction and management, keeping this package focused
// on orchestration rather than HTTP-specific details.
package txmgr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Interface guards: ensure Runner implements these interfaces
var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
	_ supervisor.Stateable  = (*Runner)(nil)
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

// Runner implements the core server coordinator that manages configuration
// lifecycle and app collection.
//
// Note: The HTTP-specific references in this comment are outdated. According to the
// HTTP listener rewrite plan, HTTP server management functionality will be moved 
// to the HTTP listener package as a dedicated SagaParticipant implementation.
type Runner struct {
	// Required dependencies
	appCollection *apps.AppCollection
	logger        *slog.Logger

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

	// State management
	fsm finitestate.Machine
}

// NewRunner creates a new core runner that coordinates configuration and services.
// It follows the functional options pattern for configuration.
func NewRunner(
	configCallback func() config.Config,
	opts ...Option,
) (*Runner, error) {
	// Create initial empty app collection
	initialApps, err := apps.NewAppCollection([]apps.App{})
	if err != nil {
		return nil, fmt.Errorf("failed to create initial app collection: %w", err)
	}

	// Initialize with default options
	runner := &Runner{
		appCollection:  initialApps,
		logger:         slog.Default().WithGroup("txmgr.Runner"),
		configCallback: configCallback,
		serverErrors:   make(chan error, 10),
		stopCh:         make(chan struct{}),
		parentCtx:      context.Background(),
	}

	// Apply options
	for _, opt := range opts {
		opt(runner)
	}

	// Initialize the finite state machine
	fsmLogger := runner.logger.WithGroup("fsm")
	fsm, err := finitestate.New(fsmLogger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}
	runner.fsm = fsm

	return runner, nil
}

// updateAppsFromConfig creates app instances from the current configuration
// and updates the app collection with the new instances.
func (r *Runner) updateAppsFromConfig() error {
	// Skip if no configuration is available
	if r.currentConfig == nil || len(r.currentConfig.Apps) == 0 {
		// Create an empty app collection
		emptyCollection, err := apps.NewAppCollection([]apps.App{})
		if err != nil {
			return fmt.Errorf("failed to create empty app collection: %w", err)
		}
		r.appCollection = emptyCollection
		return nil
	}

	// Create app instances slice with initial capacity
	validApps := make([]apps.App, 0, len(r.currentConfig.Apps))

	// Process each app definition
	for _, appDef := range r.currentConfig.Apps {
		// Create app instance based on type
		creator, exists := apps.AvailableAppImplementations[appDef.Config.Type()]
		if !exists {
			continue
		}

		app, err := creator(appDef.ID, appDef.Config)
		if err != nil {
			return fmt.Errorf("failed to create app %s: %w", appDef.ID, err)
		}

		validApps = append(validApps, app)
	}

	// Create new immutable app collection
	newCollection, err := apps.NewAppCollection(validApps)
	if err != nil {
		return fmt.Errorf("failed to create app collection: %w", err)
	}

	// Update app collection reference
	r.appCollection = newCollection
	return nil
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

	// Debug log for config callback
	r.logger.Debug("Getting initial configuration from callback")
	domainConfig := r.configCallback()

	// Debug log for config state
	r.logger.Debug("Retrieved configuration details",
		"endpoints", len(domainConfig.Endpoints),
		"listeners", len(domainConfig.Listeners),
		"apps", len(domainConfig.Apps))

	r.currentConfig = &domainConfig

	// Create app instances from the domain config
	return r.updateAppsFromConfig()
}

// Run implements the supervisor.Runnable interface.
// It initializes and starts all server components, blocking until
// the context is cancelled or Stop is called.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
		return fmt.Errorf("failed to transition to booting state: %w", err)
	}

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
		if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
			r.logger.Error("Failed to transition to error state", "error", stateErr)
		}
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Transition to running state
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// Block until the context is cancelled
	<-ctx.Done()

	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
	}

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
	return "txmgr.Runner"
}

// Stop gracefully stops all server components.
func (r *Runner) Stop() {
	r.logger.Info("Stopping transaction manager")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Transition to stopping state
	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
		// Continue with shutdown despite the state transition error
	}

	// Signal stop to error monitor
	close(r.stopCh)

	// Trigger context cancellation
	if r.cancel != nil {
		r.cancel()
	}

	// Wait for all goroutines to exit
	r.wg.Wait()

	// Transition to stopped state
	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		r.logger.Error("Failed to transition to stopped state", "error", err)
	}
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

	// Transition to reloading state
	if err := r.fsm.Transition(finitestate.StatusReloading); err != nil {
		r.logger.Error("Failed to transition to reloading state", "error", err)
		// Continue with reload despite the state transition error
	}

	// Get latest configuration
	if r.configCallback == nil {
		r.logger.Error("Cannot reload configuration", "error", "config callback is nil")
		r.serverErrors <- errors.New("config callback is nil during reload")
		if err := r.fsm.Transition(finitestate.StatusError); err != nil {
			r.logger.Error("Failed to transition to error state", "error", err)
		}
		return
	}

	domainConfig := r.configCallback()

	// Update the current config
	r.currentConfig = &domainConfig

	// Update apps from the new configuration
	if err := r.updateAppsFromConfig(); err != nil {
		r.logger.Error("Failed to update apps from config", "error", err)
		r.serverErrors <- fmt.Errorf("failed to update apps during reload: %w", err)
		if err := r.fsm.Transition(finitestate.StatusError); err != nil {
			r.logger.Error("Failed to transition to error state", "error", err)
		}
		return
	}

	// Transition back to running state
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		r.logger.Error("Failed to transition back to running state", "error", err)
		if err := r.fsm.Transition(finitestate.StatusError); err != nil {
			r.logger.Error("Failed to transition to error state", "error", err)
		}
		return
	}

	r.logger.Debug("Configuration reloaded successfully")
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
