package cfgrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/grpc"
)

// GRPCServer defines the interface for a GRPC server that can be gracefully stopped
type GRPCServer interface {
	GracefulStop()
}

// StartGRPCServerFunc is the function signature type for starting a gRPC server
type StartGRPCServerFunc func(
	logger *slog.Logger,
	listenAddr string,
	server pb.ConfigServiceServer,
) (GRPCServer, error)

// DefaultStartGRPCServer is the default implementation for starting a gRPC server.
// It parses the listen address, cleans up existing Unix sockets if necessary,
// creates a listener, and starts the gRPC server.
func DefaultStartGRPCServer(
	logger *slog.Logger,
	listenAddr string,
	server pb.ConfigServiceServer,
) (GRPCServer, error) {
	logger.Debug("Starting gRPC server", "requested_address", listenAddr)

	// 1. Parse network and address using the simplified function
	network, address, err := parseListenAddr(listenAddr)
	if err != nil {
		logger.Error("Failed to parse listen address", "address", listenAddr, "error", err)
		return nil, fmt.Errorf("parsing listen address %q: %w", listenAddr, err)
	}
	logger.Debug("Parsed listen address", "network", network, "address", address)

	// 2. Clean up Unix socket *before* listening, if applicable
	if network == "unix" {
		if err := cleanupUnixSocket(address, logger); err != nil {
			// Failing here prevents attempting to listen on a potentially problematic path
			logger.Error(
				"Failed to clean up unix socket before listening",
				"path", address,
				"error", err,
			)
			return nil, fmt.Errorf("pre-listen cleanup of unix socket %q failed: %w", address, err)
		}
	}

	// 3. Create listener
	logger.Debug("Attempting to listen", "network", network, "address", address)
	lis, err := net.Listen(network, address)
	if err != nil {
		logger.Error("Failed to listen", "network", network, "address", address, "error", err)
		return nil, fmt.Errorf("listening on %s://%s: %w", network, address, err)
	}

	// Log the actual listening address (useful for TCP port 0)
	actualAddr := lis.Addr()
	logger.Debug(
		"Successfully listening",
		"network", actualAddr.Network(),
		"address", actualAddr.String(),
	)

	// 4. Create and register gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterConfigServiceServer(grpcServer, server)

	// 5. Start gRPC server in a separate goroutine
	startupErr := make(chan error, 1)
	go func() {
		logger.Info("gRPC server starting...", "address", actualAddr.String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("gRPC server encountered an error", "error", err)
			startupErr <- fmt.Errorf("gRPC server error: %w", err)
			return
		}
		logger.Debug("gRPC server stopped gracefully")
	}()

	// 6. Wait for the server to start or fail
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	select {
	case err := <-startupErr:
		logger.Error("gRPC server failed to start", "error", err)
		err = lis.Close()
		if err != nil {
			logger.Error("Failed to close listener after gRPC server error", "error", err)
		}
		return nil, fmt.Errorf("gRPC server startup error: %w", err)
	case <-ctx.Done():
		// Context timeout reached, server is assumed to be starting
	}

	return grpcServer, nil
}

// parseListenAddr determines the network type and address from a listen string.
// It requires "unix:/path/to/socket.sock" for Unix sockets.
// All other non-empty strings are assumed to be TCP addresses ("host:port", ":port", etc.).
// An empty string is also treated as TCP (likely defaulting to ":0").
func parseListenAddr(listenAddr string) (network string, address string, err error) {
	if strings.HasPrefix(listenAddr, "unix:") {
		network = "unix"
		address = strings.TrimPrefix(listenAddr, "unix:")
		if address == "" {
			// Handle the case like "unix:" which is invalid.
			return "", "", fmt.Errorf(
				"invalid unix socket address: path cannot be empty after 'unix:' prefix",
			)
		}
	} else {
		// Assume TCP for everything else. Let net.Listen handle validation like format checks.
		network = "tcp"
		address = listenAddr
	}
	return network, address, nil
}

// cleanupUnixSocket removes the specified file path if it exists.
// It's intended for cleaning up stale Unix domain sockets before listening.
func cleanupUnixSocket(socketPath string, logger *slog.Logger) error {
	logger.Debug("Checking for existing unix socket", "path", socketPath)
	// Use Lstat to avoid following symlinks, check if the file exists.
	if _, err := os.Lstat(socketPath); err == nil {
		logger.Warn("Removing existing unix socket", "path", socketPath)
		if removeErr := os.Remove(socketPath); removeErr != nil {
			// Check if the error is because it's actually a directory or something else non-removable
			if os.IsExist(removeErr) ||
				os.IsPermission(removeErr) { // Check common persistent errors
				return fmt.Errorf(
					"failed to remove existing file/socket (possible permission issue or directory conflict) at %q: %w",
					socketPath,
					removeErr,
				)
			}
			// Return error if removal fails for other reasons
			return fmt.Errorf("failed to remove existing unix socket %q: %w", socketPath, removeErr)
		}
		logger.Warn("Successfully removed existing unix socket", "path", socketPath)
		return nil // Successfully removed
	} else if !os.IsNotExist(err) {
		// Stat failed for a reason other than the file not existing (e.g., permissions on parent dir)
		return fmt.Errorf("failed to stat potential unix socket %q: %w", socketPath, err)
	}
	// File does not exist, no cleanup needed.
	logger.Debug("No existing unix socket found, no cleanup needed", "path", socketPath)
	return nil
}
