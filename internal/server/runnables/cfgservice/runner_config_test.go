package cfgservice

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func testInvalidVersionConfig(t *testing.T, versionValue string) {
	t.Helper()
	// Create a Runner instance
	r, err := NewRunner(
		testutil.GetRandomListeningPort(t),
	)
	require.NoError(t, err)

	// Initialize FSM state to Running
	err = r.fsm.Transition(finitestate.StatusBooting)
	require.NoError(t, err)
	err = r.fsm.Transition(finitestate.StatusRunning)
	require.NoError(t, err)

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
	r, err := NewRunner(
		testutil.GetRandomListeningPort(t),
	)
	require.NoError(t, err)

	// Set a test configuration
	version := version.Version
	testPbConfig := &pb.ServerConfig{
		Version: &version,
	}

	// Convert to domain config
	domainConfig, err := config.NewFromProto(testPbConfig)
	require.NoError(t, err)

	r.lastLoadedCfgMu.Lock()
	r.lastLoadedCfg = domainConfig
	r.lastLoadedCfgMu.Unlock()

	// Call GetConfig
	resp, err := r.GetConfig(context.Background(), &pb.GetConfigRequest{})

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
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
		)
		require.NoError(t, err)

		// Set a test configuration
		version := version.Version
		testPbConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Convert to domain config
		domainConfig, err := config.NewFromProto(testPbConfig)
		require.NoError(t, err)

		r.lastLoadedCfgMu.Lock()
		r.lastLoadedCfg = domainConfig
		r.lastLoadedCfgMu.Unlock()

		// Get a clone of the config
		result := r.GetPbConfigClone()
		require.NotNil(t, result)
		// Check basic fields to avoid proto internals comparison issues
		assert.Equal(t, *testPbConfig.Version, *result.Version)

		// Change a value in the domain config and ensure the clone still has the original value
		r.lastLoadedCfgMu.Lock()
		r.lastLoadedCfg.Version = "v999"
		r.lastLoadedCfgMu.Unlock()

		assert.Equal(t, version, *result.Version)
	})

	t.Run("with_nil_config", func(t *testing.T) {
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
		)
		require.NoError(t, err)

		// Ensure config is nil
		r.lastLoadedCfgMu.Lock()
		r.lastLoadedCfg = nil
		r.lastLoadedCfgMu.Unlock()

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
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
		)
		require.NoError(t, err)

		// Set up a config channel consumer to verify transactions are broadcast
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		configChan := r.GetConfigChan()
		consumer := newConfigChannelConsumer(t, configChan)
		consumer.start(ctx)

		// Set initial version
		version := version.Version
		initialPbConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Convert to domain config
		initialDomainConfig, err := config.NewFromProto(initialPbConfig)
		require.NoError(t, err)

		r.lastLoadedCfgMu.Lock()
		r.lastLoadedCfg = initialDomainConfig
		r.lastLoadedCfgMu.Unlock()

		// Initialize FSM state to Running
		err = r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Create valid update request
		listenerId := "http_listener"
		listenerAddr := ":8080"
		validConfig := &pb.ServerConfig{
			Version: &version, // Keep v1 which is valid
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
		// Check basic fields to avoid proto internals comparison issues
		assert.Equal(t, *validConfig.Version, *validResp.Config.Version)
		assert.Equal(t, len(validConfig.Listeners), len(validResp.Config.Listeners))

		// Verify domain config was stored (not the protobuf)
		r.lastLoadedCfgMu.Lock()
		assert.NotNil(t, r.lastLoadedCfg)
		assert.Equal(t, version, r.lastLoadedCfg.Version)
		r.lastLoadedCfgMu.Unlock()

		// Verify transaction was broadcast via the channel
		tx := consumer.waitForTransaction(500 * time.Millisecond)
		require.NotNil(t, tx, "Should have received transaction via channel")
		require.NotNil(t, tx.GetConfig())
		assert.Equal(t, version, tx.GetConfig().Version)
		assert.Equal(t, 1, len(tx.GetConfig().Listeners))
	})

	t.Run("nil_config", func(t *testing.T) {
		// Create a Runner instance
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
		)
		require.NoError(t, err)

		// Initialize FSM state to Running
		err = r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Call UpdateConfig with nil request
		resp, err := r.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
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
		r, err := NewRunner(
			testutil.GetRandomListeningPort(t),
		)
		require.NoError(t, err)

		// Set up a config channel consumer to verify transactions are broadcast
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		configChan := r.GetConfigChan()
		consumer := newConfigChannelConsumer(t, configChan)
		consumer.start(ctx)

		// Set initial version
		version := version.Version
		initialPbConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Convert to domain config
		initialDomainConfig, err := config.NewFromProto(initialPbConfig)
		require.NoError(t, err)

		r.lastLoadedCfgMu.Lock()
		r.lastLoadedCfg = initialDomainConfig
		r.lastLoadedCfgMu.Unlock()

		// Initialize FSM state to Running
		err = r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Prepare test configs
		configs := []*pb.ServerConfig{
			{
				Version: &version,
				Listeners: []*pb.Listener{
					{
						Id:      proto.String("listener_A"),
						Address: proto.String(":8080"),
						Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
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
						Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
						ProtocolOptions: &pb.Listener_Http{
							Http: &pb.HttpListenerOptions{},
						},
					},
				},
			},
		}

		// Make multiple updates
		for i, cfg := range configs {
			t.Logf("Updating config %d", i)
			req := &pb.UpdateConfigRequest{Config: cfg}
			resp, err := r.UpdateConfig(context.Background(), req)

			require.NoError(t, err, "Update %d should succeed", i)
			assert.NotNil(t, resp)
			assert.True(t, *resp.Success)
		}

		// Verify multiple transactions were broadcast
		assert.Eventually(t, func() bool {
			return consumer.getTransactionCount() == len(configs)
		}, 500*time.Millisecond, 10*time.Millisecond, "Should have received all transactions")
	})
}

// TestReloadNotification_Direct tests the reload notification using direct calls
func TestReloadNotification_Direct(t *testing.T) {
	// Create a simple Runner without complex transactions
	r, err := NewRunner(
		testutil.GetRandomListeningPort(t),
	)
	require.NoError(t, err)

	// Set up the runner to running state
	err = r.fsm.Transition(finitestate.StatusBooting)
	require.NoError(t, err)
	err = r.fsm.Transition(finitestate.StatusRunning)
	require.NoError(t, err)

	// Get the reload channel
	reloadCh := r.GetReloadTrigger()
	require.NotNil(t, reloadCh)

	// Check initial state (should be empty)
	select {
	case <-reloadCh:
		t.Fatal("Reload channel should be empty initially")
	default:
		// Expected - channel is empty
	}

	// Create a simple config
	testConfig := &config.Config{
		Version: "v1",
	}

	// Directly set the config and trigger reload
	r.lastLoadedCfgMu.Lock()
	r.lastLoadedCfg = testConfig
	r.lastLoadedCfgMu.Unlock()

	// Trigger the reload directly
	r.triggerReload()

	// Verify reload notification was sent
	select {
	case <-reloadCh:
		// Success - we got the expected notification
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Did not receive reload notification within expected timeframe")
	}

	// Channel should be drained now
	select {
	case <-reloadCh:
		t.Fatal("Should have been only one notification")
	default:
		// Expected - channel is empty again
	}
}

// TestUpdateConfigWithLogger tests that logger is correctly used during configuration updates
func TestUpdateConfigWithLogger(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// Create Runner with custom logger
	r, err := NewRunner(
		testutil.GetRandomListeningPort(t),
		WithLogger(logger),
	)
	require.NoError(t, err)

	// Set up a config channel consumer to verify transactions are broadcast
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	configChan := r.GetConfigChan()
	consumer := newConfigChannelConsumer(t, configChan)
	consumer.start(ctx)

	err = r.fsm.Transition(finitestate.StatusBooting)
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
				Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
				ProtocolOptions: &pb.Listener_Http{
					Http: &pb.HttpListenerOptions{},
				},
			},
		},
	}

	// Submit the update
	req := &pb.UpdateConfigRequest{Config: validConfig}
	resp, err := r.UpdateConfig(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, *resp.Success)

	// Verify logger was used by checking that the output buffer contains something
	assert.NotEmpty(t, buf.String(), "Logger should have output something")

	// Verify transaction was broadcast via the channel
	tx := consumer.waitForTransaction(500 * time.Millisecond)
	require.NotNil(t, tx, "Should have received transaction via channel")
}

// TestHandlingInvalidVersionConfig tests configs with invalid versions
func TestHandlingInvalidVersionConfig(t *testing.T) {
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
		t,
		err.Error(),
		"unsupported config version",
		"Error should mention unsupported version",
	)
}
