package configupdates

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgfileloader"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
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

// integrationTestHarness provides setup for integration tests with the siphon pattern
type integrationTestHarness struct {
	t        *testing.T
	txmgr    *txmgr.Runner
	txSiphon chan<- *transaction.ConfigTransaction
	ctx      context.Context
	cancel   context.CancelFunc
}

// newIntegrationTestHarness creates a test harness with txmgr and siphon
func newIntegrationTestHarness(t *testing.T) *integrationTestHarness {
	t.Helper()

	// Create transaction manager
	storage := txstorage.NewMemoryStorage()
	handler := slog.Default().Handler()
	orchestr := orchestrator.NewSagaOrchestrator(storage, handler)
	txMan, err := txmgr.NewRunner(orchestr)
	require.NoError(t, err)

	// Get the transaction siphon
	txSiphon := txMan.GetTransactionSiphon()

	ctx, cancel := context.WithCancel(t.Context())

	return &integrationTestHarness{
		t:        t,
		txmgr:    txMan,
		txSiphon: txSiphon,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// startTxMgr starts the transaction manager
func (h *integrationTestHarness) startTxMgr() <-chan error {
	return startRunnerAsync(h.ctx, h.txmgr)
}

// TestConfigChannels_BasicFlow tests that config updates flow through the siphon correctly
func TestConfigChannels_BasicFlow(t *testing.T) {
	t.Parallel()

	// Create test harness with txmgr and siphon
	h := newIntegrationTestHarness(t)
	defer h.cancel()

	// Start transaction manager
	_ = h.startTxMgr()

	// Wait for txmgr to be ready
	assert.Eventually(t, func() bool {
		return h.txmgr.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Create cfgservice runner with siphon
	cfgServiceRunner, err := cfgservice.NewRunner(testutil.GetRandomListeningPort(t), h.txSiphon)
	require.NoError(t, err)

	// Start the cfgservice runner
	_ = startRunnerAsync(h.ctx, cfgServiceRunner)

	// Wait for cfgservice runner to be ready
	assert.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Test a simple config update to verify the flow works
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

	// Send update request
	req := &pb.UpdateConfigRequest{Config: testConfig}
	resp, err := cfgServiceRunner.UpdateConfig(h.ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, *resp.Success)
}

// TestConfigChannels_CfgFileLoaderIntegration tests cfgfileloader with siphon integration
func TestConfigChannels_CfgFileLoaderIntegration(t *testing.T) {
	t.Parallel()

	// Create test harness with txmgr and siphon
	h := newIntegrationTestHarness(t)
	defer h.cancel()

	// Start transaction manager
	_ = h.startTxMgr()

	// Wait for txmgr to be ready
	assert.Eventually(t, func() bool {
		return h.txmgr.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.toml")

	// Write initial config from embedded file
	err := os.WriteFile(configFile, initialConfigTOML, 0o644)
	require.NoError(t, err)

	// Create cfgfileloader runner with siphon
	fileLoader, err := cfgfileloader.NewRunner(
		configFile,
		h.txSiphon,
	)
	require.NoError(t, err)

	// Start the runner
	_ = startRunnerAsync(h.ctx, fileLoader)

	// Wait for runner to reach running state and process initial config
	assert.Eventually(t, func() bool {
		return fileLoader.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Update config file to trigger reload
	err = os.WriteFile(configFile, updatedConfigTOML, 0o644)
	require.NoError(t, err)

	// Trigger reload
	fileLoader.Reload()

	// Verify config was updated (simple check that reload worked)
	assert.Eventually(t, func() bool {
		return fileLoader.IsRunning() // Still running after reload
	}, 1*time.Second, 10*time.Millisecond)
}

// TestConfigChannels_MultipleUpdates tests multiple config updates through siphon
func TestConfigChannels_MultipleUpdates(t *testing.T) {
	t.Parallel()

	// Create test harness with txmgr and siphon
	h := newIntegrationTestHarness(t)
	defer h.cancel()

	// Start transaction manager
	_ = h.startTxMgr()

	// Wait for txmgr to be ready
	assert.Eventually(t, func() bool {
		return h.txmgr.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Create cfgservice runner with siphon
	cfgServiceRunner, err := cfgservice.NewRunner(testutil.GetRandomListeningPort(t), h.txSiphon)
	require.NoError(t, err)

	// Start the cfgservice runner
	_ = startRunnerAsync(h.ctx, cfgServiceRunner)

	// Wait for cfgservice runner to be ready
	assert.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, 2*time.Second, 10*time.Millisecond)

	// Test multiple sequential config updates
	version := "v1"
	for i := 0; i < 3; i++ {
		testConfig := &pb.ServerConfig{
			Version: &version,
			Listeners: []*pb.Listener{
				{
					Id:      proto.String("test_listener_" + string(rune('1'+i))),
					Address: proto.String(":808" + string(rune('0'+i))),
					Type:    pb.ListenerType_LISTENER_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
			},
		}

		// Send update request
		req := &pb.UpdateConfigRequest{Config: testConfig}
		resp, err := cfgServiceRunner.UpdateConfig(h.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, *resp.Success)
	}
}
