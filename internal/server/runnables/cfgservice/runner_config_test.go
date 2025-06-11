package cfgservice

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// mockTxStorage is a simple in-memory implementation of configTransactionStorage for testing
type mockTxStorage struct {
	mu      sync.RWMutex
	current *transaction.ConfigTransaction
}

func newMockTxStorage() *mockTxStorage {
	return &mockTxStorage{}
}

func (m *mockTxStorage) SetCurrent(tx *transaction.ConfigTransaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = tx
}

func (m *mockTxStorage) GetCurrent() *transaction.ConfigTransaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// testHarness provides a clean test setup for cfgservice
type testHarness struct {
	t         *testing.T
	runner    *Runner
	txSiphon  chan *transaction.ConfigTransaction
	txStorage *mockTxStorage
	ctx       context.Context
	cancel    context.CancelFunc
}

// newTestHarness creates a test harness with a buffered siphon channel
func newTestHarness(t *testing.T, listenAddr string, opts ...Option) *testHarness {
	t.Helper()
	// Use buffered channel for tests to avoid blocking
	txSiphon := make(chan *transaction.ConfigTransaction, 10)

	// Create mock transaction storage
	txStorage := newMockTxStorage()

	// Add the transaction storage option
	allOpts := append([]Option{WithConfigTransactionStorage(txStorage)}, opts...)

	runner, err := NewRunner(listenAddr, txSiphon, allOpts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())

	return &testHarness{
		t:         t,
		runner:    runner,
		txSiphon:  txSiphon,
		txStorage: txStorage,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// receiveTransaction waits for a transaction on the siphon
func (h *testHarness) receiveTransaction() *transaction.ConfigTransaction {
	select {
	case tx := <-h.txSiphon:
		return tx
	case <-time.After(2 * time.Second):
		h.t.Fatal("timeout waiting for transaction")
		return nil
	}
}

// transitionToRunning transitions the runner to Running state if not already there
func (h *testHarness) transitionToRunning() {
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

func testInvalidVersionConfig(t *testing.T, versionValue string) {
	t.Helper()
	// Create a Runner instance
	h := newTestHarness(t, testutil.GetRandomListeningPort(t))

	// Initialize FSM state to Running
	h.transitionToRunning()

	// Set up the invalid version
	invalidConfig := &pb.ServerConfig{
		Version: &versionValue,
	}

	// Convert protobuf to domain config (this will not validate yet)
	domainConfig, err := config.NewFromProto(invalidConfig)
	require.NoError(t, err, "Should be able to create domain config")

	// Run validation which should fail because version is not supported
	err = domainConfig.Validate()
	require.Error(t, err, "Validation should fail")
	require.Contains(
		t,
		err.Error(),
		"unsupported config version",
		"Error should mention unsupported version",
	)
}

// TestGetConfig tests the GetConfig gRPC method
func TestGetConfig(t *testing.T) {
	t.Parallel()

	// Create a Runner instance
	h := newTestHarness(t, testutil.GetRandomListeningPort(t))
	r := h.runner

	// Set a test configuration
	version := version.Version
	testPbConfig := &pb.ServerConfig{
		Version: &version,
	}

	// Convert to domain config
	domainConfig, err := config.NewFromProto(testPbConfig)
	require.NoError(t, err)

	// Create a transaction and store it in the mock storage
	tx, err := transaction.FromAPI("test-request-id", domainConfig, slog.Default().Handler())
	require.NoError(t, err)
	h.txStorage.SetCurrent(tx)

	// Call GetConfig
	resp, err := r.GetConfig(t.Context(), &pb.GetConfigRequest{})

	// Verify response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Check basic fields to avoid proto internals comparison issues
	assert.Equal(t, *testPbConfig.Version, *resp.Config.Version)
}

// TestGetConfigClone tests the GetConfigClone method
func TestGetConfigClone(t *testing.T) {
	t.Parallel()

	t.Run("normal_case", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Set a test configuration
		version := version.Version
		testPbConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Convert to domain config
		domainConfig, err := config.NewFromProto(testPbConfig)
		require.NoError(t, err)

		// Create a transaction and store it
		tx, err := transaction.FromAPI("test-request-id", domainConfig, slog.Default().Handler())
		require.NoError(t, err)
		h.txStorage.SetCurrent(tx)

		// Get a clone of the config
		result := r.GetPbConfigClone()
		require.NotNil(t, result)
		// Check basic fields to avoid proto internals comparison issues
		assert.Equal(t, *testPbConfig.Version, *result.Version)

		// Modify the transaction's config to ensure the clone is independent
		// This tests that GetPbConfigClone returns a proper clone
		tx.GetConfig().Version = "v999"

		// The cloned result should still have the original value
		assert.Equal(t, version, *result.Version)
	})

	t.Run("with_nil_config", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Get config clone should return a default config, not nil
		cfg := r.GetPbConfigClone()
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Version)
		assert.Equal(t, config.VersionLatest, *cfg.Version)
	})
}

// TestUpdateConfig tests the UpdateConfig method in various scenarios
func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid_config", func(t *testing.T) {
		// Create a Runner instance first
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Create valid update request
		version := version.Version
		listenerId := "http_listener"
		listenerAddr := ":8080"
		validConfig := &pb.ServerConfig{
			Version: &version, // Keep v1 which is valid
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
		}
		validReq := &pb.UpdateConfigRequest{
			Config: validConfig,
		}

		// Call UpdateConfig with valid config
		validResp, err := r.UpdateConfig(t.Context(), validReq)

		// Should succeed
		require.NoError(t, err, "Valid config should not cause error")
		assert.NotNil(t, validResp)
		assert.NotNil(t, validResp.Success)
		assert.True(t, *validResp.Success, "Success should be true for valid config")
		// Check basic fields to avoid proto internals comparison issues
		assert.Equal(t, *validConfig.Version, *validResp.Config.Version)
		assert.Equal(t, len(validConfig.Listeners), len(validResp.Config.Listeners))

		// Verify transaction was broadcast via the siphon
		tx := h.receiveTransaction()
		require.NotNil(t, tx, "Should have received transaction via siphon")
		require.NotNil(t, tx.GetConfig())
		assert.Equal(t, version, tx.GetConfig().Version)
		assert.Equal(t, 1, len(tx.GetConfig().Listeners))

		// Note: The config is NOT stored in txStorage by the runner itself.
		// That's the job of the transaction manager after processing the transaction.
		// So we should NOT expect h.txStorage.GetCurrent() to return anything here.
	})

	t.Run("nil_config", func(t *testing.T) {
		// Create a Runner instance
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Call UpdateConfig with nil request
		resp, err := r.UpdateConfig(t.Context(), &pb.UpdateConfigRequest{
			Config: nil,
		})

		// Verify response gets a gRPC InvalidArgument error
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok, "Error should be a gRPC status error")
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Nil(t, resp)
	})

	t.Run("invalid_version", func(t *testing.T) {
		testInvalidVersionConfig(t, "v2")
	})

	t.Run("invalid_format", func(t *testing.T) {
		testInvalidVersionConfig(t, "invalid-version")
	})

	t.Run("multiple_updates", func(t *testing.T) {
		// Create a Runner instance
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Prepare test configs
		version := version.Version
		configs := []*pb.ServerConfig{
			{
				Version: &version,
				Listeners: []*pb.Listener{
					{
						Id:      proto.String("listener_A"),
						Address: proto.String(":8080"),
						Type:    pb.Listener_TYPE_HTTP.Enum(),
						ProtocolOptions: &pb.Listener_Http{
							Http: &pb.HttpListenerOptions{},
						},
					},
				},
			},
			{
				Version: &version,
				Listeners: []*pb.Listener{
					{
						Id:      proto.String("listener_B"),
						Address: proto.String(":8081"),
						Type:    pb.Listener_TYPE_HTTP.Enum(),
						ProtocolOptions: &pb.Listener_Http{
							Http: &pb.HttpListenerOptions{},
						},
					},
				},
			},
		}

		// Make multiple updates
		var receivedTxs []*transaction.ConfigTransaction
		for i, cfg := range configs {
			t.Logf("Updating config %d", i)
			req := &pb.UpdateConfigRequest{Config: cfg}
			resp, err := r.UpdateConfig(t.Context(), req)

			require.NoError(t, err, "Update %d should succeed", i)
			assert.NotNil(t, resp)
			assert.True(t, *resp.Success)

			// Receive the transaction for this update
			tx := h.receiveTransaction()
			require.NotNil(t, tx)
			receivedTxs = append(receivedTxs, tx)
		}

		// Verify we received all transactions
		assert.Equal(t, len(configs), len(receivedTxs), "Should have received all transactions")
	})

	t.Run("validation_failure", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Create a config that will definitely fail validation
		// Use an unsupported version which should cause hard validation failure
		invalidVersion := "v999"
		invalidConfig := &pb.ServerConfig{
			Version: &invalidVersion, // Unsupported version
		}
		req := &pb.UpdateConfigRequest{Config: invalidConfig}

		// Call UpdateConfig
		resp, err := r.UpdateConfig(t.Context(), req)

		// Should return success false with validation error
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Success)
		assert.False(t, *resp.Success)
		require.NotNil(t, resp.Error)
		assert.Contains(t, *resp.Error, "validation failed")
	})

	t.Run("transaction_id_in_success_response", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Create valid update request
		version := version.Version
		listenerId := "http_listener"
		listenerAddr := ":8080"
		validConfig := &pb.ServerConfig{
			Version: &version,
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
		}
		req := &pb.UpdateConfigRequest{Config: validConfig}

		// Call UpdateConfig
		resp, err := r.UpdateConfig(t.Context(), req)

		// Should succeed with transaction ID
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, *resp.Success)
		require.NotNil(t, resp.TransactionId)
		assert.NotEmpty(t, *resp.TransactionId)

		// Verify the transaction ID matches the actual transaction
		tx := h.receiveTransaction()
		require.NotNil(t, tx)
		assert.Equal(t, tx.ID.String(), *resp.TransactionId)
	})

	t.Run("transaction_id_in_validation_error_response", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Initialize FSM state to Running
		h.transitionToRunning()

		// Create invalid config
		invalidVersion := "v999"
		invalidConfig := &pb.ServerConfig{
			Version: &invalidVersion,
		}
		req := &pb.UpdateConfigRequest{Config: invalidConfig}

		// Call UpdateConfig
		resp, err := r.UpdateConfig(t.Context(), req)

		// Should return failure with transaction ID
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, *resp.Success)
		require.NotNil(t, resp.TransactionId)
		assert.NotEmpty(t, *resp.TransactionId)
	})

	t.Run("transaction_id_in_context_cancelled_response", func(t *testing.T) {
		// Create a test harness with unbuffered siphon to force blocking
		txSiphon := make(chan *transaction.ConfigTransaction) // unbuffered
		txStorage := newMockTxStorage()
		runner, err := NewRunner(
			testutil.GetRandomListeningPort(t),
			txSiphon,
			WithConfigTransactionStorage(txStorage),
		)
		require.NoError(t, err)

		// Initialize FSM state to Running
		if runner.fsm.GetState() == finitestate.StatusNew {
			err = runner.fsm.Transition(finitestate.StatusBooting)
			require.NoError(t, err)
		}
		err = runner.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Create valid config
		version := version.Version
		validConfig := &pb.ServerConfig{
			Version: &version,
		}
		req := &pb.UpdateConfigRequest{Config: validConfig}

		// Create a context that will be cancelled
		ctx, cancel := context.WithCancel(t.Context())

		// Call UpdateConfig in a goroutine since it will block on siphon send
		respCh := make(chan *pb.UpdateConfigResponse, 1)
		errCh := make(chan error, 1)
		go func() {
			resp, err := runner.UpdateConfig(ctx, req)
			respCh <- resp
			errCh <- err
		}()

		// Cancel the context after a small delay to ensure we reach the select statement
		time.Sleep(10 * time.Millisecond)
		cancel()

		// Get the response
		var resp *pb.UpdateConfigResponse
		var updateErr error
		select {
		case resp = <-respCh:
			updateErr = <-errCh
		case <-time.After(1 * time.Second):
			t.Fatal("UpdateConfig did not complete in time")
		}

		// Should return context cancelled error with transaction ID
		require.NoError(t, updateErr)
		require.NotNil(t, resp)
		assert.False(t, *resp.Success)
		require.NotNil(t, resp.Error)
		assert.Contains(t, *resp.Error, "service shutting down")
		require.NotNil(t, resp.TransactionId)
		assert.NotEmpty(t, *resp.TransactionId)
	})
}

// TestUpdateConfigWithLogger tests that logger is correctly used during configuration updates
func TestUpdateConfigWithLogger(t *testing.T) {
	t.Parallel()
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// Create Runner with custom logger
	h := newTestHarness(t, testutil.GetRandomListeningPort(t), WithLogger(logger))
	r := h.runner
	defer h.cancel()

	err := r.fsm.Transition(finitestate.StatusBooting)
	require.NoError(t, err)
	err = r.fsm.Transition(finitestate.StatusRunning)
	require.NoError(t, err)

	// Create valid config for update
	version := "v1"
	validConfig := &pb.ServerConfig{
		Version: &version,
		Listeners: []*pb.Listener{
			{
				Id:      proto.String("test_listener"),
				Address: proto.String(":8080"),
				Type:    pb.Listener_TYPE_HTTP.Enum(),
				ProtocolOptions: &pb.Listener_Http{
					Http: &pb.HttpListenerOptions{},
				},
			},
		},
	}

	// Submit the update
	req := &pb.UpdateConfigRequest{Config: validConfig}
	resp, err := r.UpdateConfig(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, *resp.Success)

	// Verify logger was used by checking that the output buffer contains something
	assert.NotEmpty(t, buf.String(), "Logger should have output something")

	// Verify transaction was broadcast via the siphon
	tx := h.receiveTransaction()
	require.NotNil(t, tx, "Should have received transaction via siphon")
}

// TestValidateConfig tests the ValidateConfig gRPC method
func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid_config", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		h.transitionToRunning()

		version := version.Version
		listenerId := "http_listener"
		listenerAddr := ":8080"
		validConfig := &pb.ServerConfig{
			Version: &version,
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
		}
		req := &pb.ValidateConfigRequest{Config: validConfig}

		resp, err := r.ValidateConfig(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Valid)
		assert.True(t, *resp.Valid)
		assert.Nil(t, resp.Error)

		assert.Never(t, func() bool {
			<-h.txSiphon
			return true
		}, 10*time.Millisecond, time.Millisecond)
	})

	t.Run("nil_config", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()

		req := &pb.ValidateConfigRequest{Config: nil}
		resp, err := r.ValidateConfig(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Valid)
		assert.False(t, *resp.Valid)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "No configuration provided", *resp.Error)
	})

	t.Run("invalid_version", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()

		invalidVersion := "v999"
		invalidConfig := &pb.ServerConfig{
			Version: &invalidVersion,
		}
		req := &pb.ValidateConfigRequest{Config: invalidConfig}

		resp, err := r.ValidateConfig(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Valid)
		assert.False(t, *resp.Valid)
		require.NotNil(t, resp.Error)
		assert.Contains(t, *resp.Error, "validation failed")
		assert.Contains(t, *resp.Error, "unsupported config version")
	})

	t.Run("invalid_format_version", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()

		invalidVersion := "invalid-version-format"
		invalidConfig := &pb.ServerConfig{
			Version: &invalidVersion,
		}
		req := &pb.ValidateConfigRequest{Config: invalidConfig}

		resp, err := r.ValidateConfig(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Valid)
		assert.False(t, *resp.Valid)
		require.NotNil(t, resp.Error)
		assert.Contains(t, *resp.Error, "validation failed")
	})

	t.Run("config_with_duplicate_listener_ids", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()

		version := version.Version
		duplicateId := "same_id"
		invalidConfig := &pb.ServerConfig{
			Version: &version,
			Listeners: []*pb.Listener{
				{
					Id:      &duplicateId,
					Address: proto.String(":8080"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
				{
					Id:      &duplicateId,
					Address: proto.String(":8081"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
			},
		}
		req := &pb.ValidateConfigRequest{Config: invalidConfig}

		resp, err := r.ValidateConfig(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Valid)
		assert.False(t, *resp.Valid)
		require.NotNil(t, resp.Error)
		assert.Contains(t, *resp.Error, "validation failed")
	})

	t.Run("multiple_validations", func(t *testing.T) {
		h := newTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()

		version := version.Version
		configs := []struct {
			name        string
			config      *pb.ServerConfig
			expectValid bool
		}{
			{
				name: "valid_config_1",
				config: &pb.ServerConfig{
					Version: &version,
					Listeners: []*pb.Listener{
						{
							Id:      proto.String("listener_A"),
							Address: proto.String(":8080"),
							Type:    pb.Listener_TYPE_HTTP.Enum(),
							ProtocolOptions: &pb.Listener_Http{
								Http: &pb.HttpListenerOptions{},
							},
						},
					},
				},
				expectValid: true,
			},
			{
				name: "invalid_config",
				config: &pb.ServerConfig{
					Version: proto.String("v2"),
				},
				expectValid: false,
			},
			{
				name: "valid_config_2",
				config: &pb.ServerConfig{
					Version: &version,
					Listeners: []*pb.Listener{
						{
							Id:      proto.String("listener_B"),
							Address: proto.String(":8081"),
							Type:    pb.Listener_TYPE_HTTP.Enum(),
							ProtocolOptions: &pb.Listener_Http{
								Http: &pb.HttpListenerOptions{},
							},
						},
					},
				},
				expectValid: true,
			},
		}

		for _, tc := range configs {
			t.Run(tc.name, func(t *testing.T) {
				req := &pb.ValidateConfigRequest{Config: tc.config}
				resp, err := r.ValidateConfig(t.Context(), req)

				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.Valid)
				assert.Equal(t, tc.expectValid, *resp.Valid)

				if !tc.expectValid {
					assert.NotNil(t, resp.Error)
				} else {
					assert.Nil(t, resp.Error)
				}
			})
		}

		assert.Never(t, func() bool {
			<-h.txSiphon
			return true
		}, 10*time.Millisecond, time.Millisecond)
	})
}

// TestValidateConfigDoesNotModifyState tests that ValidateConfig does not modify server state
func TestValidateConfigDoesNotModifyState(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t, testutil.GetRandomListeningPort(t))
	r := h.runner
	defer h.cancel()

	h.transitionToRunning()

	version := version.Version
	initialConfig := &pb.ServerConfig{
		Version: &version,
		Listeners: []*pb.Listener{
			{
				Id:      proto.String("initial_listener"),
				Address: proto.String(":9000"),
				Type:    pb.Listener_TYPE_HTTP.Enum(),
				ProtocolOptions: &pb.Listener_Http{
					Http: &pb.HttpListenerOptions{},
				},
			},
		},
	}

	domainConfig, err := config.NewFromProto(initialConfig)
	require.NoError(t, err)
	tx, err := transaction.FromAPI("initial-setup", domainConfig, slog.Default().Handler())
	require.NoError(t, err)
	h.txStorage.SetCurrent(tx)

	initialState := r.GetDomainConfig()

	newConfig := &pb.ServerConfig{
		Version: &version,
		Listeners: []*pb.Listener{
			{
				Id:      proto.String("new_listener"),
				Address: proto.String(":8080"),
				Type:    pb.Listener_TYPE_HTTP.Enum(),
				ProtocolOptions: &pb.Listener_Http{
					Http: &pb.HttpListenerOptions{},
				},
			},
		},
	}
	req := &pb.ValidateConfigRequest{Config: newConfig}

	resp, err := r.ValidateConfig(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, *resp.Valid)

	afterState := r.GetDomainConfig()
	assert.Equal(t, initialState.Version, afterState.Version)
	assert.Equal(t, len(initialState.Listeners), len(afterState.Listeners))
	if len(initialState.Listeners) > 0 && len(afterState.Listeners) > 0 {
		assert.Equal(t, initialState.Listeners[0].ID, afterState.Listeners[0].ID)
	}

	assert.Never(t, func() bool {
		<-h.txSiphon
		return true
	}, 10*time.Millisecond, time.Millisecond)
}

// TestHandlingInvalidVersionConfig tests configs with invalid versions
func TestHandlingInvalidVersionConfig(t *testing.T) {
	t.Parallel()
	// Create a domain config with an invalid version
	invalidVersion := "invalid_version_format"
	pbConfig := &pb.ServerConfig{
		Version: &invalidVersion,
	}

	// Convert protobuf to domain config (this will not validate yet)
	domainConfig, err := config.NewFromProto(pbConfig)
	require.NoError(t, err, "Should be able to create domain config")

	// Run validation which should fail because it's an unsupported version
	err = domainConfig.Validate()
	require.Error(t, err, "Validation should fail for invalid version")
	require.Contains(
		t, err.Error(),
		"unsupported config version",
		"Error should mention unsupported version",
	)
}
