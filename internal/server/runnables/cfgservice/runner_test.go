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

// runnerTestHarness provides a clean test setup for cfgservice Runner tests
type runnerTestHarness struct {
	t        *testing.T
	runner   *Runner
	txSiphon chan *transaction.ConfigTransaction
	ctx      context.Context
	cancel   context.CancelFunc
}

// newRunnerTestHarness creates a test harness for runner tests
func newRunnerTestHarness(t *testing.T, listenAddr string, opts ...Option) *runnerTestHarness {
	t.Helper()
	// Use buffered channel for tests to avoid blocking
	txSiphon := make(chan *transaction.ConfigTransaction, 10)
	runner, err := NewRunner(listenAddr, txSiphon, opts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())

	return &runnerTestHarness{
		t:        t,
		runner:   runner,
		txSiphon: txSiphon,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// transitionToRunning transitions the runner to Running state if not already there
func (h *runnerTestHarness) transitionToRunning() {
	h.t.Helper()
	r := h.runner
	if r.fsm.GetState() != finitestate.StatusRunning {
		if r.fsm.GetState() == finitestate.StatusNew {
			err := r.fsm.Transition(finitestate.StatusBooting)
			require.NoError(h.t, err)
		}
		err := r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(h.t, err)
	}
}

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
		listenAddr := testutil.GetRandomListeningPort(t)
		h := newRunnerTestHarness(t, listenAddr)
		r := h.runner
		assert.NotNil(t, r)
		assert.NotNil(t, r.logger)
		assert.Equal(t, listenAddr, r.listenAddr)
	})

	t.Run("with custom logger", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		h := newRunnerTestHarness(t, listenAddr, WithLogger(customLogger))
		r := h.runner
		assert.NotNil(t, r)
		assert.Equal(t, listenAddr, r.listenAddr)
		assert.Equal(t, customLogger, r.logger)
	})

	t.Run("with custom grpc server", func(t *testing.T) {
		listenAddr := testutil.GetRandomListeningPort(t)
		mockServer := new(MockGRPCServer)
		h := newRunnerTestHarness(t, listenAddr, WithGRPCServer(mockServer))
		r := h.runner
		assert.NotNil(t, r)
		assert.Equal(t, listenAddr, r.listenAddr)
		assert.Equal(t, mockServer, r.grpcServer)
	})

	t.Run("with empty listen address", func(t *testing.T) {
		// Use a buffered channel for this test case
		txSiphon := make(chan *transaction.ConfigTransaction, 1)
		r, err := NewRunner("", txSiphon)
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "listen address cannot be empty")
	})
}

// TestStop tests the Stop method of Runner
func TestStop(t *testing.T) {
	t.Run("with grpc server", func(t *testing.T) {
		// Create a Runner instance
		listenAddr := testutil.GetRandomListeningPort(t)
		h := newRunnerTestHarness(t, listenAddr)
		r := h.runner
		defer h.cancel()

		runErrCh := make(chan error, 1)
		go func() {
			runErrCh <- r.Run(h.ctx)
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
		h := newRunnerTestHarness(t, listenAddr)
		r := h.runner

		// Transition to running state to simulate a started runner
		h.transitionToRunning()

		// Ensure server is nil
		r.grpcServer = nil

		// Stop should not panic
		r.Stop()
	})
}

// TestString tests the String method of Runner
func TestString(t *testing.T) {
	listenAddr := testutil.GetRandomListeningPort(t)
	h := newRunnerTestHarness(t, listenAddr)
	r := h.runner

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

	h := newRunnerTestHarness(t, listenAddr)
	r := h.runner

	// Initialize the state properly for testing
	h.transitionToRunning()

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

	// Clean up
	server.Stop()
}

// TestRun tests all Run method functionality with different configurations
func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("basic_functionality", func(t *testing.T) {
		// Create a Runner instance with a listen address
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		assert.NoError(t, chanErr)
	})

	t.Run("with_invalid_address", func(t *testing.T) {
		// Create a Runner with an invalid listen address that will cause NewGRPCManager to fail
		listenAddr := "invalid:address:with:too:many:colons"
		h := newRunnerTestHarness(t, listenAddr)
		r := h.runner

		// Run should return the error from NewGRPCManager
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := r.Run(ctx)
		assert.Error(
			t,
			err,
			"Run should return an error when NewGRPCManager fails with an invalid address",
		)
	})

	t.Run("with_custom_logger", func(t *testing.T) {
		// Create a Runner instance with custom logger
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t),
			WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
		r := h.runner

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		assert.NoError(t, chanErr)
	})

	t.Run("stop_before_run", func(t *testing.T) {
		// Create a Runner instance
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Call Stop before Run
		r.Stop()

		// This should not panic or cause issues
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run should handle being stopped before starting
		err := r.Run(ctx)
		assert.NoError(t, err)
	})
}
