package cfgservice

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// GRPCServer defines the interface for a GRPC server that can be started and stopped
type GRPCServer interface {
	// Start begins the gRPC server and returns immediately.
	// It will wait up to the server start timeout to confirm the server has started.
	// The provided context can be used to cancel server startup.
	Start(ctx context.Context) error

	// GracefulStop stops the gRPC server gracefully
	GracefulStop()

	// GetListenAddress returns the actual address the server is listening on
	GetListenAddress() string
}

// TransactionStorage defines the interface for retrieving configuration transactions
type transactionStorage interface {
	// GetCurrent returns the current active transaction
	GetCurrent() *transaction.ConfigTransaction
}
