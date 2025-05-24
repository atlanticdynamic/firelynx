package server

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListenAddr(t *testing.T) {
	tests := []struct {
		name           string
		listenAddr     string
		expectedNet    string
		expectedAddr   string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "empty string",
			listenAddr:   "",
			expectedNet:  "tcp",
			expectedAddr: "",
			expectError:  false,
		},
		{
			name:         "tcp address with port",
			listenAddr:   "localhost:8080",
			expectedNet:  "tcp",
			expectedAddr: "localhost:8080",
			expectError:  false,
		},
		{
			name:         "tcp address with only port",
			listenAddr:   ":8080",
			expectedNet:  "tcp",
			expectedAddr: ":8080",
			expectError:  false,
		},
		{
			name:         "unix socket address",
			listenAddr:   "unix:/tmp/test.sock",
			expectedNet:  "unix",
			expectedAddr: "/tmp/test.sock",
			expectError:  false,
		},
		{
			name:           "invalid unix socket (empty path)",
			listenAddr:     "unix:",
			expectedNet:    "",
			expectedAddr:   "",
			expectError:    true,
			expectedErrMsg: "invalid unix socket address: path cannot be empty after 'unix:' prefix",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			network, address, err := parseListenAddr(tc.listenAddr)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedNet, network)
				assert.Equal(t, tc.expectedAddr, address)
			}
		})
	}
}

func TestCleanupUnixSocket(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		socketSetup   func(path string) error
		socketPath    string
		expectError   bool
		checkExistsAt string
	}{
		{
			name:        "non-existent socket",
			socketPath:  filepath.Join(tempDir, "nonexistent.sock"),
			expectError: false,
		},
		{
			name: "existing socket file",
			socketSetup: func(path string) error {
				// Create a dummy socket file
				f, err := os.Create(path)
				if err != nil {
					return err
				}
				return f.Close()
			},
			socketPath:  filepath.Join(tempDir, "existing.sock"),
			expectError: false,
		},
		{
			name: "socket path is a directory",
			socketSetup: func(path string) error {
				return os.Mkdir(path, 0o755)
			},
			socketPath:  filepath.Join(tempDir, "dir.sock"),
			expectError: false, // The current implementation doesn't return an error for directories
			// On real systems, this will fail at the later net.Listen step
		},
		{
			name:        "parent directory doesn't exist",
			socketPath:  filepath.Join(tempDir, "nonexistent-dir", "socket.sock"),
			expectError: false, // No error because we're only checking if the file exists
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the socket if needed
			if tc.socketSetup != nil {
				err := tc.socketSetup(tc.socketPath)
				require.NoError(t, err, "Failed to set up test socket")
			}

			// Run cleanupUnixSocket
			err := cleanupUnixSocket(tc.socketPath, logger)

			// Check results
			if tc.expectError {
				assert.Error(t, err)
				// If we expect the socket to still exist, verify it
				if tc.checkExistsAt != "" {
					_, statErr := os.Stat(tc.checkExistsAt)
					assert.NoError(
						t,
						statErr,
						"Expected socket to still exist at %s",
						tc.checkExistsAt,
					)
				}
			} else {
				assert.NoError(t, err)
				// Verify socket is gone
				_, statErr := os.Stat(tc.socketPath)
				assert.True(t, os.IsNotExist(statErr), "Expected socket to be removed at %s", tc.socketPath)
			}
		})
	}
}
