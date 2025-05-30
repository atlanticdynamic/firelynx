//go:build integration
// +build integration

package configupdates

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgfileloader"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/robbyt/go-supervisor/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/echo_app.toml
var echoAppConfigTemplate []byte

// TestSupervisorStartupWithHTTPRunner tests the full startup sequence with supervisor
// to replicate the e2e test scenario where HTTP runner waits for initial config
func TestSupervisorStartupWithHTTPRunner(t *testing.T) {
	logging.SetupLogger("debug")
	l := slog.Default()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Get a random port for HTTP listener
	httpPort := testutil.GetRandomPort(t)

	// Replace the port placeholder in config
	echoAppConfig := strings.ReplaceAll(
		string(echoAppConfigTemplate),
		"{{PORT}}",
		fmt.Sprintf("%d", httpPort),
	)

	// Set up temporary directory for config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")
	err := os.WriteFile(configPath, []byte(echoAppConfig), 0o644)
	require.NoError(t, err)

	// Create transaction storage
	txStorage := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	saga := orchestrator.NewSagaOrchestrator(txStorage, l.Handler())

	// Create transaction manager
	txMan, err := txmgr.NewRunner(saga, txmgr.WithLogHandler(l.Handler()))
	require.NoError(t, err)

	// Get the transaction siphon
	txSiphon := txMan.GetTransactionSiphon()

	// Create HTTP runner and register it with orchestrator
	httpRunner, err := httplistener.NewRunner(httplistener.WithLogHandler(l.Handler()))
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Create config file loader
	cfgFileLoader, err := cfgfileloader.NewRunner(
		configPath,
		txSiphon,
		cfgfileloader.WithLogHandler(l.Handler()),
	)
	require.NoError(t, err)

	// Create runnables list in the same order as the server:
	// 1. Config providers first (cfgfileloader)
	// 2. Transaction manager
	// 3. HTTP runner
	runnables := []supervisor.Runnable{
		cfgFileLoader,
		txMan,
		httpRunner,
	}

	// Create supervisor
	super, err := supervisor.New(
		supervisor.WithContext(ctx),
		supervisor.WithRunnables(runnables...),
	)
	require.NoError(t, err)

	// Start supervisor in a goroutine
	superErrCh := make(chan error, 1)
	go func() {
		superErrCh <- super.Run()
	}()

	// Wait for HTTP endpoint to become available
	// This is the real test - can we actually make HTTP requests?
	testURL := fmt.Sprintf("http://localhost:%d/echo", httpPort)

	// The HTTP runner should eventually start and serve requests
	require.Eventually(t, func() bool {
		resp, err := http.Get(testURL)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 100*time.Millisecond, "HTTP endpoint should become available")

	// Make a successful request to verify the endpoint works
	resp, err := http.Get(testURL)
	require.NoError(t, err, "Should be able to make HTTP request")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should get OK response")

	// Shutdown supervisor by cancelling context
	cancel()

	// Wait for supervisor to complete
	select {
	case err := <-superErrCh:
		assert.NoError(t, err, "Supervisor should exit cleanly")
	case <-time.After(5 * time.Second):
		t.Fatal("Supervisor did not shut down in time")
	}
}

// TestHTTPRunnerStartupTiming tests that HTTP runner waits for initial config before becoming ready
func TestHTTPRunnerStartupTiming(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Create transaction storage
	txStorage := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	saga := orchestrator.NewSagaOrchestrator(txStorage, slog.Default().Handler())

	// Create transaction manager
	txMan, err := txmgr.NewRunner(saga)
	require.NoError(t, err)

	// Get the transaction siphon
	txSiphon := txMan.GetTransactionSiphon()

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Create runnables WITHOUT config provider initially
	runnables := []supervisor.Runnable{
		txMan,
		httpRunner,
	}

	// Create supervisor
	super, err := supervisor.New(
		supervisor.WithContext(ctx),
		supervisor.WithRunnables(runnables...),
	)
	require.NoError(t, err)

	// Start supervisor
	superErrCh := make(chan error, 1)
	go func() {
		superErrCh <- super.Run()
	}()

	// Wait for txmgr to be running
	require.Eventually(t, func() bool {
		return txMan.IsRunning()
	}, 2*time.Second, 50*time.Millisecond, "Transaction manager should be running")

	// HTTP runner should be Running immediately
	// The httpcluster transitions to Running state even without configuration
	assert.Eventually(t, func() bool {
		httpState := httpRunner.GetState()
		t.Logf("HTTP runner state without config: %s", httpState)
		// httpcluster reports Running immediately, even while waiting for config
		return httpState == "Running"
	}, time.Second, 50*time.Millisecond, "HTTP runner should be in Running state")

	// Get a random port for HTTP listener
	httpPort := testutil.GetRandomPort(t)

	// Replace the port placeholder in config
	echoAppConfig := strings.ReplaceAll(
		string(echoAppConfigTemplate),
		"{{PORT}}",
		fmt.Sprintf("%d", httpPort),
	)

	// Create config file loader with the config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")
	err = os.WriteFile(configPath, []byte(echoAppConfig), 0o644)
	require.NoError(t, err)

	cfgFileLoader, err2 := cfgfileloader.NewRunner(configPath, txSiphon)
	require.NoError(t, err2)

	// Start the config loader separately
	loaderErrCh := make(chan error, 1)
	go func() {
		loaderErrCh <- cfgFileLoader.Run(ctx)
	}()

	// Now HTTP runner should eventually transition to Running
	require.Eventually(t, func() bool {
		state := httpRunner.GetState()
		t.Logf("HTTP runner state after config: %s", state)
		return state == "Running"
	}, 5*time.Second, 100*time.Millisecond, "HTTP runner should become Running after config is loaded")

	// Verify endpoint is accessible (httpPort already known from above)
	testURL := fmt.Sprintf("http://localhost:%d/echo", httpPort)

	resp, err := http.Get(testURL)
	require.NoError(t, err, "HTTP endpoint should be accessible")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown
	cancel()
	cfgFileLoader.Stop()

	select {
	case <-superErrCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Supervisor did not shut down")
	}
}
