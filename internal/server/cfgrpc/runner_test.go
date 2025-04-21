package cfgrpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// MockGRPCServer implements the GRPCServer interface for testing with testify/mock
type MockGRPCServer struct {
	mock.Mock
}

// GracefulStop implements the GRPCServer interface
func (m *MockGRPCServer) GracefulStop() {
	m.Called()
}

// TestConfigManager_New tests the creation of a new ConfigManager
func TestConfigManager_New(t *testing.T) {
	t.Run("ConfigManager instance with minimal config", func(t *testing.T) {
		r, err := New(
			WithListenAddr(":8080"),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.NotNil(t, r.logger)
		assert.NotNil(t, r.reloadCh)
		assert.Nil(t, r.config)
		assert.Equal(t, r.listenAddr, ":8080")
		assert.Empty(t, r.configPath)
		assert.NotNil(t, r.startGRPCServer)
	})

	t.Run("complete config", func(t *testing.T) {
		r, err := New(
			WithListenAddr(":8080"),
			WithConfigPath("test.toml"),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, ":8080", r.listenAddr)
		assert.Equal(t, "test.toml", r.configPath)
	})
}

func TestConfigManager_GetCurrentConfig(t *testing.T) {
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	r.config = testConfig
	result := r.GetConfigClone()
	require.NotNil(t, result)
	assert.Equal(t, testConfig, result)

	// change a value, confirm they're not the same
	v := "v999"
	testConfig.Version = &v
	assert.NotEqual(t, testConfig, result)
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	// In this test, we explicitly test the validation behavior
	// Create a ConfigManager instance
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	version := "v1"
	initialConfig := &pb.ServerConfig{
		Version: &version,
	}
	r.config = initialConfig

	// Create update request with INVALID configuration (v2 is not supported)
	newVersion := "v2"
	invalidConfig := &pb.ServerConfig{
		Version: &newVersion,
	}
	invalidReq := &pb.UpdateConfigRequest{
		Config: invalidConfig,
	}

	// Call UpdateConfig with invalid config
	invalidResp, err := r.UpdateConfig(context.Background(), invalidReq)

	// Expect validation error as a gRPC InvalidArgument error
	require.Error(t, err, "Should receive validation error for unsupported version")
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Nil(t, invalidResp)

	// Verify that the internal config was NOT updated
	result := r.GetConfigClone()
	assert.Equal(t, initialConfig, result, "Config should not change after failed validation")

	// Now create a valid update request
	validConfig := &pb.ServerConfig{
		Version: &version, // Keep v1 which is valid
		Listeners: []*pb.Listener{
			{
				Id:      &[]string{"http_listener"}[0],
				Address: &[]string{":8080"}[0],
			},
		},
	}
	validReq := &pb.UpdateConfigRequest{
		Config: validConfig,
	}

	// Call UpdateConfig with valid config
	validResp, err := r.UpdateConfig(context.Background(), validReq)

	// Should succeed
	require.NoError(t, err, "Valid config should not cause error")
	assert.NotNil(t, validResp)
	assert.NotNil(t, validResp.Success)
	assert.True(t, *validResp.Success, "Success should be true for valid config")
	assert.Equal(t, validConfig, validResp.Config)

	// Verify that the internal config was updated
	result = r.GetConfigClone()
	assert.Equal(t, validConfig, result, "Config should be updated after successful validation")
}

func TestConfigManager_GetConfig(t *testing.T) {
	// Create a ConfigManager instance
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	// Set a test configuration
	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	r.configMu.Lock()
	r.config = testConfig
	r.configMu.Unlock()

	// Call GetConfig
	resp, err := r.GetConfig(context.Background(), &pb.GetConfigRequest{})

	// Verify response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, testConfig, resp.Config)
}

func TestConfigManager_ReloadChannel(t *testing.T) {
	// Create a ConfigManager instance
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	// Get the reload channel
	reloadCh := r.GetReloadTrigger()

	// Create update request with new configuration
	version := "v1"
	config := &pb.ServerConfig{
		Version: &version,
	}
	req := &pb.UpdateConfigRequest{
		Config: config,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup a goroutine to call UpdateConfig
	go func() {
		// We're only testing the notification, not the response
		resp, err := r.UpdateConfig(ctx, req)
		if err != nil {
			t.Logf("UpdateConfig error (expected in tests): %v", err)
		}
		if resp == nil {
			t.Logf("UpdateConfig returned nil response (expected in tests)")
		}
	}()

	// Wait for reload notification
	select {
	case <-reloadCh:
		// Success - reload notification received
	case <-ctx.Done():
		t.Fatal("Timeout waiting for reload notification")
	}
}

func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

// TestConfigManager_Run tests the Run method of ConfigManager
func TestConfigManager_Run(t *testing.T) {
	// Create a ConfigManager instance with a listen address
	r, err := New(
		WithListenAddr("localhost:0"), // Use port 0 for automatic port assignment in tests
	)
	require.NoError(t, err)

	// Create a context that will cancel after a short time
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run the ConfigManager in a goroutine
	runErr := make(chan error)
	go func() {
		runErr <- r.Run(ctx)
	}()

	// Wait for the context to time out
	chanErr := <-runErr
	assert.NoError(t, chanErr)
}

// TestConfigManager_Stop tests that Stop calls GracefulStop on the GRPC server
func TestConfigManager_Stop(t *testing.T) {
	// Create a mock GRPC server
	mockServer := new(MockGRPCServer)
	mockServer.On("GracefulStop").Return()

	// Create a ConfigManager instance
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	// Set the grpcServer directly instead of starting it
	r.grpcServer = mockServer

	// Call Stop
	r.Stop()

	// Verify that GracefulStop was called on our mock
	mockServer.AssertCalled(t, "GracefulStop")
}

// TestConfigManager_RunWithConfigPath tests the Run method with a config path
func TestConfigManager_RunWithConfigPath(t *testing.T) {
	// Create a temporary directory that's automatically cleaned up
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write default config to the file
	err := os.WriteFile(configPath, []byte("version = \"v1\"\n"), 0o644)
	require.NoError(t, err)

	// Create a ConfigManager with the config path
	r, err := New(
		WithListenAddr(":8080"),
		WithConfigPath(configPath),
	)
	require.NoError(t, err)

	// Run for a short time
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = r.Run(ctx)
	assert.NoError(t, err) // Should return nil on clean shutdown with supervisor pattern
}

// TestConfigManager_RunWithListenAddr tests the Run method with a listen address
func TestConfigManager_RunWithListenAddr(t *testing.T) {
	// Use a random port to avoid conflicts
	r, err := New(
		WithListenAddr(":8080"),
		WithListenAddr(":0"), // Use port 0 to get a random available port
	)
	require.NoError(t, err)

	// Run for a short time
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()

	// Wait for the context to time out
	select {
	case err := <-errCh:
		assert.NoError(t, err) // Should return nil on clean shutdown with supervisor pattern
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for ConfigManager to run")
	}
}

// TestConfigManager_InvalidUpdateConfig tests the UpdateConfig method with invalid input
func TestConfigManager_InvalidUpdateConfig(t *testing.T) {
	// Create a ConfigManager instance
	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	// Call UpdateConfig with nil request
	resp, err := r.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
		Config: nil,
	})

	// Verify response gets a gRPC InvalidArgument error
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "No configuration provided")
	assert.Nil(t, resp)
}

func TestConfigManager_GRPC(t *testing.T) {
	// Create a buffer for the gRPC connection
	bufSize := 1024 * 1024
	listener := bufconn.Listen(bufSize)

	r, err := New(WithListenAddr(":8080"))
	require.NoError(t, err)

	// Set initial configuration
	version := "v1"
	initialConfig := &pb.ServerConfig{
		Version: &version,
	}
	r.configMu.Lock()
	r.config = initialConfig
	r.configMu.Unlock()

	// Create a gRPC server
	server := grpc.NewServer()
	pb.RegisterConfigServiceServer(server, r)

	// Serve gRPC in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("Failed to serve: %v", err)
		}
	}()

	// Create a gRPC client
	// We need to use a passthrough resolver with a valid URI scheme for NewClient
	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection (non-critical error): %v", err)
		}
	}()

	// Create a client
	client := pb.NewConfigServiceClient(conn)

	// Test GetConfig
	getResp, err := client.GetConfig(ctx, &pb.GetConfigRequest{})
	require.NoError(t, err)
	// Compare only the version field, not the entire object
	assert.Equal(t, *initialConfig.Version, *getResp.Config.Version)

	// Test UpdateConfig with valid configuration
	// Use same version (v1) but add listeners to make it different
	updateReq := &pb.UpdateConfigRequest{
		Config: &pb.ServerConfig{
			Version: &version, // Keep using v1 which is valid
			Listeners: []*pb.Listener{
				{
					Id:      &[]string{"http_listener"}[0],
					Address: &[]string{":8080"}[0],
				},
			},
		},
	}
	updateResp, err := client.UpdateConfig(ctx, updateReq)
	require.NoError(t, err)
	assert.True(t, *updateResp.Success)

	// Test GetConfig again to verify update
	getResp, err = client.GetConfig(ctx, &pb.GetConfigRequest{})
	require.NoError(t, err)
	assert.Equal(t, version, *getResp.Config.Version)
	assert.Equal(t, 1, len(getResp.Config.Listeners))

	// Clean up
	server.Stop()
}
