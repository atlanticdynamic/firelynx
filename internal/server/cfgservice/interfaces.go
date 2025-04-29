package cfgservice

import "context"

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
