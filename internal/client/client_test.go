package client

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLoader is a mock implementation of the loader.Loader interface
type MockLoader struct {
	mock.Mock
}

func (m *MockLoader) LoadProto() (*pb.ServerConfig, error) {
	args := m.Called()
	return args.Get(0).(*pb.ServerConfig), args.Error(1)
}

func (m *MockLoader) GetProtoConfig() *pb.ServerConfig {
	args := m.Called()
	return args.Get(0).(*pb.ServerConfig)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "with logger",
			cfg: Config{
				Logger:     slog.Default(),
				ServerAddr: "localhost:8080",
			},
		},
		{
			name: "without logger",
			cfg: Config{
				ServerAddr: "localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.cfg)
			assert.NotNil(t, client)
			assert.NotNil(t, client.logger)
			assert.Equal(t, tt.cfg.ServerAddr, client.serverAddr)
		})
	}
}

func TestApplyConfig(t *testing.T) {
	// Create a mock loader
	mockLoader := new(MockLoader)

	// Set up the mock loader to return a test config
	testConfig := &pb.ServerConfig{}
	mockLoader.On("LoadProto").Return(testConfig, nil)

	// Create a client with an invalid address to force connection error
	client := New(Config{
		ServerAddr: "invalid-host:-1",                              // Invalid port to force error
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)), // Discard logs for test
	})

	// This should fail at connection time due to invalid address
	err := client.ApplyConfig(context.Background(), mockLoader)
	assert.Error(t, err)

	// Verify the mock was called
	mockLoader.AssertExpectations(t)
}

func TestSaveConfig(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "config.toml")

	// Create a test client
	client := New(Config{
		ServerAddr: "localhost:8080",
	})

	// Create a test config
	version := "v1"
	format := pb.LogFormat_LOG_FORMAT_JSON
	level := pb.LogLevel_LOG_LEVEL_INFO
	testConfig := &pb.ServerConfig{
		Version: &version,
		Logging: &pb.LogOptions{
			Level:  &level,
			Format: &format,
		},
	}

	// Save the config
	err := client.SaveConfig(testConfig, outputPath)
	require.NoError(t, err)

	// Verify the file exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	// Verify the file contains the expected content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Just check for some expected content (uppercase V in Version)
	assert.Contains(t, string(content), "Version")

	// Check for enum values (these will be numbers in the TOML output)
	contentStr := string(content)
	assert.Contains(t, contentStr, "Format")
	assert.Contains(t, contentStr, "Level")
}

func TestConnect(t *testing.T) {
	tests := []struct {
		name        string
		serverAddr  string
		wantErr     bool
		expectedErr error
	}{
		{
			name:       "valid tcp address with prefix",
			serverAddr: "tcp://localhost:8080",
			wantErr:    false,
		},
		{
			name:       "valid tcp address without prefix",
			serverAddr: "localhost:8080",
			wantErr:    false,
		},
		{
			name:        "invalid tcp address format",
			serverAddr:  "invalid:::address",
			wantErr:     true,
			expectedErr: ErrInvalidTCPFormat,
		},
		{
			name:       "valid unix socket address",
			serverAddr: "unix:///tmp/socket",
			wantErr:    false,
		},
		{
			name:        "unsupported network type",
			serverAddr:  "http://localhost:8080",
			wantErr:     true,
			expectedErr: ErrUnsupportedNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(Config{
				ServerAddr: tt.serverAddr,
				// Use a discarded logger to prevent log output during tests
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			conn, err := client.connect(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil && err != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
				assert.Nil(t, conn)
			} else {
				// In a real test, we'd mock the gRPC client
				// This test may fail when run as a unit test without a server,
				// but our implementation won't actually try to connect until
				// the client makes an RPC call, so no error here
				assert.NotNil(t, conn)
				if conn != nil {
					if closeErr := conn.Close(); closeErr != nil {
						t.Logf("Failed to close connection: %v", closeErr)
					}
				}
			}
		})
	}
}
