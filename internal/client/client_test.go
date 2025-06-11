package client

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
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
	err := client.ApplyConfig(t.Context(), mockLoader)
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
	version := version.Version
	testConfig := &pb.ServerConfig{
		Version: &version,
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

	// Check for basic config structure
	contentStr := string(content)
	assert.Contains(t, contentStr, "Version")
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

			conn, err := client.connect(t.Context())
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

func TestFormatConfig(t *testing.T) {
	// Create a test client
	client := New(Config{
		ServerAddr: "localhost:8080",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	v := version.Version
	tests := []struct {
		name    string
		config  *pb.ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &pb.ServerConfig{
				Version: &v,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false, // TOML can marshal nil as empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.FormatConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	// Create a client with an invalid address to force connection error
	client := New(Config{
		ServerAddr: "invalid-host:-1",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// This should fail at connection time
	config, err := client.GetConfig(t.Context())
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestValidateConfig(t *testing.T) {
	// Create a client with an invalid address to force connection error
	client := New(Config{
		ServerAddr: "invalid-host:-1",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	v := version.Version
	testConfig := &pb.ServerConfig{Version: &v}

	// This should fail at connection time
	isValid, err := client.ValidateConfig(t.Context(), testConfig)
	assert.Error(t, err)
	assert.False(t, isValid)
	assert.Contains(t, err.Error(), "failed to validate configuration")
}

func TestApplyConfigWithMockLoader(t *testing.T) {
	v := version.Version
	tests := []struct {
		name           string
		setupMock      func(*MockLoader)
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "loader returns error",
			setupMock: func(m *MockLoader) {
				m.On("LoadProto").Return((*pb.ServerConfig)(nil), assert.AnError)
			},
			wantErr:        true,
			expectedErrMsg: "failed to parse configuration",
		},
		{
			name: "valid config but connection fails",
			setupMock: func(m *MockLoader) {
				testConfig := &pb.ServerConfig{Version: &v}
				m.On("LoadProto").Return(testConfig, nil)
			},
			wantErr:        true,
			expectedErrMsg: "failed to connect to server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLoader := new(MockLoader)
			tt.setupMock(mockLoader)

			client := New(Config{
				ServerAddr: "invalid-host:-1",
				Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			err := client.ApplyConfig(t.Context(), mockLoader)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}

			mockLoader.AssertExpectations(t)
		})
	}
}
