package server

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
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

func TestExtractRequestID(t *testing.T) {
	t.Parallel()

	t.Run("extracts request-id from metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"request-id", "test-request-123",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "test-request-123", requestID)
	})

	t.Run("extracts x-request-id from metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"x-request-id", "test-xrequest-456",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "test-xrequest-456", requestID)
	})

	t.Run("extracts requestid from metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"requestid", "test-requestid-789",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "test-requestid-789", requestID)
	})

	t.Run("prefers request-id over other headers", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"request-id", "preferred-id",
			"x-request-id", "not-this-one",
			"requestid", "not-this-either",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "preferred-id", requestID)
	})

	t.Run("prefers x-request-id over requestid when request-id missing", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"x-request-id", "x-preferred",
			"requestid", "not-this-one",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "x-preferred", requestID)
	})

	t.Run("generates UUID when no request ID headers present", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"some-other-header", "value",
		))

		requestID := ExtractRequestID(ctx)
		assert.NotEmpty(t, requestID)
		// UUID v6 format check (8-4-4-4-12 pattern)
		assert.Regexp(
			t,
			`^[0-9a-f]{8}-[0-9a-f]{4}-6[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$`,
			requestID,
		)
	})

	t.Run("generates UUID when context has no metadata", func(t *testing.T) {
		ctx := context.Background()

		requestID := ExtractRequestID(ctx)
		assert.NotEmpty(t, requestID)
		// UUID v6 format check
		assert.Regexp(
			t,
			`^[0-9a-f]{8}-[0-9a-f]{4}-6[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$`,
			requestID,
		)
	})

	t.Run("ignores empty request ID values and generates UUID", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"request-id", "",
			"x-request-id", "",
			"requestid", "",
		))

		requestID := ExtractRequestID(ctx)
		assert.NotEmpty(t, requestID)
		// Should generate UUID since all values are empty
		assert.Regexp(
			t,
			`^[0-9a-f]{8}-[0-9a-f]{4}-6[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$`,
			requestID,
		)
	})

	t.Run("handles multiple values for same header by using first", func(t *testing.T) {
		md := metadata.New(nil)
		md.Append("request-id", "first-value", "second-value", "third-value")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "first-value", requestID)
	})

	t.Run("handles case variations in metadata keys", func(t *testing.T) {
		// gRPC metadata keys are case-insensitive and normalized to lowercase
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"Request-ID", "uppercase-request-id",
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, "uppercase-request-id", requestID)
	})

	t.Run("generates different UUIDs on each call without metadata", func(t *testing.T) {
		ctx := context.Background()

		requestID1 := ExtractRequestID(ctx)
		requestID2 := ExtractRequestID(ctx)

		assert.NotEmpty(t, requestID1)
		assert.NotEmpty(t, requestID2)
		assert.NotEqual(t, requestID1, requestID2)
	})

	t.Run("preserves special characters in request ID", func(t *testing.T) {
		specialID := "test-123_ABC.def~xyz"
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"request-id", specialID,
		))

		requestID := ExtractRequestID(ctx)
		assert.Equal(t, specialID, requestID)
	})

	t.Run("handles whitespace in request ID", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"request-id", "  has-spaces  ",
		))

		requestID := ExtractRequestID(ctx)
		// The function doesn't trim whitespace, so it should preserve it
		assert.Equal(t, "  has-spaces  ", requestID)
	})

	t.Run("verifies UUID v6 characteristics", func(t *testing.T) {
		ctx := context.Background()

		// Generate multiple UUIDs to verify they follow v6 pattern
		for i := 0; i < 10; i++ {
			requestID := ExtractRequestID(ctx)
			require.NotEmpty(t, requestID)

			// Split UUID to check version field
			parts := strings.Split(requestID, "-")
			require.Len(t, parts, 5)

			// Version field is the first character of the 3rd group
			// For UUID v6, this should be '6'
			assert.True(t, strings.HasPrefix(parts[2], "6"),
				"UUID version field should start with 6 for v6 UUIDs, got: %s", requestID)
		}
	})
}
