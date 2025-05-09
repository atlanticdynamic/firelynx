package server

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

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
