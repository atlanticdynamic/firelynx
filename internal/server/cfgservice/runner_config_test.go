package cfgservice

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// TestGetConfig tests the GetConfig gRPC method
func TestGetConfig(t *testing.T) {
	t.Parallel()

	// Create a Runner instance
	r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
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

// TestGetConfigClone tests the GetConfigClone method
func TestGetConfigClone(t *testing.T) {
	t.Parallel()

	t.Run("normal_case", func(t *testing.T) {
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Set a test configuration
		version := "v1"
		testConfig := &pb.ServerConfig{
			Version: &version,
		}
		r.configMu.Lock()
		r.config = testConfig
		r.configMu.Unlock()

		// Get a clone of the config
		result := r.GetConfigClone()
		require.NotNil(t, result)
		assert.Equal(t, testConfig, result)

		// Change a value in the original config and confirm the clone doesn't change
		newVersion := "v999"
		testConfig.Version = &newVersion
		assert.NotEqual(t, testConfig, result)
		assert.Equal(t, version, *result.Version)
	})

	t.Run("with_nil_config", func(t *testing.T) {
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Ensure config is nil
		r.configMu.Lock()
		r.config = nil
		r.configMu.Unlock()

		// Get config clone should return a default config, not nil
		cfg := r.GetConfigClone()
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Version)
	})
}

// TestUpdateConfig tests the UpdateConfig method in various scenarios
func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid_config", func(t *testing.T) {
		// Create a Runner instance
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Set initial version
		version := "v1"
		initialConfig := &pb.ServerConfig{
			Version: &version,
		}
		r.configMu.Lock()
		r.config = initialConfig
		r.configMu.Unlock()

		// Create valid update request
		listenerId := "http_listener"
		listenerAddr := ":8080"
		validConfig := &pb.ServerConfig{
			Version: &version, // Keep v1 which is valid
			Listeners: []*pb.Listener{
				{
					Id:      &listenerId,
					Address: &listenerAddr,
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
		result := r.GetConfigClone()
		assert.Equal(t, validConfig, result, "Config should be updated after successful validation")
	})

	t.Run("nil_config", func(t *testing.T) {
		// Create a Runner instance
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
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
	})

	t.Run("validation_error", func(t *testing.T) {
		// Create a Runner instance
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Set initial version
		version := "v1"
		initialConfig := &pb.ServerConfig{
			Version: &version,
		}
		r.configMu.Lock()
		r.config = initialConfig
		r.configMu.Unlock()

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
	})

	t.Run("invalid_version", func(t *testing.T) {
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Create a config with an invalid version that will fail validation
		invalidVersion := "invalid-version"
		invalidConfig := &pb.ServerConfig{
			Version: &invalidVersion,
		}

		req := &pb.UpdateConfigRequest{
			Config: invalidConfig,
		}

		// Update should fail with validation error
		resp, err := r.UpdateConfig(context.Background(), req)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "validation error")
		assert.Nil(t, resp)
	})

	t.Run("response_isolation", func(t *testing.T) {
		// This test verifies that when we modify the response config, it doesn't affect
		// the internal stored config, which demonstrates proper deep copying
		r, err := New(WithListenAddr(testutil.GetRandomListeningPort(t)))
		require.NoError(t, err)

		// Start with a valid config
		version := "v1"
		updateConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Create update request
		req := &pb.UpdateConfigRequest{
			Config: updateConfig,
		}

		// Update config
		resp, err := r.UpdateConfig(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Now modify the response config - this should not affect the stored config
		newVersion := "v999" // An invalid version
		resp.Config.Version = &newVersion

		// Get the stored config
		storedConfig := r.GetConfigClone()

		// Check that it still has the original valid version
		assert.Equal(
			t,
			version,
			*storedConfig.Version,
			"Stored config should not be affected by changes to the response config",
		)
	})

	t.Run("multiple_updates", func(t *testing.T) {
		// Create a custom logger that won't print warnings during tests
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		r, err := New(
			WithListenAddr(testutil.GetRandomListeningPort(t)),
			WithLogger(logger),
		)
		require.NoError(t, err)

		// Get the reload channel
		reloadCh := r.GetReloadTrigger()

		// Create initial valid config
		version := "v1"
		initialConfig := &pb.ServerConfig{
			Version: &version,
		}

		// Create a context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create three configurations to update with
		configs := []*pb.ServerConfig{
			proto.Clone(initialConfig).(*pb.ServerConfig),
			proto.Clone(initialConfig).(*pb.ServerConfig),
			proto.Clone(initialConfig).(*pb.ServerConfig),
		}

		// Add different listeners to each config to make them distinguishable
		for i, cfg := range configs {
			id := "listener_" + string(rune('A'+i))
			addr := ":808" + string(rune('0'+i))
			cfg.Listeners = []*pb.Listener{
				{
					Id:      &id,
					Address: &addr,
				},
			}
		}

		// Update the config multiple times rapidly
		for i, cfg := range configs {
			t.Logf("Updating config %d", i)
			req := &pb.UpdateConfigRequest{
				Config: cfg,
			}

			resp, err := r.UpdateConfig(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, *resp.Success)
		}

		// Check if at least one notification was received
		select {
		case <-reloadCh:
			// Success - reload notification received
		case <-ctx.Done():
			t.Fatal("Timeout waiting for reload notification")
		}

		// Check if the final config is correctly stored
		storedConfig := r.GetConfigClone()
		assert.Equal(t, configs[2].Listeners[0].Id, storedConfig.Listeners[0].Id,
			"Final config should match the last update")
	})
}
