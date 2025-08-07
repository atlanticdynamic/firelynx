// Runner manages configuration state and serves a gRPC API for clients to retrieve
// and update the configuration. It integrates with the supervisor package for
// lifecycle management, and implements the ReloadSender interface to allow
// subscribers to detect configuration changes.
package cfgservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

	// ctx is passed in to Run, and is used to cancel the Run loop
	ctx      context.Context
	cancel   context.CancelFunc
	txSiphon chan<- *transaction.ConfigTransaction
	fsm      finitestate.Machine
	logger   *slog.Logger
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

	r.ctx, r.cancel = context.WithCancel(ctx)

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
	<-r.ctx.Done()

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
	if r.cancel != nil {
		r.cancel()
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
		// Use the constructor to ensure Apps is initialized
		minimalCfg, err := config.NewFromProto(&pb.ServerConfig{})
		if err != nil {
			r.logger.Error("Failed to create minimal config", "error", err)
			return config.Config{} // Return zero value as fallback
		}
		return *minimalCfg
	}

	cfg := cfgTx.GetConfig()
	if cfg == nil {
		// Return a minimal valid config if none exists
		r.logger.Warn("txStorage.GetCurrent().GetConfig() returned nil, returning minimal default")
		// Use the constructor to ensure Apps is initialized
		minimalCfg, err := config.NewFromProto(&pb.ServerConfig{})
		if err != nil {
			r.logger.Error("Failed to create minimal config", "error", err)
			return config.Config{} // Return zero value as fallback
		}
		return *minimalCfg
	}

	r.logger.Debug(
		"GetDomainConfig: returning config",
		"listeners", len(cfg.Listeners),
		"endpoints", len(cfg.Endpoints),
		"apps", cfg.Apps.Len())

	return *cfg
}

// pageToken represents the internal structure of a pagination token
type pageToken struct {
	Offset   int    `json:"offset"`
	PageSize int    `json:"pageSize"`
	State    string `json:"state,omitempty"`
	Source   string `json:"source,omitempty"`
}

// encodePageToken creates an opaque page token from pagination parameters
func encodePageToken(offset, pageSize int, state, source string) (string, error) {
	token := pageToken{
		Offset:   offset,
		PageSize: pageSize,
		State:    state,
		Source:   source,
	}

	data, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal page token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// decodePageToken extracts pagination parameters from an opaque token
func decodePageToken(tokenStr string) (pageToken, error) {
	if tokenStr == "" {
		return pageToken{}, nil
	}

	data, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return pageToken{}, fmt.Errorf("invalid page token format: %w", err)
	}

	var token pageToken
	if err := json.Unmarshal(data, &token); err != nil {
		return pageToken{}, fmt.Errorf("failed to unmarshal page token: %w", err)
	}

	return token, nil
}

// createAPITransaction creates a new transaction from an API request.
func (r *Runner) createAPITransaction(
	ctx context.Context,
	cfg *config.Config,
) (*transaction.ConfigTransaction, error) {
	requestID := server.ExtractRequestID(ctx)
	return transaction.FromAPI(requestID, cfg, r.logger.Handler())
}

// ValidateConfig handles requests to validate a configuration via gRPC.
func (r *Runner) ValidateConfig(
	ctx context.Context,
	req *pb.ValidateConfigRequest,
) (*pb.ValidateConfigResponse, error) {
	logger := r.logger.With("request_id", server.ExtractRequestID(ctx), "service", "ValidateConfig")
	logger.Info("Received ValidateConfig request")

	if req.Config == nil {
		return &pb.ValidateConfigResponse{
			Valid: proto.Bool(false),
			Error: proto.String("No configuration provided"),
		}, nil
	}

	// Convert protobuf to domain config
	domainConfig, err := config.NewFromProto(req.Config)
	if err != nil {
		logger.Warn("Failed to convert protobuf to domain config", "error", err)
		return &pb.ValidateConfigResponse{
			Valid: proto.Bool(false),
			Error: proto.String(fmt.Sprintf("conversion error: %v", err)),
		}, nil
	}

	// Validate the configuration directly without creating a transaction
	// This avoids creating transactions that get stuck in non-terminal states during shutdown
	if err := domainConfig.Validate(); err != nil {
		logger.Debug("Configuration validation failed", "error", err)
		return &pb.ValidateConfigResponse{
			Valid: proto.Bool(false),
			Error: proto.String(fmt.Sprintf("validation failed: %v", err)),
		}, nil
	}

	logger.Debug("Config validated successfully", "request_id", server.ExtractRequestID(ctx))
	return &pb.ValidateConfigResponse{
		Valid: proto.Bool(true),
	}, nil
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
			Success:       &success,
			Error:         proto.String(fmt.Sprintf("transaction validation failed: %v", err)),
			Config:        req.Config, // Return the invalid submitted config
			TransactionId: proto.String(tx.ID.String()),
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
			Success:       &success,
			Error:         proto.String("service shutting down"),
			Config:        req.Config,
			TransactionId: proto.String(tx.ID.String()),
		}, nil
	}

	logger.Debug("Config updated successfully", "request_id", server.ExtractRequestID(ctx))
	success := true
	return &pb.UpdateConfigResponse{
		Success:       &success,
		Config:        tx.GetConfig().ToProto(), // convert back to pb to get defaults
		TransactionId: proto.String(tx.ID.String()),
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

// GetCurrentConfigTransaction returns the current active transaction
func (r *Runner) GetCurrentConfigTransaction(
	ctx context.Context,
	req *pb.GetCurrentConfigTransactionRequest,
) (*pb.GetCurrentConfigTransactionResponse, error) {
	r.logger.Debug(
		"Received request",
		"request_id", server.ExtractRequestID(ctx),
		"service", "GetCurrentConfigTransaction",
	)

	currentTx := r.txStorage.GetCurrent()
	return &pb.GetCurrentConfigTransactionResponse{
		Transaction: currentTx.ToProto(),
	}, nil
}

// ListConfigTransactions returns a paginated list of configuration transactions
func (r *Runner) ListConfigTransactions(
	ctx context.Context,
	req *pb.ListConfigTransactionsRequest,
) (*pb.ListConfigTransactionsResponse, error) {
	logger := r.logger.With(
		"request_id",
		server.ExtractRequestID(ctx),
		"service",
		"ListConfigTransactions",
	)
	logger.Debug("Received request")

	// Decode page token to get pagination parameters and filters
	token, err := decodePageToken(req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
	}

	// Use filters from token if available, otherwise from request
	state := req.GetState()
	source := req.GetSource()
	if token.State != "" {
		state = token.State
	}
	if token.Source != "" {
		source = token.Source
	}

	// Validate that filters match between token and request (if both provided)
	if req.GetPageToken() != "" {
		if req.State != nil && *req.State != "" && *req.State != token.State {
			return nil, status.Error(
				codes.InvalidArgument,
				"state filter must match previous request",
			)
		}
		if req.Source != nil && *req.Source != "" && *req.Source != token.Source {
			return nil, status.Error(
				codes.InvalidArgument,
				"source filter must match previous request",
			)
		}
	}

	// Set page size with defaults and limits
	pageSize := int(10)
	if req.PageSize != nil {
		pageSize = int(*req.PageSize)
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Get all transactions and apply filters
	allTxs := r.txStorage.GetAll()
	var filteredTxs []*transaction.ConfigTransaction
	for _, tx := range allTxs {
		// Filter by state if specified
		if state != "" && tx.GetState() != state {
			continue
		}

		// Filter by source if specified
		if source != "" {
			var sourceStr string
			switch tx.Source {
			case transaction.SourceFile:
				sourceStr = "file"
			case transaction.SourceAPI:
				sourceStr = "api"
			case transaction.SourceTest:
				sourceStr = "test"
			default:
				sourceStr = "unspecified"
			}
			if sourceStr != source {
				continue
			}
		}

		filteredTxs = append(filteredTxs, tx)
	}

	// Apply pagination
	offset := token.Offset
	totalCount := len(filteredTxs)

	if offset >= totalCount {
		// Beyond available data
		return &pb.ListConfigTransactionsResponse{
			Transactions:  []*pb.ConfigTransaction{},
			NextPageToken: proto.String(""),
		}, nil
	}

	end := offset + pageSize
	if end > totalCount {
		end = totalCount
	}

	paginatedTxs := filteredTxs[offset:end]

	// Convert to protobuf
	pbTxs := make([]*pb.ConfigTransaction, len(paginatedTxs))
	for i, tx := range paginatedTxs {
		pbTxs[i] = tx.ToProto()
	}

	// Generate next page token if there are more results
	var nextPageToken string
	if end < totalCount {
		nextPageToken, err = encodePageToken(end, pageSize, state, source)
		if err != nil {
			logger.Error("Failed to encode next page token", "error", err)
			return nil, status.Error(codes.Internal, "failed to generate next page token")
		}
	}

	logger.Debug(
		"Returning transactions",
		"total", totalCount,
		"offset", offset,
		"pageSize", pageSize,
		"returned", len(pbTxs),
		"hasNextPage", nextPageToken != "",
	)

	return &pb.ListConfigTransactionsResponse{
		Transactions:  pbTxs,
		NextPageToken: proto.String(nextPageToken),
	}, nil
}

// GetConfigTransaction returns a specific transaction by ID
func (r *Runner) GetConfigTransaction(
	ctx context.Context,
	req *pb.GetConfigTransactionRequest,
) (*pb.GetConfigTransactionResponse, error) {
	logger := r.logger.With(
		"request_id",
		server.ExtractRequestID(ctx),
		"service",
		"GetConfigTransaction",
	)
	logger.Debug("Received request", "transaction_id", req.TransactionId)

	if req.TransactionId == nil || *req.TransactionId == "" {
		return nil, status.Error(codes.InvalidArgument, "transaction_id is required")
	}

	tx := r.txStorage.GetByID(*req.TransactionId)
	if tx == nil {
		return nil, status.Error(codes.NotFound, "transaction not found")
	}

	return &pb.GetConfigTransactionResponse{
		Transaction: tx.ToProto(),
	}, nil
}

// ClearConfigTransactions clears transaction history
func (r *Runner) ClearConfigTransactions(
	ctx context.Context,
	req *pb.ClearConfigTransactionsRequest,
) (*pb.ClearConfigTransactionsResponse, error) {
	logger := r.logger.With(
		"request_id",
		server.ExtractRequestID(ctx),
		"service",
		"ClearConfigTransactions",
	)
	logger.Info("Received clear request", "keep_last", req.KeepLast)

	keepLast := int(0)
	if req.KeepLast != nil {
		keepLast = int(*req.KeepLast)
	}

	cleared, err := r.txStorage.Clear(keepLast)
	if err != nil {
		logger.Error("Failed to clear transactions", "error", err)
		return &pb.ClearConfigTransactionsResponse{
			Success:      proto.Bool(false),
			Error:        proto.String(fmt.Sprintf("failed to clear transactions: %v", err)),
			ClearedCount: proto.Int32(0),
		}, nil
	}

	logger.Info("Cleared transactions", "cleared", cleared)

	return &pb.ClearConfigTransactionsResponse{
		Success:      proto.Bool(true),
		ClearedCount: proto.Int32(int32(cleared)),
	}, nil
}
