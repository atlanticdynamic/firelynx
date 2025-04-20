package config_manager

import (
	"errors"
	"log/slog"
	"net"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/grpc"
)

// GRPCServer defines the interface for a GRPC server that can be gracefully stopped
type GRPCServer interface {
	GracefulStop()
}

// StartGRPCServerFunc is the function type that starts a gRPC server
type StartGRPCServerFunc func(
	logger *slog.Logger,
	listenAddr string,
	server pb.ConfigServiceServer,
) (GRPCServer, error)

// DefaultStartGRPCServer is the default implementation for starting a gRPC server
func DefaultStartGRPCServer(
	logger *slog.Logger,
	listenAddr string,
	server pb.ConfigServiceServer,
) (GRPCServer, error) {
	logger.Info("Starting gRPC server", "address", listenAddr)

	// Parse the listen address to determine network type (tcp or unix socket)
	// This is a simplified implementation
	network := "tcp"
	address := listenAddr

	// TODO: Parse network and address from listenAddr

	// Create listener
	lis, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register the ConfigService
	pb.RegisterConfigServiceServer(grpcServer, server)

	// Start gRPC server in a separate goroutine
	go func() {
		logger.Info("gRPC server listening", "address", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("gRPC server error", "error", err)
		}
	}()

	return grpcServer, nil
}
