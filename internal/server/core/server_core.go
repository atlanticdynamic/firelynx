package core

import (
	"context"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Version constants used in the server core
const (
	VersionLatest  = config.VersionLatest  // Latest supported version
	VersionUnknown = config.VersionUnknown // Used when version is not specified
)

// ServerCore implements the core server functionality.
// It implements supervisor.Runnable and supervisor.Reloadable interfaces
// for lifecycle management and dynamic configuration updates.

// Interface guards: ensure ServerCore implements required interfaces
var (
	_ supervisor.Runnable   = (*ServerCore)(nil)
	_ supervisor.Reloadable = (*ServerCore)(nil)
)

type ServerCore struct {
	logger     *slog.Logger
	configFunc func() *pb.ServerConfig

	// For concurrent operations
	reloadLock sync.Mutex
}

// Config for creating a new ServerCore
type Config struct {
	Logger     *slog.Logger
	ConfigFunc func() *pb.ServerConfig
}

// New creates a new ServerCore instance
func New(cfg Config) *ServerCore {
	return &ServerCore{
		logger:     cfg.Logger,
		configFunc: cfg.ConfigFunc,
	}
}

// Run implements the Runnable interface and starts the ServerCore
func (sc *ServerCore) Run(ctx context.Context) error {
	sc.logger.Info("Starting ServerCore")

	// Initial config load and processing
	config := sc.configFunc()
	if err := sc.processConfig(config); err != nil {
		return err
	}

	// Block until context is done
	<-ctx.Done()
	sc.logger.Info("ServerCore shutting down")

	return nil
}

// Stop implements the Runnable interface and stops the ServerCore
func (sc *ServerCore) Stop() {
	sc.logger.Info("Stopping ServerCore")
	// When used with the supervisor, the supervisor will cancel the context
	// passed to Run, which will cause Run to return
}

// Reload implements the Reloadable interface and reloads the ServerCore with the latest configuration
func (sc *ServerCore) Reload() {
	sc.reloadLock.Lock()
	defer sc.reloadLock.Unlock()

	sc.logger.Info("Reloading ServerCore")

	// Get the latest configuration
	config := sc.configFunc()

	// Process the updated configuration
	if err := sc.processConfig(config); err != nil {
		sc.logger.Error("Failed to reload", "error", err)
		return
	}

	sc.logger.Info("ServerCore reloaded successfully")
}

// processConfig processes the provided configuration
func (sc *ServerCore) processConfig(config *pb.ServerConfig) error {
	if config == nil {
		sc.logger.Warn("Received nil configuration, using default empty config")
		version := VersionLatest
		config = &pb.ServerConfig{
			Version: &version,
		}
	}

	// Get version safely
	version := VersionUnknown
	if config.Version != nil {
		version = *config.Version
	}

	sc.logger.Info("Processing configuration", "version", version)

	// For now, just log the configuration settings
	sc.logConfig(config)

	return nil
}

// logConfig logs the configuration details
func (sc *ServerCore) logConfig(config *pb.ServerConfig) {
	if config == nil {
		sc.logger.Warn("Cannot log nil configuration")
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

	sc.logger.Info("Server configuration",
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

			sc.logger.Info("Listener configuration",
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

			sc.logger.Info("Endpoint configuration",
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

			sc.logger.Info("App configuration",
				"index", i,
				"id", id,
			)
		}
	}
}
