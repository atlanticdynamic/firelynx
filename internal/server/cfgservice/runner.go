// Runner manages configuration state and serves a gRPC API for clients to retrieve
// and update the configuration. It integrates with the supervisor package for
// lifecycle management, and implements the ReloadSender interface to allow
// subscribers to detect configuration changes.
package cfgservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/cfgservice/server"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Interface guard: ensure Runner implements required interfaces
var (
	_ supervisor.Runnable     = (*Runner)(nil)
	_ supervisor.ReloadSender = (*Runner)(nil)
	_ supervisor.Stateable    = (*Runner)(nil)
	_ supervisor.Reloadable   = (*Runner)(nil)
)

type Runner struct {
	// Embed the UnimplementedConfigServiceServer for gRPC compatibility
	pb.UnimplementedConfigServiceServer

	logger   *slog.Logger
	config   *config.Config
	configMu sync.RWMutex

	// gRPC server stuff
	grpcServer GRPCServer
	listenAddr string
	grpcLock   sync.RWMutex

	// Initial config path
	configPath string

	// For triggering a reload when a new config is received
	reloadCh chan struct{}

	fsm finitestate.Machine
}

// NewRunner creates a new Runner instance with the provided functional options.
func NewRunner(opts ...Option) (*Runner, error) {
	r := &Runner{
		logger:   slog.Default(),
		reloadCh: make(chan struct{}, 1),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(r)
	}

	// Check if we have either a config path or listen address
	if r.configPath == "" && r.listenAddr == "" {
		return nil, errors.New("either a config path or a listen address must be provided")
	}

	// Initialize the finite state machine
	fsmLogger := r.logger.WithGroup("fsm")
	fsm, err := finitestate.New(fsmLogger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}
	r.fsm = fsm

	return r, nil
}

func (r *Runner) String() string {
	return "cfgrpc.Runner"
}

// Run starts the configuration service and blocks until the context is canceled.
// It first initializes with an empty configuration, attempts to load from disk
// if a config path was provided, and finally starts the gRPC server if a listen
// address was configured. This ordering ensures we have a valid configuration
// before accepting client connections.
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting Runner")

	if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
		return fmt.Errorf("failed to transition to booting state: %w", err)
	}

	r.grpcLock.RLock()
	grpcServer := r.grpcServer
	r.grpcLock.RUnlock()
	if grpcServer != nil {
		if err := r.fsm.Transition(finitestate.StatusError); err != nil {
			r.logger.Error("Failed to transition to error state", "error", err)
		}
		return errors.New("gRPC server is already running")
	}

	// Initialize with at least an empty config
	r.configMu.Lock()
	r.config = &config.Config{
		Version: config.VersionLatest,
	}
	r.configMu.Unlock()

	// Load initial configuration if path is provided and not already loaded
	if r.configPath != "" {
		r.configMu.RLock()
		// Check if config has actual content, not just an empty placeholder
		hasContent := r.config != nil &&
			(len(r.config.Listeners) > 0 || len(r.config.Endpoints) > 0 || len(r.config.Apps) > 0)
		r.configMu.RUnlock()

		if !hasContent {
			r.logger.Debug("No meaningful config content detected, loading from disk")
			if err := r.LoadInitialConfig(); err != nil {
				r.logger.Error("Failed to load initial configuration", "error", err)

				if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
					r.logger.Error("Failed to transition to error state", "error", stateErr)
				}

				// If we don't have a listen address, fail immediately
				if r.listenAddr == "" {
					return err
				}
				// Otherwise continue with the empty config
				// Reset state to allow transition to running later
				if stateErr := r.fsm.SetState(finitestate.StatusBooting); stateErr != nil {
					r.logger.Error("Failed to reset state to booting", "error", stateErr)
				} else {
					r.logger.Debug("Reset state to booting successfully")
				}
			}
		} else {
			r.logger.Debug("Config with content already loaded, skipping initial load")
		}
	}

	// Start gRPC server if listen address is provided
	if r.listenAddr != "" {
		var err error
		grpcServer, err = server.NewGRPCManager(r.logger, r.listenAddr, r)
		if err != nil {
			if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
				r.logger.Error("Failed to transition to error state", "error", stateErr)
			}
			return err
		}

		// lock before starting the server to make sure that Stop isn't being called while we're starting
		// which would cause a listener conflict
		r.grpcLock.Lock()
		if err = grpcServer.Start(ctx); err != nil {
			r.grpcLock.Unlock()
			if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
				r.logger.Error("Failed to transition to error state", "error", stateErr)
			}
			return err
		}
		// store the started server, for graceful shutdown later
		r.grpcServer = grpcServer
		r.grpcLock.Unlock()
	}

	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// Block until context is done
	<-ctx.Done()

	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
	}
	r.logger.Info("Runner shutting down")

	return nil
}

// Stop gracefully shuts down the gRPC server if one is running.
// A lock is held during the entire shutdown to prevent concurrent modifications
// to the configuration while the server is shutting down.
func (r *Runner) Stop() {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	r.logger.Debug("Stopping Runner")

	// Transition to stopping state
	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
		// Continue with shutdown despite the state transition error
	}

	// Stop the gRPC server if it's running
	r.grpcLock.RLock()
	grpcServer := r.grpcServer
	r.grpcLock.RUnlock()
	if grpcServer != nil {
		r.grpcLock.Lock()
		grpcServer.GracefulStop()
		r.grpcServer = nil
		r.grpcLock.Unlock()
		r.logger.Info("gRPC server stopped")
	}

	// Transition to stopped state
	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		r.logger.Error("Failed to transition to stopped state", "error", err)
	}
}

// Reload reloads the configuration from disk if a configuration file path is provided.
// This method is called by the supervisor when a SIGHUP signal is received.
func (r *Runner) Reload() {
	r.logger.Info("Reloading configuration from disk...")

	// Transition to reloading state
	if err := r.fsm.Transition(finitestate.StatusReloading); err != nil {
		r.logger.Error("Failed to transition to reloading state", "error", err)
		return
	}

	// If we have a config path, reload from disk
	if r.configPath != "" {
		// Log the file path being reloaded for debugging
		r.logger.Debug("Attempting to reload config from path", "path", r.configPath)

		if err := r.LoadInitialConfig(); err != nil {
			r.logger.Error("Failed to reload configuration from disk", "error", err)

			// Try to return to running state
			if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
				r.logger.Error("Failed to transition back to running state", "error", err)

				// If can't transition back to running, go to error state
				if errState := r.fsm.Transition(finitestate.StatusError); errState != nil {
					r.logger.Error("Failed to transition to error state", "error", errState)
				}
			}
			return
		}

		// Log the loaded configuration details for debugging
		r.configMu.RLock()
		if r.config != nil {
			// Log endpoint routes for debugging
			for i, endpoint := range r.config.Endpoints {
				r.logger.Info("Reloaded endpoint config",
					"index", i,
					"id", endpoint.ID,
					"routes", len(endpoint.Routes))

				// Get HTTP routes in a structured format
				httpRoutes := endpoint.Routes.GetStructuredHTTPRoutes()
				for j, httpRoute := range httpRoutes {
					r.logger.Info("Reloaded HTTP route",
						"endpoint", endpoint.ID,
						"route_idx", j,
						"path", httpRoute.PathPrefix)
				}
			}
		}
		r.configMu.RUnlock()

		r.logger.Info("Configuration reloaded successfully from disk", "path", r.configPath)
	} else {
		r.logger.Debug("No config path provided, skipping reload")
	}

	// Return to running state
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		r.logger.Error("Failed to transition back to running state", "error", err)
	}
}

// GetPbConfigClone returns the current domain config converted to a protobuf message.
// If the domain config is nil, a minimal valid config is returned rather than nil
// to simplify client code.
func (r *Runner) GetPbConfigClone() *pb.ServerConfig {
	r.configMu.RLock()
	cfg := r.config
	r.configMu.RUnlock()

	if cfg == nil {
		r.logger.Warn("Current configuration is nil, returning empty config")
		version := config.VersionLatest
		emptyConfig := &config.Config{
			Version: version,
		}
		return emptyConfig.ToProto()
	}

	// Convert domain config to protobuf
	pbConfig := cfg.ToProto()

	// Use proto.Clone to ensure we're returning a deep copy
	// This is an extra safety measure to avoid potential modification
	return proto.Clone(pbConfig).(*pb.ServerConfig)
}

// GetDomainConfig returns a copy of the current domain config by value
func (r *Runner) GetDomainConfig() config.Config {
	r.configMu.RLock()
	defer r.configMu.RUnlock()

	if r.config == nil {
		// Return a minimal valid config if none exists
		r.logger.Warn("GetDomainConfig: config is nil, returning minimal default")
		return config.Config{
			Version: config.VersionLatest,
		}
	}

	// Debug log the config details
	r.logger.Debug("GetDomainConfig: returning config",
		"listeners", len(r.config.Listeners),
		"endpoints", len(r.config.Endpoints),
		"apps", len(r.config.Apps))

	// Return a copy by value
	return *r.config
}

// GetReloadTrigger implements the supervisor.ReloadSender interface.
// It exposes a channel that receives notifications whenever the configuration
// is updated, allowing systems to react to configuration changes without polling.
func (r *Runner) GetReloadTrigger() <-chan struct{} {
	return r.reloadCh
}

// triggerReload sends a notification to the reload channel
// to inform subscribers that the configuration has changed
func (r *Runner) triggerReload() {
	select {
	case r.reloadCh <- struct{}{}:
		r.logger.Debug("Reload notification sent")
	default:
		r.logger.Warn("Reload notification channel full, skipping notification")
	}
}

// LoadInitialConfig attempts to load configuration from disk.
// This can be called explicitly to load config before starting the runner.
func (r *Runner) LoadInitialConfig() error {
	r.logger.Info("Loading initial configuration", "path", r.configPath)

	// Use the config package's NewConfig which already handles loading and validation
	domainConfig, err := config.NewConfig(r.configPath)
	if err != nil {
		r.logger.Error("Failed to load or validate initial configuration", "error", err)
		return err
	}

	// Log details of the loaded config
	if domainConfig != nil {
		r.logger.Debug("Domain config loaded details",
			"listeners", len(domainConfig.Listeners),
			"endpoints", len(domainConfig.Endpoints),
			"apps", len(domainConfig.Apps))

		// Log first listener if available
		if len(domainConfig.Listeners) > 0 {
			listener := domainConfig.Listeners[0]
			r.logger.Debug("First listener details",
				"id", listener.ID,
				"address", listener.Address,
				"type", listener.Type)
		}
	}

	// Store the domain config directly
	r.configMu.Lock()
	r.config = domainConfig
	r.configMu.Unlock()

	r.logger.Info("Initial configuration loaded successfully")
	return nil
}

// UpdateConfig handles requests to update the configuration via gRPC.
func (r *Runner) UpdateConfig(
	ctx context.Context,
	req *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	r.logger.Info("Received UpdateConfig request")

	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "No configuration provided")
	}

	if err := r.fsm.Transition(finitestate.StatusReloading); err != nil {
		r.logger.Error("Failed to transition to reloading state", "error", err)
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"service not in a state that can accept configuration updates",
		)
	}

	domainConfig, err := config.NewFromProto(req.Config)
	if err != nil {
		if errState := r.fsm.Transition(finitestate.StatusError); errState != nil {
			r.logger.Error("Failed to transition to error state", "error", errState)
		}
		return nil, status.Errorf(codes.InvalidArgument, "conversion error: %v", err)
	}

	if err := domainConfig.Validate(); err != nil {
		r.logger.Warn("Configuration validation failed", "error", err)
		if errState := r.fsm.Transition(finitestate.StatusError); errState != nil {
			r.logger.Error("Failed to transition to error state", "error", errState)
		}
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	r.configMu.Lock()
	r.config = domainConfig
	r.configMu.Unlock()

	// Notify subscribers about the config change
	r.triggerReload()

	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		r.logger.Error("Failed to transition to running state", "error", err)
		if errState := r.fsm.Transition(finitestate.StatusError); errState != nil {
			r.logger.Error("Failed to transition to error state", "error", errState)
		}
		return nil, status.Errorf(
			codes.Internal,
			"configuration updated but service failed to return to running state",
		)
	}

	success := true
	return &pb.UpdateConfigResponse{
		Success: &success,
		Config:  r.GetPbConfigClone(),
	}, nil
}

// GetConfig responds to gRPC requests for the current configuration.
// It returns a deep copy to prevent clients from modifying the server's state.
func (r *Runner) GetConfig(
	ctx context.Context,
	req *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	r.logger.Info("Received GetConfig request")
	return &pb.GetConfigResponse{
		Config: r.GetPbConfigClone(),
	}, nil
}
