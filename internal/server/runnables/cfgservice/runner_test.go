package cfgservice

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
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

// MockConfigOrchestrator implements the ConfigOrchestrator interface for testing
type MockConfigOrchestrator struct {
	mock.Mock
}

// ProcessTransaction implements the ConfigOrchestrator interface
func (m *MockConfigOrchestrator) ProcessTransaction(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// RegisterParticipant implements the ConfigOrchestrator interface
func (m *MockConfigOrchestrator) RegisterParticipant(participant SagaParticipant) error {
	args := m.Called(participant)
	return args.Error(0)
}

// GetTransactionStatus implements the ConfigOrchestrator interface
func (m *MockConfigOrchestrator) GetTransactionStatus(txID string) (map[string]interface{}, error) {
	args := m.Called(txID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// TestRunner_New tests the creation of a new Runner
func TestRunner_New(t *testing.T) {
	mockOrchestrator := new(MockConfigOrchestrator)

	t.Run("minimal config with listen address", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		r, err := NewRunner(listenAddr, mockOrchestrator)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.NotNil(t, r.logger)
		assert.NotNil(t, r.reloadCh)
		assert.Equal(t, listenAddr, r.listenAddr)
		assert.Equal(t, mockOrchestrator, r.orchestrator)
	})

	t.Run("with custom logger", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		r, err := NewRunner(
			listenAddr,
			mockOrchestrator,
			WithLogger(customLogger),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, listenAddr, r.listenAddr)
		assert.Equal(t, customLogger, r.logger)
	})

	t.Run("with custom grpc server", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		mockServer := new(MockGRPCServer)
		r, err := NewRunner(
			listenAddr,
			mockOrchestrator,
			WithGRPCServer(mockServer),
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, listenAddr, r.listenAddr)
		assert.Equal(t, mockServer, r.grpcServer)
	})

	t.Run("with empty listen address", func(t *testing.T) {
		r, err := NewRunner("", mockOrchestrator)
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "listen address cannot be empty")
	})

	t.Run("with nil orchestrator", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		r, err := NewRunner(listenAddr, nil)
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "config orchestrator cannot be nil")
	})
}

// TestStop tests the Stop method of Runner
func TestStop(t *testing.T) {
	mockOrchestrator := new(MockConfigOrchestrator)

	t.Run("with grpc server", func(t *testing.T) {
		// Create a Runner instance
		listenAddr := testutil.GetRandomListeningPort(t)
		r, err := NewRunner(listenAddr, mockOrchestrator)
		require.NoError(t, err)

		// Start the runner in a goroutine
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		runErrCh := make(chan error, 1)
		go func() {
			runErrCh <- r.Run(ctx)
		}()

		// Wait for the server to start (it will transition to Running state)
		require.Eventually(t, func() bool {
			return r.IsRunning()
		}, 1*time.Second, 10*time.Millisecond, "Server should reach Running state")

		// Now replace the gRPC server with a mock to test GracefulStop
		mockServer := new(MockGRPCServer)
		mockServer.On("GracefulStop").Return()

		r.grpcLock.Lock()
		r.grpcServer = mockServer
		r.grpcLock.Unlock()

		// Call Stop
		r.Stop()

		// Wait for Run to complete
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Run did not complete in time")
		}

		// Verify that GracefulStop was called on our mock
		mockServer.AssertCalled(t, "GracefulStop")
	})

	t.Run("with nil server", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		r, err := NewRunner(listenAddr, mockOrchestrator)
		require.NoError(t, err)

		// Transition to running state to simulate a started runner
		err = r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Ensure server is nil
		r.grpcServer = nil

		// Stop should not panic
		r.Stop()
	})
}

// TestString tests the String method of Runner
func TestString(t *testing.T) {
	mockOrchestrator := new(MockConfigOrchestrator)

	listenAddr := testutil.GetRandomListeningPort(t)
	r, err := NewRunner(listenAddr, mockOrchestrator)
	require.NoError(t, err)

	// Check that String returns expected value
	assert.Equal(t, "cfgservice.Runner", r.String())
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

	listenAddr := testutil.GetRandomListeningPort(t)

	// Create a mock orchestrator
	mockOrchestrator := new(MockConfigOrchestrator)

	r, err := NewRunner(listenAddr, mockOrchestrator)
	require.NoError(t, err)

	// Set up the mock to simulate successful transaction completion
	mockOrchestrator.On("ProcessTransaction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Simulate successful transaction completion by setting it as current
			tx := args.Get(1).(*transaction.ConfigTransaction)
			r.txStorage.SetCurrent(tx)
		}).
		Return(nil)

	// Initialize the state properly for testing
	err = r.fsm.Transition(finitestate.StatusBooting)
	require.NoError(t, err)
	err = r.fsm.Transition(finitestate.StatusRunning)
	require.NoError(t, err)

	// Set initial configuration
	version := "v1"
	initialPbConfig := &pb.ServerConfig{
		Version: &version,
	}

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
					Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
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

	// Verify orchestrator was called
	mockOrchestrator.AssertCalled(t, "ProcessTransaction", mock.Anything, mock.Anything)

	// Clean up
	server.Stop()
}

// TestReloadChannel tests the reload notification channel
func TestReloadChannel(t *testing.T) {
	// Create a mock orchestrator
	mockOrchestrator := new(MockConfigOrchestrator)

	// Create a Runner instance
	listenAddr := testutil.GetRandomListeningPort(t)
	r, err := NewRunner(listenAddr, mockOrchestrator)
	require.NoError(t, err)

	// Set up the mock to simulate successful transaction processing
	mockOrchestrator.On("ProcessTransaction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			tx := args.Get(1).(*transaction.ConfigTransaction)
			r.txStorage.SetCurrent(tx)
		}).
		Return(nil)

	// Initialize the state properly for testing
	err = r.fsm.Transition(finitestate.StatusBooting)
	require.NoError(t, err)
	err = r.fsm.Transition(finitestate.StatusRunning)
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
