package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/gofrs/uuid/v5"
	"google.golang.org/grpc/metadata"
)

// Sentinel errors for parseListenAddr
var (
	ErrInvalidURLFormat       = errors.New("invalid URL format")
	ErrTCPSchemeRequiresHost  = errors.New("tcp scheme requires host:port after tcp://")
	ErrUnixSchemeRequiresPath = errors.New("unix scheme requires path after unix://")
	ErrUnixColonRequiresPath  = errors.New("unix scheme requires path after unix")
	ErrUnsupportedURLScheme   = errors.New("unsupported URL scheme")
)

// parseListenAddr determines the network type and address from a listen string.
// Supports URL schemes: tcp://, unix://, unix:, or no scheme (defaults to TCP).
// Does not validate file system state - only parses address format.
// Examples:
//   - "tcp://localhost:8080" → network="tcp", address="localhost:8080"
//   - "unix:///tmp/socket" → network="unix", address="/tmp/socket"
//   - "unix:/tmp/socket" → network="unix", address="/tmp/socket"
//   - "localhost:8080" → network="tcp", address="localhost:8080"
//   - "" → network="tcp", address=""
func parseListenAddr(listenAddr string) (network string, address string, err error) {
	// Handle empty string as TCP with default port
	if listenAddr == "" {
		return "tcp", "", nil
	}

	// Handle URL schemes with ://
	if strings.Contains(listenAddr, "://") {
		u, err := url.Parse(listenAddr)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrInvalidURLFormat, err)
		}

		switch u.Scheme {
		case "tcp":
			if u.Host == "" {
				return "", "", ErrTCPSchemeRequiresHost
			}
			return "tcp", u.Host, nil

		case "unix":
			if u.Path == "" {
				return "", "", ErrUnixSchemeRequiresPath
			}
			return "unix", u.Path, nil

		default:
			return "", "", fmt.Errorf(
				"%w: %s (supported: tcp, unix)",
				ErrUnsupportedURLScheme,
				u.Scheme,
			)
		}
	}

	// Handle legacy "unix:" prefix (without //)
	if address, ok := strings.CutPrefix(listenAddr, "unix:"); ok {
		if address == "" {
			return "", "", ErrUnixColonRequiresPath
		}
		return "unix", address, nil
	}

	// No scheme, assume TCP
	return "tcp", listenAddr, nil
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

// ExtractRequestID extracts a request ID from the gRPC context metadata.
// It first tries to find common request ID headers in the metadata.
// If none are found, it generates a new UUID using the V6 format.
func ExtractRequestID(ctx context.Context) string {
	// Try to extract request ID from common metadata keys
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// Check common request ID header variations
		for _, key := range []string{"request-id", "x-request-id", "requestid"} {
			if values := md.Get(key); len(values) > 0 && values[0] != "" {
				return values[0]
			}
		}
	}

	// If no request ID found in metadata, generate a new UUID
	return uuid.Must(uuid.NewV6()).String()
}
