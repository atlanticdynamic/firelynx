package cfgrpc

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Runner implements the configuration management functionality
// and serves as a gRPC server for configuration updates.
// It implements the supervisor.Runnable interface for lifecycle management.

// Interface guard: ensure Runner implements supervisor.Runnable
var (
	_ supervisor.Runnable     = (*Runner)(nil)
	_ supervisor.ReloadSender = (*Runner)(nil)
)

type Runner struct {
	pb.UnimplementedConfigServiceServer

	logger   *slog.Logger
	config   *pb.ServerConfig
	configMu sync.RWMutex

	// gRPC server stuff
	grpcServer GRPCServer
	listenAddr string

	// Function to start gRPC server, can be replaced for testing
	startGRPCServer StartGRPCServerFunc

	// Initial config path
	configPath string

	// For triggering a reload when a new config is received
	reloadCh chan struct{}
}

// New creates a new Runner instance with the provided options
func New(opts ...Option) (*Runner, error) {
	r := &Runner{
		logger:          slog.Default(),
		reloadCh:        make(chan struct{}, 1),
		startGRPCServer: DefaultStartGRPCServer,
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

func (cm *Runner) String() string {
	return "cfgrpc.Runner"
}

// Run implements the Runnable interface and starts the Runner
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting Runner")

	// Initialize with at least an empty config
	r.configMu.Lock()
	version := config.VersionLatest
	r.config = &pb.ServerConfig{
		Version: &version,
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
		r.grpcServer, err = r.startGRPCServer(r.logger, r.listenAddr, r)
		if err != nil {
			return err
		}
	}

	// Block until context is done
	<-ctx.Done()
	r.logger.Info("Runner shutting down")

	return nil
}

// Stop implements the Runnable interface and stops the Runner
func (r *Runner) Stop() {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	r.logger.Debug("Stopping Runner")

	// Stop gRPC server
	if r.grpcServer != nil {
		r.grpcServer.GracefulStop()
		r.logger.Info("gRPC server stopped")
	}
	// When used with the supervisor, the supervisor will cancel the context
	// passed to Run, which will cause Run to return
	// TODO: add local context
}

// GetConfigClone returns a deep copy of the current configuration, to avoid external modification.
// (this is different from the gRPC GetConfig method)
func (r *Runner) GetConfigClone() *pb.ServerConfig {
	r.configMu.RLock()
	cfg := r.config
	r.configMu.RUnlock()

	if cfg == nil {
		r.logger.Warn("Current configuration is nil, returning empty config")
		version := config.VersionLatest
		cfg = &pb.ServerConfig{
			Version: &version,
		}
	}

	// Return a copy of the config to avoid
	// concurrent modification issues
	return proto.Clone(cfg).(*pb.ServerConfig)
}

// GetReloadTrigger returns a channel that will be notified when a reload is triggered.
// This implements the supervisor.ReloadSender interface, to trigger a reload when the config is
// received.
func (r *Runner) GetReloadTrigger() <-chan struct{} {
	return r.reloadCh
}

// loadInitialConfig loads the initial configuration from the provided path
func (r *Runner) loadInitialConfig() error {
	r.logger.Info("Loading initial configuration", "path", r.configPath)

	// Use the config package's NewConfig which already handles loading and validation
	domainConfig, err := config.NewConfig(r.configPath)
	if err != nil {
		r.logger.Error("Failed to load or validate initial configuration", "error", err)
		return err
	}

	// Convert to proto and update
	protoConfig := domainConfig.ToProto()

	// Update the configuration
	r.configMu.Lock()
	r.config = protoConfig
	r.configMu.Unlock()

	r.logger.Info("Initial configuration loaded successfully")
	return nil
}

// UpdateConfig implements the ConfigService UpdateConfig RPC method
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

	// Update the configuration with the valid config
	r.configMu.Lock()
	r.config = req.Config
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
		Config:  r.GetConfigClone(),
	}, nil
}

// GetConfig implements the ConfigService GetConfig RPC method
func (r *Runner) GetConfig(
	ctx context.Context,
	req *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	r.logger.Info("Received GetConfig request")
	return &pb.GetConfigResponse{
		Config: r.GetConfigClone(),
	}, nil
}
