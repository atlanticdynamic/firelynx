// Runner manages configuration state and serves a gRPC API for clients to retrieve
// and update the configuration. It integrates with the supervisor package for
// lifecycle management, and implements the ReloadSender interface to allow
// subscribers to detect configuration changes.
package cfgservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/cfgservice/server"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Interface guard: ensure Runner implements supervisor.Runnable
var (
	_ supervisor.Runnable     = (*Runner)(nil)
	_ supervisor.ReloadSender = (*Runner)(nil)
)

type Runner struct {
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
}

// New creates a new Runner instance with the provided functional options.
func New(opts ...Option) (*Runner, error) {
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

	r.grpcLock.RLock()
	grpcServer := r.grpcServer
	r.grpcLock.RUnlock()
	if grpcServer != nil {
		return errors.New("gRPC server is already running")
	}

	// Initialize with at least an empty config
	r.configMu.Lock()
	r.config = &config.Config{
		Version: config.VersionLatest,
	}
	r.configMu.Unlock()

	// Load initial configuration if path is provided
	if r.configPath != "" {
		if err := r.loadInitialConfig(); err != nil {
			r.logger.Error("Failed to load initial configuration", "error", err)
			// If we don't have a listen address, fail immediately
			if r.listenAddr == "" {
				return err
			}
			// Otherwise continue with the empty config
		}
	}

	// Start gRPC server if listen address is provided
	if r.listenAddr != "" {
		var err error
		grpcServer, err = server.NewGRPCManager(r.logger, r.listenAddr, r)
		if err != nil {
			return err
		}

		// lock before starting the server to make sure that Stop isn't being called while we're starting
		// which would cause a listener conflict
		r.grpcLock.Lock()
		if err = grpcServer.Start(ctx); err != nil {
			r.grpcLock.Unlock()
			return err
		}
		// store the started server, for graceful shutdown later
		r.grpcServer = grpcServer
		r.grpcLock.Unlock()
	}

	// Block until context is done
	<-ctx.Done()
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
	// When used with the supervisor, the supervisor will cancel the context
	// passed to Run, which will cause Run to return
	// TODO: add local context
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
		return config.Config{
			Version: config.VersionLatest,
		}
	}

	// Return a copy by value
	return *r.config
}

// GetReloadTrigger implements the supervisor.ReloadSender interface.
// It exposes a channel that receives notifications whenever the configuration
// is updated, allowing systems to react to configuration changes without polling.
func (r *Runner) GetReloadTrigger() <-chan struct{} {
	return r.reloadCh
}

// loadInitialConfig attempts to load configuration from disk at startup.
func (r *Runner) loadInitialConfig() error {
	r.logger.Info("Loading initial configuration", "path", r.configPath)

	// Use the config package's NewConfig which already handles loading and validation
	domainConfig, err := config.NewConfig(r.configPath)
	if err != nil {
		r.logger.Error("Failed to load or validate initial configuration", "error", err)
		return err
	}

	// Store the domain config directly
	r.configMu.Lock()
	r.config = domainConfig
	r.configMu.Unlock()

	r.logger.Info("Initial configuration loaded successfully")
	return nil
}

// UpdateConfig handles requests to update the configuration via gRPC.
// It performs validation in domain model space before accepting the update.
func (r *Runner) UpdateConfig(
	ctx context.Context,
	req *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	r.logger.Info("Received UpdateConfig request")

	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "No configuration provided")
	}

	// Validate the configuration by converting to domain model and validating
	domainConfig, err := config.NewFromProto(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "conversion error: %v", err)
	}

	if err := domainConfig.Validate(); err != nil {
		r.logger.Warn("Configuration validation failed", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	// Update the configuration with the validated domain config
	r.configMu.Lock()
	r.config = domainConfig
	r.configMu.Unlock()

	// Trigger a reload notification
	select {
	case r.reloadCh <- struct{}{}:
		r.logger.Info("Reload notification sent")
	default:
		// TODO: consider removing the default case to provide back pressure instead of dropping reload triggers
		r.logger.Warn("Reload notification channel full, skipping")
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
