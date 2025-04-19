package config_manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/grpc"
)

// ConfigManager implements the configuration management functionality
// and serves as a gRPC server for configuration updates.
// It implements the supervisor.Runnable interface for lifecycle management.

// Interface guard: ensure ConfigManager implements supervisor.Runnable
var _ supervisor.Runnable = (*ConfigManager)(nil)

type ConfigManager struct {
	pb.UnimplementedConfigServiceServer

	logger   *slog.Logger
	config   *pb.ServerConfig
	configMu sync.RWMutex

	// gRPC related fields
	grpcServer *grpc.Server
	listener   net.Listener
	listenAddr string

	// Initial config path
	configPath string

	// For reload notifications
	reloadCh chan struct{}
}

// Config for creating a new ConfigManager
type Config struct {
	Logger     *slog.Logger
	ListenAddr string
	ConfigPath string
}

// New creates a new ConfigManager instance
func New(cfg Config) *ConfigManager {
	return &ConfigManager{
		logger:     cfg.Logger,
		listenAddr: cfg.ListenAddr,
		configPath: cfg.ConfigPath,
		reloadCh:   make(chan struct{}, 1), // Buffer of 1 to avoid blocking
	}
}

// Run implements the Runnable interface and starts the ConfigManager
func (cm *ConfigManager) Run(ctx context.Context) error {
	cm.logger.Info("Starting ConfigManager")

	// Initialize with at least an empty config
	cm.configMu.Lock()
	version := config.VersionLatest
	cm.config = &pb.ServerConfig{
		Version: &version,
	}
	cm.configMu.Unlock()

	// Load initial configuration if path is provided
	if cm.configPath != "" {
		if err := cm.loadInitialConfig(); err != nil {
			cm.logger.Error("Failed to load initial configuration", "error", err)
			// Continue with the empty config - don't fail
		}
	}

	// Start gRPC server if listen address is provided
	if cm.listenAddr != "" {
		if err := cm.startGRPCServer(); err != nil {
			return err
		}
	}

	// Block until context is done
	<-ctx.Done()
	cm.logger.Info("ConfigManager shutting down")

	return nil
}

// Stop implements the Runnable interface and stops the ConfigManager
func (cm *ConfigManager) Stop() {
	cm.logger.Info("Stopping ConfigManager")

	// Stop gRPC server
	if cm.grpcServer != nil {
		cm.grpcServer.GracefulStop()
		cm.logger.Info("gRPC server stopped")
	}
	// When used with the supervisor, the supervisor will cancel the context
	// passed to Run, which will cause Run to return
}

// GetCurrentConfig returns the current configuration (this is different from the gRPC GetConfig method)
func (cm *ConfigManager) GetCurrentConfig() *pb.ServerConfig {
	cm.configMu.RLock()
	defer cm.configMu.RUnlock()

	// Return a copy of the config to avoid concurrent modification issues
	// For now, we'll return the actual instance, but in a real implementation
	// we would create a deep copy of the config
	return cm.config
}

// GetReloadChannel returns a channel that will be notified when a reload is triggered
func (cm *ConfigManager) GetReloadChannel() <-chan struct{} {
	return cm.reloadCh
}

// loadInitialConfig loads the initial configuration from the provided path
func (cm *ConfigManager) loadInitialConfig() error {
	cm.logger.Info("Loading initial configuration", "path", cm.configPath)

	// Use the loader package to load the configuration
	configLoader, err := loader.NewLoaderFromFilePath(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config loader: %w", err)
	}

	// Parse the configuration
	config, err := configLoader.LoadProto()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update the configuration
	cm.configMu.Lock()
	cm.config = config
	cm.configMu.Unlock()

	cm.logger.Info("Initial configuration loaded successfully")
	return nil
}

// startGRPCServer starts the gRPC server for configuration updates
func (cm *ConfigManager) startGRPCServer() error {
	cm.logger.Info("Starting gRPC server", "address", cm.listenAddr)

	// Parse the listen address to determine network type (tcp or unix socket)
	// This is a simplified implementation
	network := "tcp"
	address := cm.listenAddr

	// TODO: Parse network and address from listenAddr

	// Create listener
	lis, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	cm.listener = lis

	// Create gRPC server
	cm.grpcServer = grpc.NewServer()

	// Register the ConfigService
	pb.RegisterConfigServiceServer(cm.grpcServer, cm)

	// Start gRPC server in a separate goroutine
	go func() {
		cm.logger.Info("gRPC server listening", "address", cm.listener.Addr())
		if err := cm.grpcServer.Serve(cm.listener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			cm.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

// UpdateConfig implements the ConfigService UpdateConfig RPC method
func (cm *ConfigManager) UpdateConfig(
	ctx context.Context,
	req *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	cm.logger.Info("Received UpdateConfig request")

	if req.Config == nil {
		success := false
		errorMessage := "No configuration provided"
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   &errorMessage,
		}, nil
	}

	// TODO: Validate configuration

	// Update the configuration
	cm.configMu.Lock()
	cm.config = req.Config
	cm.configMu.Unlock()

	// Trigger a reload notification
	select {
	case cm.reloadCh <- struct{}{}:
		cm.logger.Info("Reload notification sent")
	default:
		cm.logger.Info("Reload notification channel full, skipping")
	}

	success := true
	return &pb.UpdateConfigResponse{
		Success: &success,
		Config:  cm.GetCurrentConfig(),
	}, nil
}

// GetConfig implements the ConfigService GetConfig RPC method
func (cm *ConfigManager) GetConfig(
	ctx context.Context,
	req *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	cm.logger.Info("Received GetConfig request")

	return &pb.GetConfigResponse{
		Config: cm.GetCurrentConfig(),
	}, nil
}
