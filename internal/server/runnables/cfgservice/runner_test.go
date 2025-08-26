package cfgservice

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	ctx, cancel := context.WithCancel(t.Context())

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
		require.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "listen address cannot be empty")
	})

	t.Run("with nil tx siphon", func(t *testing.T) {
		r, err := NewRunner(testutil.GetRandomListeningPort(t), nil)
		require.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "transaction siphon cannot be nil")
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

// TestGetDomainConfig tests the GetDomainConfig method
func TestGetDomainConfig(t *testing.T) {
	t.Run("nil transaction storage", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Set transaction storage to return nil
		r.txStorage = &mockTxStorageNil{}

		// Should return a minimal default config
		cfg := r.GetDomainConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, config.VersionLatest, cfg.Version)
		assert.NotNil(t, cfg.Apps, "Apps should be initialized")
		assert.Equal(t, 0, cfg.Apps.Len(), "Apps should be empty for minimal config")
	})

	t.Run("normal transaction with valid config", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Create a test config
		testConfig, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		testConfig.Version = config.VersionLatest

		// Set transaction storage to return transaction with valid config
		r.txStorage = &mockTxStorageWithConfig{cfg: testConfig}

		// Should return the actual config
		cfg := r.GetDomainConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, config.VersionLatest, cfg.Version)
		assert.NotNil(t, cfg.Apps, "Apps should be initialized")
		assert.Equal(t, 0, cfg.Apps.Len(), "Should preserve apps from original config")
	})

	t.Run("config creation error fallback", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Set transaction storage to return nil (will trigger fallback)
		r.txStorage = &mockTxStorageNil{}

		// This test ensures the error path fallback works
		// The fallback should still work in normal cases, but we test the path exists
		cfg := r.GetDomainConfig()
		assert.NotNil(t, cfg)
		// Even with error, should return something (zero value or minimal config)
	})
}

// mockTxStorageNil is a test implementation that always returns nil
type mockTxStorageNil struct{}

func (m *mockTxStorageNil) SetCurrent(tx *transaction.ConfigTransaction) {}

func (m *mockTxStorageNil) GetCurrent() *transaction.ConfigTransaction {
	return nil
}

func (m *mockTxStorageNil) GetAll() []*transaction.ConfigTransaction {
	return []*transaction.ConfigTransaction{}
}

func (m *mockTxStorageNil) GetByID(id string) *transaction.ConfigTransaction {
	return nil
}

func (m *mockTxStorageNil) Clear(keepLast int) (int, error) {
	return 0, nil
}

// mockTxStorageWithConfig returns a transaction that has a specific config
type mockTxStorageWithConfig struct {
	cfg *config.Config
	tx  *transaction.ConfigTransaction
}

func (m *mockTxStorageWithConfig) SetCurrent(tx *transaction.ConfigTransaction) {}

func (m *mockTxStorageWithConfig) GetCurrent() *transaction.ConfigTransaction {
	// Create a test transaction with the specific config
	if m.tx == nil {
		var err error
		m.tx, err = transaction.FromTest("test-config", m.cfg, slog.Default().Handler())
		if err != nil {
			// Transaction creation failure is a test setup error - fail fast
			panic("test setup error: failed to create test transaction: " + err.Error())
		}
	}
	return m.tx
}

func (m *mockTxStorageWithConfig) GetAll() []*transaction.ConfigTransaction {
	return []*transaction.ConfigTransaction{}
}

func (m *mockTxStorageWithConfig) GetByID(id string) *transaction.ConfigTransaction {
	return nil
}

func (m *mockTxStorageWithConfig) Clear(keepLast int) (int, error) {
	return 0, nil
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
	ctx := t.Context()
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
					Type:    pb.Listener_TYPE_HTTP.Enum(),
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
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		require.NoError(t, chanErr)
	})

	t.Run("with_invalid_address", func(t *testing.T) {
		// Create a Runner with an invalid listen address that will cause NewGRPCManager to fail
		listenAddr := "invalid:address:with:too:many:colons"
		h := newRunnerTestHarness(t, listenAddr)
		r := h.runner

		// Run should return the error from NewGRPCManager
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		err := r.Run(ctx)
		require.Error(
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
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		require.NoError(t, chanErr)
	})

	t.Run("stop_before_run", func(t *testing.T) {
		// Create a Runner instance
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Call Stop before Run
		r.Stop()

		// This should not panic or cause issues
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// Run should handle being stopped before starting
		err := r.Run(ctx)
		require.NoError(t, err)
	})

	t.Run("grpc_server_already_running", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Set a mock grpc server to simulate it already running
		mockServer := new(MockGRPCServer)
		r.grpcServer = mockServer

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		err := r.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gRPC server is already running")
	})
}

// TestRunErrorHandling tests error handling scenarios in the Run method
func TestRunErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("grpc_manager_creation_failure", func(t *testing.T) {
		// Test failure during NewGRPCManager creation by using an address that's already in use
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { require.NoError(t, listener.Close()) }()

		// Use the address that's already being listened on
		busyAddr := listener.Addr().String()

		h := newRunnerTestHarness(t, busyAddr)
		r := h.runner

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		err = r.Run(ctx)
		require.Error(t, err)

		// The FSM should be in error state after NewGRPCManager failure
		assert.Equal(t, finitestate.StatusError, r.fsm.GetState())
	})

	t.Run("concurrent_run_calls", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		// Start two Run calls concurrently
		errCh1 := make(chan error, 1)
		errCh2 := make(chan error, 1)

		go func() {
			errCh1 <- r.Run(ctx)
		}()

		// Give first Run a chance to start
		time.Sleep(10 * time.Millisecond)

		go func() {
			errCh2 <- r.Run(ctx)
		}()

		// Wait for both to complete
		err1 := <-errCh1
		err2 := <-errCh2

		// One should succeed (or timeout), one should fail with "already running"
		if err1 != nil && err2 != nil {
			// Both failed - one should be "already running"
			hasAlreadyRunningError := strings.Contains(
				err1.Error(),
				"gRPC server is already running",
			) ||
				strings.Contains(err2.Error(), "gRPC server is already running")
			assert.True(
				t,
				hasAlreadyRunningError,
				"One error should be about server already running",
			)
		}
	})
}

// TestUpdateConfigErrorHandling tests error scenarios in UpdateConfig
func TestUpdateConfigErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil_config_request", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		h.transitionToRunning()

		// Call UpdateConfig with nil config
		resp, err := r.UpdateConfig(t.Context(), &pb.UpdateConfigRequest{
			Config: nil,
		})

		// Should return gRPC error
		require.Error(t, err)
		assert.Nil(t, resp)

		// Should be InvalidArgument error
		grpcStatus, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, grpcStatus.Code())
		assert.Contains(t, grpcStatus.Message(), "No configuration provided")
	})

	t.Run("transaction_creation_failure", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		h.transitionToRunning()

		// Create a config that should convert successfully but fail transaction creation
		// We'll use an invalid version that passes proto conversion but fails validation
		version := "v999" // Invalid version
		config := &pb.ServerConfig{
			Version: &version,
		}

		resp, err := r.UpdateConfig(t.Context(), &pb.UpdateConfigRequest{
			Config: config,
		})

		// Should return success=false response (not gRPC error)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.GetSuccess())
		assert.Contains(t, resp.GetError(), "validation failed")
		assert.Equal(t, config, resp.Config) // Should return submitted config
	})

	t.Run("siphon_channel_blocked", func(t *testing.T) {
		// Create a runner with unbuffered siphon to simulate blocking
		txSiphon := make(chan *transaction.ConfigTransaction) // unbuffered
		runner, err := NewRunner(
			testutil.GetRandomListeningPort(t),
			txSiphon,
		)
		require.NoError(t, err)

		// Transition to running state manually
		require.NoError(t, runner.fsm.Transition(finitestate.StatusBooting))
		require.NoError(t, runner.fsm.Transition(finitestate.StatusRunning))

		// Create a context that will cancel quickly
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		// Create a valid config
		version := "v1"
		config := &pb.ServerConfig{
			Version: &version,
		}

		// Since the siphon channel is unbuffered and no one is reading from it,
		// the send should block and the context should cancel
		resp, err := runner.UpdateConfig(ctx, &pb.UpdateConfigRequest{
			Config: config,
		})

		// Should return success=false due to context cancellation
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.GetSuccess())
		assert.Contains(t, resp.GetError(), "service shutting down")
	})
}

// TestConcurrentAccess tests concurrent access scenarios
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("concurrent_grpc_calls", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		h.transitionToRunning()

		const numGoroutines = 10
		var wg sync.WaitGroup
		results := make(chan error, numGoroutines)

		// Launch multiple GetConfig calls concurrently
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := r.GetConfig(t.Context(), &pb.GetConfigRequest{})
				results <- err
			}()
		}

		wg.Wait()
		close(results)

		// All calls should succeed
		errorCount := 0
		for err := range results {
			if err != nil {
				t.Errorf("GetConfig call failed: %v", err)
				errorCount++
			}
		}
		assert.Equal(t, 0, errorCount, "All concurrent calls should succeed")
	})

	t.Run("concurrent_transaction_operations", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		h.transitionToRunning()

		const numGoroutines = 5
		var wg sync.WaitGroup
		results := make(chan error, numGoroutines*2)

		// Launch concurrent GetCurrentConfigTransaction and ListConfigTransactions calls
		for range numGoroutines {
			wg.Add(2)

			go func() {
				defer wg.Done()
				_, err := r.GetCurrentConfigTransaction(
					t.Context(),
					&pb.GetCurrentConfigTransactionRequest{},
				)
				results <- err
			}()

			go func() {
				defer wg.Done()
				_, err := r.ListConfigTransactions(t.Context(), &pb.ListConfigTransactionsRequest{})
				results <- err
			}()
		}

		wg.Wait()
		close(results)

		// All calls should succeed
		errorCount := 0
		for err := range results {
			if err != nil {
				t.Errorf("Transaction operation failed: %v", err)
				errorCount++
			}
		}
		assert.Equal(t, 0, errorCount, "All concurrent transaction operations should succeed")
	})
}

// TestEncodeDecodePageToken tests the pagination token encoding and decoding functions
func TestEncodeDecodePageToken(t *testing.T) {
	t.Parallel()

	t.Run("Valid token round trip", func(t *testing.T) {
		offset := 10
		pageSize := 25
		state := "completed"
		source := "api"

		// Encode token
		token, err := encodePageToken(offset, pageSize, state, source)
		require.NoError(t, err, "Should encode token successfully")
		assert.NotEmpty(t, token, "Token should not be empty")

		// Decode token
		decoded, err := decodePageToken(token)
		require.NoError(t, err, "Should decode token successfully")
		assert.Equal(t, offset, decoded.Offset, "Offset should match")
		assert.Equal(t, pageSize, decoded.PageSize, "Page size should match")
		assert.Equal(t, state, decoded.State, "State should match")
		assert.Equal(t, source, decoded.Source, "Source should match")
	})

	t.Run("Empty strings in token", func(t *testing.T) {
		offset := 0
		pageSize := 10
		state := ""
		source := ""

		// Encode token
		token, err := encodePageToken(offset, pageSize, state, source)
		require.NoError(t, err, "Should encode token with empty strings")
		assert.NotEmpty(t, token, "Token should not be empty")

		// Decode token
		decoded, err := decodePageToken(token)
		require.NoError(t, err, "Should decode token with empty strings")
		assert.Equal(t, offset, decoded.Offset, "Offset should match")
		assert.Equal(t, pageSize, decoded.PageSize, "Page size should match")
		assert.Equal(t, state, decoded.State, "State should match")
		assert.Equal(t, source, decoded.Source, "Source should match")
	})

	t.Run("Decode empty token", func(t *testing.T) {
		decoded, err := decodePageToken("")
		require.NoError(t, err, "Should handle empty token gracefully")
		assert.Equal(t, pageToken{}, decoded, "Should return zero value for empty token")
	})

	t.Run("Decode invalid base64", func(t *testing.T) {
		invalidToken := "invalid-base64-!@#$%"

		decoded, err := decodePageToken(invalidToken)
		require.Error(t, err, "Should return error for invalid base64")
		assert.Equal(t, pageToken{}, decoded, "Should return zero value on error")
		assert.Contains(t, err.Error(), "invalid page token format", "Error should mention invalid format")
	})

	t.Run("Decode invalid JSON", func(t *testing.T) {
		// Create a valid base64 string that contains invalid JSON
		invalidJSON := base64.URLEncoding.EncodeToString([]byte("{invalid json syntax"))

		decoded, err := decodePageToken(invalidJSON)
		require.Error(t, err, "Should return error for invalid JSON")
		assert.Equal(t, pageToken{}, decoded, "Should return zero value on error")
		assert.Contains(t, err.Error(), "failed to unmarshal page token", "Error should mention unmarshal failure")
	})

	t.Run("Large values", func(t *testing.T) {
		offset := 999999
		pageSize := 100
		state := "very_long_state_string_that_should_still_work_fine_in_encoding"
		source := "very_long_source_string_that_should_also_work_fine"

		// Encode token
		token, err := encodePageToken(offset, pageSize, state, source)
		require.NoError(t, err, "Should encode token with large values")
		assert.NotEmpty(t, token, "Token should not be empty")

		// Decode token
		decoded, err := decodePageToken(token)
		require.NoError(t, err, "Should decode token with large values")
		assert.Equal(t, offset, decoded.Offset, "Large offset should match")
		assert.Equal(t, pageSize, decoded.PageSize, "Page size should match")
		assert.Equal(t, state, decoded.State, "Long state should match")
		assert.Equal(t, source, decoded.Source, "Long source should match")
	})

	t.Run("Negative values", func(t *testing.T) {
		offset := -1
		pageSize := -10
		state := "failed"
		source := "test"

		// Encode token (should work even with negative values)
		token, err := encodePageToken(offset, pageSize, state, source)
		require.NoError(t, err, "Should encode token with negative values")

		// Decode token
		decoded, err := decodePageToken(token)
		require.NoError(t, err, "Should decode token with negative values")
		assert.Equal(t, offset, decoded.Offset, "Negative offset should be preserved")
		assert.Equal(t, pageSize, decoded.PageSize, "Negative page size should be preserved")
	})
}
