package cfgservice

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// MockGRPCServer implements the GRPCServer interface for testing with testify/mock
type MockGRPCServer struct {
	mock.Mock
}

// Start implements the GRPCServer interface
func (m *MockGRPCServer) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// GracefulStop implements the GRPCServer interface
func (m *MockGRPCServer) GracefulStop() {
	m.Called()
}

// GetListenAddress implements the GRPCServer interface
func (m *MockGRPCServer) GetListenAddress() string {
	args := m.Called()
	return args.String(0)
}

// TestRunner_New tests the creation of a new Runner
func TestRunner_New(t *testing.T) {
	t.Run("minimal config with listen address", func(t *testing.T) {
		r, err := New(
			WithListenAddr(testutil.GetRandomListeningPort(t)),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.NotNil(t, r.logger)
		assert.NotNil(t, r.reloadCh)
		assert.Nil(t, r.config)
		assert.NotEmpty(t, r.listenAddr)
		assert.Empty(t, r.configPath)
	})

	t.Run("with config path", func(t *testing.T) {
		r, err := New(
			WithConfigPath("test.toml"),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.Empty(t, r.listenAddr)
		assert.Equal(t, "test.toml", r.configPath)
	})

	t.Run("with both listen address and config path", func(t *testing.T) {
		r, err := New(
			WithListenAddr(testutil.GetRandomListeningPort(t)),
			WithConfigPath("test.toml"),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.NotEmpty(t, r.listenAddr)
		assert.Equal(t, "test.toml", r.configPath)
	})

	t.Run("without listen address or config path", func(t *testing.T) {
		r, err := New()
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

// TestStop tests the Stop method of Runner
func TestStop(t *testing.T) {
	t.Run("with grpc server", func(t *testing.T) {
		// Create a mock GRPC server
		mockServer := new(MockGRPCServer)
		mockServer.On("GracefulStop").Return()

		// Create a Runner instance
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Set the grpcServer directly instead of starting it
		r.grpcServer = mockServer

		// Call Stop
		r.Stop()

		// Verify that GracefulStop was called on our mock
		mockServer.AssertCalled(t, "GracefulStop")
	})

	t.Run("with nil server", func(t *testing.T) {
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Ensure server is nil
		r.grpcServer = nil

		// Stop should not panic
		r.Stop()
	})
}

// TestString tests the String method of Runner
func TestString(t *testing.T) {
	r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
	require.NoError(t, err)

	// Check that String returns expected value
	assert.Equal(t, "cfgrpc.Runner", r.String())
}

// Helper function for gRPC testing
func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

// TestGRPCIntegration tests the integration between Runner and gRPC
func TestGRPCIntegration(t *testing.T) {
	// Create a buffer for the gRPC connection
	bufSize := 1024 * 1024
	listener := bufconn.Listen(bufSize)

	r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
	require.NoError(t, err)

	// Set initial configuration
	version := "v1"
	initialPbConfig := &pb.ServerConfig{
		Version: &version,
	}

	// Convert to domain config
	initialDomainConfig, err := config.NewFromProto(initialPbConfig)
	require.NoError(t, err)

	r.configMu.Lock()
	r.config = initialDomainConfig
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
	assert.Equal(t, *initialPbConfig.Version, *getResp.Config.Version)

	// Test UpdateConfig with valid configuration
	listenerId := "http_listener"
	listenerAddr := ":8080"
	updateReq := &pb.UpdateConfigRequest{
		Config: &pb.ServerConfig{
			Version: &version, // Keep using v1 which is valid
			Listeners: []*pb.Listener{
				{
					Id:      &listenerId,
					Address: &listenerAddr,
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
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

// TestReloadChannel tests the reload notification channel
func TestReloadChannel(t *testing.T) {
	// Create a Runner instance
	r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
	require.NoError(t, err)

	// Get the reload channel
	reloadCh := r.GetReloadTrigger()

	// Create update request with new configuration
	version := "v1"
	pbConfig := &pb.ServerConfig{
		Version: &version,
	}
	req := &pb.UpdateConfigRequest{
		Config: pbConfig,
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
