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
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice/server"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/robbyt/go-supervisor/supervisor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Interface guard: ensure Runner implements required interfaces
var (
	_ supervisor.Runnable  = (*Runner)(nil)
	_ supervisor.Stateable = (*Runner)(nil)
)

type Runner struct {
	// Embed the UnimplementedConfigServiceServer for gRPC compatibility
	pb.UnimplementedConfigServiceServer

	// listenAddr is the address or socket the gRPC server will listen on
	listenAddr string

	// gRPC server implementation and mutex for updating or accessing
	grpcServer GRPCServer
	grpcLock   sync.RWMutex // TODO: is this still needed?

	// Transaction storage for configuration history
	txStorage configTransactionStorage

	// runCtx is passed in to Run, and is used to cancel the Run loop
	runCtx    context.Context
	runCancel context.CancelFunc
	parentCtx context.Context
	txSiphon  chan<- *transaction.ConfigTransaction
	fsm       finitestate.Machine
	logger    *slog.Logger
}

// NewRunner creates a new Runner instance with required listenAddr and transaction siphon.
func NewRunner(
	listenAddr string,
	txSiphon chan<- *transaction.ConfigTransaction,
	opts ...Option,
) (*Runner, error) {
	if listenAddr == "" {
		return nil, errors.New("listen address cannot be empty")
	}
	if txSiphon == nil {
		return nil, errors.New("transaction siphon cannot be nil")
	}

	r := &Runner{
		listenAddr: listenAddr,
		txSiphon:   txSiphon,
		parentCtx:  context.Background(),
		logger:     slog.Default().WithGroup("cfgservice.Runner"),
	}

	// Initialize the finite state machine
	fsmLogger := r.logger.WithGroup("fsm")
	fsm, err := finitestate.New(fsmLogger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}
	r.fsm = fsm

	// Apply functional options
	for _, opt := range opts {
		opt(r)
	}

	// Initialize transaction storage if not provided
	if r.txStorage == nil {
		r.logger.Warn("no transaction storage provided, creating a local in-memory storage")
		r.txStorage = txstorage.NewMemoryStorage()
	}

	return r, nil
}

func (r *Runner) String() string {
	return "cfgservice.Runner"
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

	r.runCtx, r.runCancel = context.WithCancel(ctx)

	r.grpcLock.RLock()
	grpcServer := r.grpcServer
	r.grpcLock.RUnlock()
	if grpcServer != nil {
		if err := r.fsm.Transition(finitestate.StatusError); err != nil {
			return fmt.Errorf("failed to transition to error state: %w", err)
		}
		return errors.New("gRPC server is already running")
	}

	// Start gRPC server (listenAddr is always provided now)
	var err error
	grpcServer, err = server.NewGRPCManager(r.logger, r.listenAddr, r)
	if err != nil {
		if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
			return fmt.Errorf("failed to transition to error state: %w", stateErr)
		}
		return err
	}

	// lock before starting the server to make sure that Stop isn't being called while we're starting
	// which would cause a listener conflict
	r.grpcLock.Lock()
	if err = grpcServer.Start(ctx); err != nil {
		r.grpcLock.Unlock()
		if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
			return fmt.Errorf("failed to transition to error state: %w", stateErr)
		}
		return err
	}
	// store the started server, for graceful shutdown later
	r.grpcServer = grpcServer
	r.grpcLock.Unlock()

	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// block here waiting for a context cancellation
	select {
	case <-r.runCtx.Done():
		// context was canceled, so we're done
	case <-r.parentCtx.Done():
		// parent context was canceled, so we're done
	}

	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		return fmt.Errorf("failed to transition to stopping state: %w", err)
	}

	// Stop the gRPC server if it's available
	r.logger.Debug("Stopping gRPC server")
	r.grpcLock.Lock()
	if r.grpcServer != nil {
		r.grpcServer.GracefulStop()
		r.grpcServer = nil
		r.logger.Info("gRPC server stopped", "listenAddr", r.listenAddr)
	}
	r.grpcLock.Unlock()

	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		r.logger.Error("Failed to transition to stopped state", "error", err)
	}

	r.logger.Debug("Runner stopped")
	return nil
}

// Stop gracefully shuts down the gRPC server if one is running.
// A lock is held during the entire shutdown to prevent concurrent modifications
// to the configuration while the server is shutting down.
func (r *Runner) Stop() {
	r.logger.Debug("Stopping Runner")

	// Cancel the context and let Run() handle the state transitions
	if r.runCancel != nil {
		r.runCancel()
	}
}

// GetPbConfigClone returns the current domain config converted to a protobuf message.
func (r *Runner) GetPbConfigClone() *pb.ServerConfig {
	cfg := r.GetDomainConfig()
	pbConfig := cfg.ToProto()
	return proto.Clone(pbConfig).(*pb.ServerConfig)
}

// GetDomainConfig returns a copy of the current domain config by value
func (r *Runner) GetDomainConfig() config.Config {
	cfgTx := r.txStorage.GetCurrent()
	if cfgTx == nil {
		// Return a minimal valid config if none exists
		r.logger.Warn("txStorage.GetCurrent() returned nil, returning minimal default")
		return config.Config{
			Version: config.VersionLatest,
		}
	}

	cfg := cfgTx.GetConfig()
	if cfg == nil {
		// Return a minimal valid config if none exists
		r.logger.Warn("txStorage.GetCurrent().GetConfig() returned nil, returning minimal default")
		return config.Config{
			Version: config.VersionLatest,
		}
	}

	r.logger.Debug(
		"GetDomainConfig: returning config",
		"listeners", len(cfg.Listeners),
		"endpoints", len(cfg.Endpoints),
		"apps", len(cfg.Apps))

	return *cfg
}

// createAPITransaction creates a new transaction from an API request.
func (r *Runner) createAPITransaction(
	ctx context.Context,
	cfg *config.Config,
) (*transaction.ConfigTransaction, error) {
	requestID := server.ExtractRequestID(ctx)
	return transaction.FromAPI(requestID, cfg, r.logger.Handler())
}

// UpdateConfig handles requests to update the configuration via gRPC.
func (r *Runner) UpdateConfig(
	ctx context.Context,
	req *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	logger := r.logger.With("request_id", server.ExtractRequestID(ctx), "service", "UpdateConfig")
	logger.Info("Received UpdateConfig request")

	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "No configuration provided")
	}

	// Convert protobuf to domain config
	domainConfig, err := config.NewFromProto(req.Config)
	if err != nil {
		// Return a failed response with the submitted config
		logger.Warn("Failed to convert protobuf to domain config", "error", err)
		success := false
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   proto.String(fmt.Sprintf("conversion error: %v", err)),
			Config:  req.Config, // Return the invalid submitted config to help with corrections
		}, nil
	}

	// Create a transaction for this API request
	tx, err := r.createAPITransaction(ctx, domainConfig)
	if err != nil {
		logger.Warn("Failed to create config transaction", "error", err)
		success := false
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   proto.String(fmt.Sprintf("transaction creation failed: %v", err)),
			Config:  req.Config, // Return the invalid submitted config
		}, nil
	}

	// Validate the transaction (but don't orchestrate it)
	if err := tx.RunValidation(); err != nil {
		logger.Warn("Failed to validate config transaction", "error", err)
		success := false
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   proto.String(fmt.Sprintf("transaction validation failed: %v", err)),
			Config:  req.Config, // Return the invalid submitted config
		}, nil
	}

	// Send the validated transaction to the siphon
	select {
	case r.txSiphon <- tx:
		logger.Debug("Transaction sent to siphon", "id", tx.ID)
	case <-ctx.Done():
		logger.Warn("Context cancelled while sending transaction", "id", tx.ID)
		success := false
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   proto.String("context cancelled"),
			Config:  req.Config,
		}, nil
	case <-r.parentCtx.Done():
		logger.Warn("Parent context cancelled while sending transaction", "id", tx.ID)
		success := false
		return &pb.UpdateConfigResponse{
			Success: &success,
			Error:   proto.String("service shutting down"),
			Config:  req.Config,
		}, nil
	}

	logger.Debug("Config updated successfully", "request_id", server.ExtractRequestID(ctx))
	success := true
	return &pb.UpdateConfigResponse{
		Success: &success,
		Config:  tx.GetConfig().ToProto(), // convert back to pb to get defaults
	}, nil
}

// GetConfig responds to gRPC requests for the current configuration.
// It returns a deep copy to prevent clients from modifying the server's state.
func (r *Runner) GetConfig(
	ctx context.Context,
	req *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	r.logger.Debug(
		"Received request",
		"request_id", server.ExtractRequestID(ctx),
		"service", "GetConfig",
	)
	return &pb.GetConfigResponse{
		Config: r.GetPbConfigClone(),
	}, nil
}
