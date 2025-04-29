package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/grpc"
)

// listener defines the interface needed for a network listener
type listener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}

// GRPCManager implements GRPCManager interface for the gRPC server
type GRPCManager struct {
	logger     *slog.Logger
	listenAddr string
	grpcServer *grpc.Server
	listener   listener
}

// NewGRPCManager creates a new gRPC manager instance, which configures a gRPC server.
// It parses the listen address, cleans up existing Unix sockets if necessary,
// creates a listener, and configures the gRPC server, and abstracts the underlying
// Start() and GracefulStop() on the actual gRPC server instance.
func NewGRPCManager(
	logger *slog.Logger,
	listenAddr string,
	pbCfgService pb.ConfigServiceServer,
) (*GRPCManager, error) {
	logger.Debug("Creating gRPC server", "requested_address", listenAddr)

	// 1. Parse network and address
	network, address, err := parseListenAddr(listenAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing listen address %q: %w", listenAddr, err)
	}
	logger.Debug("Parsed listen address", "network", network, "address", address)

	// 2. Clean up Unix socket if applicable
	if network == "unix" {
		if err := cleanupUnixSocket(address, logger); err != nil {
			return nil, fmt.Errorf("pre-listen cleanup of unix socket %q failed: %w", address, err)
		}
	}

	// 3. Create listener
	logger.Debug("Attempting to listen", "network", network, "address", address)
	lis, err := net.Listen(network, address)
	if err != nil {
		logger.Error("Failed to listen", "network", network, "address", address, "error", err)
		return nil, fmt.Errorf("failed to listen on %s://%s: %w", network, address, err)
	}

	// 4. Create and register gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterConfigServiceServer(grpcServer, pbCfgService)

	// Log the actual listening address (useful for TCP port 0)
	actualAddr := lis.Addr()
	logger.Debug(
		"Successfully configured gRPC server",
		"network", actualAddr.Network(),
		"address", actualAddr.String(),
	)

	// 5. Return the server instance
	return &GRPCManager{
		logger:     logger,
		listenAddr: listenAddr,
		grpcServer: grpcServer,
		listener:   lis,
	}, nil
}

// Start implements the GRPCServer interface and begins serving gRPC requests (non blocking)
func (s *GRPCManager) Start(ctx context.Context) error {
	startupErr := make(chan error, 1)
	actualAddr := s.listener.Addr()

	// Start the server in a goroutine
	go func() {
		s.logger.Info("gRPC server starting...", "address", actualAddr.String())
		if err := s.grpcServer.Serve(s.listener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			s.logger.Error("gRPC server encountered an error", "error", err)
			startupErr <- fmt.Errorf("gRPC server error: %w", err)
			return
		}
		s.logger.Debug("gRPC server stopped gracefully")
	}()

	// Wait for the server to start or fail
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	select {
	case err := <-startupErr:
		s.logger.Error("gRPC server failed to start", "error", err)
		closeErr := s.listener.Close()
		if closeErr != nil {
			s.logger.Error("Failed to close listener after gRPC server error", "error", closeErr)
		}
		return fmt.Errorf("gRPC server startup error: %w", err)
	case <-ctx.Done():
		// Context timeout reached or context canceled, server is assumed to be starting
		if ctx.Err() == context.Canceled {
			s.logger.Info("gRPC server start canceled by context")
			s.GracefulStop()
			return ctx.Err()
		}
	}

	return nil
}

// GracefulStop implements the GRPCServer interface
func (s *GRPCManager) GracefulStop() {
	if s.grpcServer != nil {
		s.logger.Debug("Gracefully stopping gRPC server")
		s.grpcServer.GracefulStop()
	}
}

// GetListenAddress returns the actual address the server is listening on
func (s *GRPCManager) GetListenAddress() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.listenAddr
}
