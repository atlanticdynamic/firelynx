// Package core provides the core functionality of the firelynx server
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/registry"
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

type configCallback func() *pb.ServerConfig

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

// New creates a new Runner instance
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

	// Initialize the app registry
	r.appRegistry = registry.New()
	echoApp := echo.New("echo")
	if err := r.appRegistry.RegisterApp(echoApp); err != nil {
		return nil, fmt.Errorf("failed to register echo app: %w", err)
	}

	return r, nil
}

func (r *Runner) String() string {
	return "core.Runner"
}

// Run implements the Runnable interface and starts the Runner
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

// Stop implements the Runnable interface and stops the Runner
func (r *Runner) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Stopping Runner")
	r.runCancel()
	r.logger.Debug("Runner stopped")
}

// boot handles the initialization phase of the Runner by loading the initial configuration
func (r *Runner) boot() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.configCallback == nil {
		return fmt.Errorf("configCallback is nil")
	}

	r.logger.Debug("Fetching initial configuration")
	serverConfig := r.configCallback()

	// Handle case with nil config
	if serverConfig == nil {
		r.logger.Debug("No initial configuration provided, starting with empty config")
		return nil // This is not an error, we can start with no config
	}

	// Convert the proto config to a domain config for listeners
	domainConfig, err := config.NewFromProto(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to convert protobuf config to domain config: %w", err)
	}

	r.currentConfig = domainConfig
	return nil
}

// Reload implements the Reloadable interface and reloads the Runner with the latest configuration
func (r *Runner) Reload() {
	r.logger.Debug("Reloading Runner")

	// Use a separate goroutine to handle the reload to avoid blocking
	go func() {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Get latest configuration
		if r.configCallback == nil {
			r.logger.Error("Cannot reload: config callback is nil")
			r.serverErrors <- errors.New("config callback is nil during reload")
			return
		}

		serverConfig := r.configCallback()

		// Simply log when we receive an empty config
		if serverConfig == nil {
			r.logger.Info("Empty configuration received during reload")
			// We can set currentConfig to nil to represent an empty configuration
			r.currentConfig = nil
			r.logger.Info("Runner reloaded with empty configuration")
			return
		}

		// Process the configuration
		domainConfig, err := config.NewFromProto(serverConfig)
		if err != nil {
			r.logger.Error("Failed to convert protobuf config to domain config", "error", err)
			r.serverErrors <- fmt.Errorf("failed to convert config during reload: %w", err)
			return
		}

		// Update the current config - this will be picked up by GetListenersConfigCallback
		// when the composite runner calls it
		r.currentConfig = domainConfig
		r.logger.Info("Runner reloaded successfully")
	}()
}
