package core

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
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

// Runner implements the supervisor.Runnable and supervisor.Reloadable interfaces.
type Runner struct {
	configCallback func() *pb.ServerConfig
	reloadLock     sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// New creates a new Runner instance
func New(opts ...Option) (*Runner, error) {
	r := &Runner{
		logger: slog.Default(),
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// Apply functional options
	for _, opt := range opts {
		opt(r)
	}

	if r.configCallback == nil {
		return nil, errors.New("config callback is required")
	}

	return r, nil
}

func (r *Runner) String() string {
	return "core.Runner"
}

// Run implements the Runnable interface and starts the Runner
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting Runner")
	config := r.configCallback()
	if err := r.processConfig(config); err != nil {
		return err
	}

	// Block here until context is done
	select {
	case <-r.ctx.Done():
		r.logger.Debug("Runner context closed")
	case <-ctx.Done():
		r.logger.Debug("Runner external context closed")
	}

	return nil
}

// Stop implements the Runnable interface and stops the Runner
func (r *Runner) Stop() {
	r.reloadLock.Lock()
	defer r.reloadLock.Unlock()
	r.logger.Debug("Stopping Runner")
	r.cancel()
	r.logger.Debug("Runner stopped")
}

// Reload implements the Reloadable interface and reloads the Runner with the latest configuration
func (r *Runner) Reload() {
	r.reloadLock.Lock()
	defer r.reloadLock.Unlock()
	r.logger.Debug("Reloading Runner")
	config := r.configCallback()

	// "process" the updated configuration
	if err := r.processConfig(config); err != nil {
		r.logger.Error("Failed to reload", "error", err)
		return
	}

	r.logger.Info("Runner reloaded successfully")
}

// processConfig processes the provided configuration
func (r *Runner) processConfig(config *pb.ServerConfig) error {
	if config == nil {
		r.logger.Warn("Received nil configuration, using default empty config")
		version := VersionLatest
		config = &pb.ServerConfig{
			Version: &version,
		}
	}

	// Get version safely
	version := VersionUnknown
	if v := config.Version; v != nil {
		version = *v
	}
	r.logger.Debug("Processing configuration", "version", version)

	// For now, just print the configuration settings
	r.logConfig(config)

	return nil
}

// logConfig logs the configuration details
func (r *Runner) logConfig(config *pb.ServerConfig) {
	if config == nil {
		r.logger.Warn("Cannot log nil configuration")
		return
	}

	// Get version safely
	version := VersionUnknown
	if config.Version != nil {
		version = *config.Version
	}

	// Get counts safely
	listeners := 0
	if config.Listeners != nil {
		listeners = len(config.Listeners)
	}

	endpoints := 0
	if config.Endpoints != nil {
		endpoints = len(config.Endpoints)
	}

	apps := 0
	if config.Apps != nil {
		apps = len(config.Apps)
	}

	r.logger.Debug("Server configuration",
		"version", version,
		"listeners", listeners,
		"endpoints", endpoints,
		"apps", apps,
	)

	// Log details about listeners
	if config.Listeners != nil {
		for i, listener := range config.Listeners {
			id := "undefined"
			if listener.Id != nil {
				id = *listener.Id
			}

			address := "undefined"
			if listener.Address != nil {
				address = *listener.Address
			}

			r.logger.Info("Listener configuration",
				"index", i,
				"id", id,
				"address", address,
			)
		}
	}

	// Log details about endpoints
	if config.Endpoints != nil {
		for i, endpoint := range config.Endpoints {
			id := "undefined"
			if endpoint.Id != nil {
				id = *endpoint.Id
			}

			routes := 0
			if endpoint.Routes != nil {
				routes = len(endpoint.Routes)
			}

			r.logger.Info("Endpoint configuration",
				"index", i,
				"id", id,
				"listener_ids", endpoint.ListenerIds,
				"routes", routes,
			)
		}
	}

	// Log details about apps
	if config.Apps != nil {
		for i, app := range config.Apps {
			id := "undefined"
			if app.Id != nil {
				id = *app.Id
			}

			r.logger.Info("App configuration",
				"index", i,
				"id", id,
			)
		}
	}
}
