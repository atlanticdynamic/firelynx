package integration_tests

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgfileloader"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

//go:embed testdata/initial_config.toml
var initialConfigTOML []byte

//go:embed testdata/updated_config.toml
var updatedConfigTOML []byte

// startRunnerAsync starts a runner in a goroutine and returns an error channel
func startRunnerAsync(
	ctx context.Context,
	runner interface{ Run(context.Context) error },
) <-chan error {
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- runner.Run(ctx)
	}()
	return runErrCh
}

// TestConfigChannels_UnbufferedBackpressure tests that unbuffered channels provide proper back-pressure
func TestConfigChannels_UnbufferedBackpressure(t *testing.T) {
	t.Parallel()

	// Create cfgservice runner with minimal config
	cfgServiceRunner, err := cfgservice.NewRunner(testutil.GetRandomListeningPort(t))
	require.NoError(t, err)

	// Start the runner
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	runErrCh := startRunnerAsync(ctx, cfgServiceRunner)

	// Wait for runner to be ready
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Get a config channel from cfgservice
	configChan := cfgServiceRunner.GetConfigChan()

	// Start the config consumer that will intentionally block
	var firstTxReceived sync.WaitGroup
	var allowConsumerToContinue sync.WaitGroup
	firstTxReceived.Add(1)
	allowConsumerToContinue.Add(1)

	var receivedCount int32

	go func() {
		for tx := range configChan {
			if tx != nil {
				count := atomic.AddInt32(&receivedCount, 1)
				if count == 1 {
					firstTxReceived.Done()
					// Block here to test back-pressure
					allowConsumerToContinue.Wait()
				}
			}
		}
	}()

	// Create test configs for multiple updates
	version := "v1"
	configs := []*pb.ServerConfig{
		{
			Version: &version,
			Listeners: []*pb.Listener{
				{
					Id:      proto.String("listener_1"),
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
					Id:      proto.String("listener_2"),
					Address: proto.String(":8081"),
					Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
			},
		},
	}

	// Send first update (this should succeed and reach consumer)
	req1 := &pb.UpdateConfigRequest{Config: configs[0]}
	updateDone := make(chan error, 1)
	go func() {
		_, err := cfgServiceRunner.UpdateConfig(ctx, req1)
		updateDone <- err
	}()

	// Wait for first transaction to reach consumer
	firstTxReceived.Wait()

	// Send second update (this should block due to backpressure since consumer is blocked)
	req2 := &pb.UpdateConfigRequest{Config: configs[1]}
	secondUpdateStarted := make(chan struct{})
	secondUpdateDone := make(chan error, 1)
	go func() {
		close(secondUpdateStarted)
		_, err := cfgServiceRunner.UpdateConfig(ctx, req2)
		secondUpdateDone <- err
	}()

	// Wait for second update to start
	<-secondUpdateStarted

	// First update should complete quickly
	assert.Eventually(t, func() bool {
		select {
		case err := <-updateDone:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 200*time.Millisecond, 10*time.Millisecond, "First update should complete")

	// Second update should still be blocked
	assert.Never(t, func() bool {
		select {
		case <-secondUpdateDone:
			return true
		default:
			return false
		}
	}, 200*time.Millisecond, 10*time.Millisecond, "Second update should be blocked")

	// Now allow consumer to continue
	allowConsumerToContinue.Done()

	// Second update should now complete
	assert.Eventually(t, func() bool {
		select {
		case err := <-secondUpdateDone:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 500*time.Millisecond, 10*time.Millisecond, "Second update should complete after consumer unblocks")

	// Verify we received both transactions
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&receivedCount) == 2
	}, 500*time.Millisecond, 10*time.Millisecond, "Should have received both transactions")

	// Clean up
	cfgServiceRunner.Stop()
	assert.Eventually(t, func() bool {
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond, "Runner should stop")
}

// TestConfigChannels_CfgFileLoaderIntegration tests cfgfileloader config channel integration
func TestConfigChannels_CfgFileLoaderIntegration(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.toml")

	// Write initial config from embedded file
	err := os.WriteFile(configFile, initialConfigTOML, 0o644)
	require.NoError(t, err)

	// Create cfgfileloader runner
	fileLoader, err := cfgfileloader.NewRunner(configFile, cfgfileloader.WithContext(t.Context()))
	require.NoError(t, err)

	// Get config channel BEFORE starting the runner
	configChan := fileLoader.GetConfigChan()

	// Start the runner
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	runErrCh := startRunnerAsync(ctx, fileLoader)

	// Collect transactions
	receivedTxs := make(chan *transaction.ConfigTransaction, 10)
	go func() {
		for tx := range configChan {
			receivedTxs <- tx
		}
	}()

	// Wait for runner to reach running state
	require.Eventually(t, func() bool {
		return fileLoader.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Should receive initial config
	select {
	case tx := <-receivedTxs:
		assert.NotNil(t, tx)
		assert.NotNil(t, tx.GetConfig())
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive initial config transaction")
	}

	// Update config file to trigger reload
	err = os.WriteFile(configFile, updatedConfigTOML, 0o644)
	require.NoError(t, err)

	// Trigger reload
	fileLoader.Reload()

	// Should receive updated config
	select {
	case tx := <-receivedTxs:
		assert.NotNil(t, tx)
		cfg := tx.GetConfig()
		assert.NotNil(t, cfg)
		// Verify it's the updated config
		assert.Len(t, cfg.Listeners, 1)
		assert.Equal(t, "http_listener_updated", cfg.Listeners[0].ID)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive updated config transaction")
	}

	// Stop runner
	fileLoader.Stop()
	assert.Eventually(t, func() bool {
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond, "Runner should stop")
}

// TestConfigChannels_MultipleConsumersBackpressure tests backpressure with multiple consumers
func TestConfigChannels_MultipleConsumersBackpressure(t *testing.T) {
	t.Parallel()

	// Create cfgservice runner
	cfgServiceRunner, err := cfgservice.NewRunner(testutil.GetRandomListeningPort(t))
	require.NoError(t, err)

	// Start the runner
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	runErrCh := startRunnerAsync(ctx, cfgServiceRunner)

	// Wait for runner to be ready
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Create multiple config channels (multiple consumers)
	configChan1 := cfgServiceRunner.GetConfigChan()
	configChan2 := cfgServiceRunner.GetConfigChan()

	// Start first consumer (will block after first transaction)
	var consumer1Started, consumer1Received, consumer1Continue sync.WaitGroup
	consumer1Started.Add(1)
	consumer1Received.Add(1)
	consumer1Continue.Add(1)

	go func() {
		consumer1Started.Done()
		// Block BEFORE reading from the channel to create backpressure
		consumer1Continue.Wait()
		for tx := range configChan1 {
			if tx != nil {
				consumer1Received.Done()
			}
		}
	}()

	// Start second consumer (will consume normally)
	var consumer2Count int32
	go func() {
		for tx := range configChan2 {
			if tx != nil {
				atomic.AddInt32(&consumer2Count, 1)
			}
		}
	}()

	// Wait for consumer1 to start
	consumer1Started.Wait()

	// Send config update
	version := "v1"
	testConfig := &pb.ServerConfig{
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

	// This should block because consumer1 will block after receiving the transaction
	updateDone := make(chan error, 1)
	var updateStarted sync.WaitGroup
	updateStarted.Add(1)
	go func() {
		updateStarted.Done()
		req := &pb.UpdateConfigRequest{Config: testConfig}
		_, err := cfgServiceRunner.UpdateConfig(ctx, req)
		updateDone <- err
	}()

	// Ensure update goroutine has started
	updateStarted.Wait()

	// Verify that update remains blocked due to consumer1 backpressure
	assert.Never(t, func() bool {
		select {
		case <-updateDone:
			return true
		default:
			return false
		}
	}, 200*time.Millisecond, 10*time.Millisecond, "Update should be blocked due to consumer1 backpressure")

	// Consumer2 may or may not have received the transaction yet,
	// depending on the order of iteration in sync.Map.Range.
	// What matters is that UpdateConfig is blocked.

	// Allow consumer1 to continue
	consumer1Continue.Done()

	// Wait for consumer1 to receive its transaction
	consumer1Received.Wait()

	// Update should now complete
	assert.Eventually(t, func() bool {
		select {
		case err := <-updateDone:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "Update should complete after consumer1 unblocks")

	// Now consumer2 should also receive the transaction
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&consumer2Count) >= 1
	}, 500*time.Millisecond, 10*time.Millisecond, "Consumer2 should receive transaction")

	// Clean up
	cfgServiceRunner.Stop()
	assert.Eventually(t, func() bool {
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
			return true
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond, "Runner should stop")
}
