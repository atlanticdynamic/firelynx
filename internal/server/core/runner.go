// Package core provides the core functionality of the firelynx server
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/registry"
	httpserver "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
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

	ctx         context.Context
	cancel      context.CancelFunc
	appRegistry apps.Registry
	httpManager *httpserver.Manager
	logger      *slog.Logger
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

	// Initialize the app registry
	r.appRegistry = registry.New()

	// Register a sample echo app for testing
	echoApp := echo.New("echo")
	if err := r.appRegistry.RegisterApp(echoApp); err != nil {
		return nil, fmt.Errorf("failed to register echo app: %w", err)
	}

	// Create a config callback that converts protobuf config to domain config
	domainConfigCallback := func() *config.Config {
		pbConfig := r.configCallback()
		if pbConfig == nil {
			r.logger.Warn("Config callback returned nil protobuf config")
			return nil
		}

		r.logger.Debug("Converting protobuf config to domain config",
			"listeners_count", len(pbConfig.GetListeners()),
			"endpoints_count", len(pbConfig.GetEndpoints()),
			"apps_count", len(pbConfig.GetApps()))

		// Convert protobuf config to domain config
		domainConfig, err := config.NewFromProto(pbConfig)
		if err != nil {
			r.logger.Warn("Failed to convert protobuf config to domain config", "error", err)
			return nil
		}

		r.logger.Debug("Domain config conversion result",
			"listeners_count", len(domainConfig.Listeners),
			"endpoints_count", len(domainConfig.Endpoints),
			"apps_count", len(domainConfig.Apps))

		return domainConfig
	}

	// Initialize the HTTP manager
	httpManager, err := httpserver.NewManager(
		r.appRegistry,
		domainConfigCallback,
		httpserver.WithManagerLogger(r.logger.With("component", "http.Manager")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP manager: %w", err)
	}

	r.httpManager = httpManager

	return r, nil
}

func (r *Runner) String() string {
	return "core.Runner"
}

// Run implements the Runnable interface and starts the Runner
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting Runner")

	// Initial config must exist and be valid or we fail immediately
	serverConfig := r.configCallback()
	if serverConfig == nil {
		return fmt.Errorf("initial configuration is nil, cannot start server")
	}

	// Process initial config, fail if it's invalid
	if err := r.processConfig(serverConfig); err != nil {
		r.logger.Error("Failed to process initial config", "error", err)
		return fmt.Errorf("initial configuration is invalid: %w", err)
	}

	// Create a context for the HTTP manager
	httpCtx, httpCancel := context.WithCancel(ctx)
	defer httpCancel()

	// Start the HTTP manager in a goroutine
	httpErrCh := make(chan error, 1)
	go func() {
		if err := r.httpManager.Run(httpCtx); err != nil {
			httpErrCh <- err
		}
		close(httpErrCh)
	}()

	// Wait a short time to ensure HTTP manager is running
	time.Sleep(100 * time.Millisecond)

	// Block here until context is done or HTTP manager fails
	select {
	case err := <-httpErrCh:
		if err != nil {
			r.logger.Error("HTTP manager failed", "error", err)
			return err
		}
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

	// Stop the HTTP manager
	if r.httpManager != nil {
		r.httpManager.Stop()
	}

	r.cancel()
	r.logger.Debug("Runner stopped")
}

// Reload implements the Reloadable interface and reloads the Runner with the latest configuration
func (r *Runner) Reload() {
	r.reloadLock.Lock()
	defer r.reloadLock.Unlock()
	r.logger.Debug("Reloading Runner")
	serverConfig := r.configCallback()

	// "process" the updated configuration
	if err := r.processConfig(serverConfig); err != nil {
		r.logger.Error("Failed to reload", "error", err)
		return
	}

	// If HTTP manager exists, try to reload it
	if r.httpManager != nil {
		// Check if HTTP manager is in an error state by getting its child states
		states := r.httpManager.GetListenerStates()
		if len(states) == 0 {
			r.logger.Info("HTTP manager has no listeners, creating a new manager")

			// Create a domain config callback
			domainConfigCallback := func() *config.Config {
				serverConfig := r.configCallback()
				if serverConfig == nil {
					r.logger.Warn("Config callback returned nil protobuf config")
					return nil
				}

				// Convert protobuf config to domain config
				cfg, err := config.NewFromProto(serverConfig)
				if err != nil {
					r.logger.Warn(
						"Failed to convert protobuf config to domain config",
						"error",
						err,
					)
					return nil
				}
				return cfg
			}

			// Initialize a new HTTP manager
			httpManager, err := httpserver.NewManager(
				r.appRegistry,
				domainConfigCallback,
				httpserver.WithManagerLogger(r.logger.With("component", "http.Manager")),
			)
			if err != nil {
				r.logger.Error("Failed to create new HTTP manager", "error", err)
				return
			}

			// Replace the old manager
			oldManager := r.httpManager
			r.httpManager = httpManager

			// Stop the old manager if it exists
			if oldManager != nil {
				oldManager.Stop()
			}

			// Start the new manager in a goroutine
			go func() {
				if err := r.httpManager.Run(r.ctx); err != nil {
					r.logger.Error("HTTP manager failed", "error", err)
				}
			}()
		} else {
			// Normal reload
			r.httpManager.Reload()
		}
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
