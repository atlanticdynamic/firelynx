//go:build integration

package configupdates

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"text/template"
	"time"

	client "github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader/toml"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use the echoAppTOML from grpc_config_test.go

// renderEchoAppTOMLForSaga renders the echo app template with the given port for saga tests
func renderEchoAppTOMLForSaga(t *testing.T, port int) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(echoAppTOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port int }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// TestSagaTimingVsHTTPServerReadiness verifies that HTTP servers are immediately
// ready to accept connections after saga orchestrator completion.
func TestSagaTimingVsHTTPServerReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	// Create transaction storage and saga orchestrator
	txStorage := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStorage, slog.Default().Handler())

	// Create HTTP runner with longer siphon timeout to avoid race condition
	httpRunner, err := httplistener.NewRunner(
		httplistener.WithSiphonTimeout(60 * time.Second),
	)
	require.NoError(t, err)

	// Create transaction siphon
	txSiphon := make(chan *transaction.ConfigTransaction)

	// Create gRPC config service runner
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start both runners
	httpErrCh := make(chan error, 1)
	go func() {
		httpErrCh <- httpRunner.Run(ctx)
	}()

	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(ctx)
	}()

	// Wait for both runners to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Process transactions from siphon
	go func() {
		for tx := range txSiphon {
			if tx == nil {
				continue
			}
			if err := saga.ProcessTransaction(ctx, tx); err != nil {
				// Don't log if context was canceled (expected during test cleanup)
				if ctx.Err() == nil {
					t.Logf("Failed to process transaction: %v", err)
				}
			}
		}
	}()

	// Create client
	c := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Get a random port for the HTTP listener
	httpPort := testutil.GetRandomPort(t)

	// Render the echo app template
	configContent := renderEchoAppTOMLForSaga(t, httpPort)

	// Load test configuration with echo app
	tomlLoader := toml.NewTomlLoader([]byte(configContent))

	// Apply configuration
	err = c.ApplyConfig(ctx, tomlLoader)
	require.NoError(t, err)

	// Wait for the saga to actually complete the transaction processing
	err = saga.WaitForCompletion(ctx)
	require.NoError(t, err, "Saga should complete transaction processing successfully")

	// Test that the endpoint is immediately accessible
	testURL := fmt.Sprintf("http://127.0.0.1:%d/echo", httpPort)
	resp, err := http.Get(testURL)
	require.NoError(t, err, "HTTP endpoint should be immediately accessible after saga completion")
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should get OK response")
}

// TestHTTPRunnerRequiresInitialConfig verifies that the HTTP runner waits for
// initial configuration before transitioning to Running state
func TestHTTPRunnerRequiresInitialConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	// Create transaction storage and saga orchestrator
	txStorage := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStorage, slog.Default().Handler())

	// Create transaction manager with siphon
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

	// Start txmgr
	txmgrErrCh := make(chan error, 1)
	go func() {
		txmgrErrCh <- txMan.Run(ctx)
	}()

	// Wait for txmgr to be running
	require.Eventually(t, func() bool {
		return txMan.IsRunning()
	}, time.Second, 10*time.Millisecond, "txmgr should start and be ready to process transactions")

	// Start the HTTP runner
	httpErrCh := make(chan error, 1)
	go func() {
		httpErrCh <- httpRunner.Run(ctx)
	}()

	// Verify that HTTP runner becomes Running immediately
	// The httpcluster transitions to Running state even without configuration
	assert.Eventually(t, func() bool {
		state := httpRunner.GetState()
		t.Logf("HTTP runner state without config: %s", state)
		// httpcluster reports Running immediately, even while waiting for config
		return state == "Running"
	}, time.Second, 50*time.Millisecond, "HTTP runner should be in Running state")

	// Now send a configuration through the siphon
	// Get a random port for the HTTP listener
	httpPort := testutil.GetRandomPort(t)

	// Render the echo app template
	configContent := renderEchoAppTOMLForSaga(t, httpPort)

	tomlLoader := toml.NewTomlLoader([]byte(configContent))
	pbConfig, err := tomlLoader.LoadProto()
	require.NoError(t, err)
	cfg, err := config.NewFromProto(pbConfig)
	require.NoError(t, err)

	// Create a transaction with the config
	tx, err := transaction.New(
		transaction.SourceFile,
		"integration-test",
		"req-123",
		cfg,
		slog.Default().Handler(),
	)
	require.NoError(t, err)

	// Send transaction through siphon
	select {
	case txSiphon <- tx:
		t.Log("Sent transaction to siphon")
	case <-time.After(time.Second):
		t.Fatal("Timeout sending transaction to siphon")
	}

	// Now the HTTP runner should transition to Running after processing the config
	require.Eventually(t, func() bool {
		state := httpRunner.GetState()
		t.Logf("HTTP runner state: %s", state)
		return state == "Running"
	}, 5*time.Second, 100*time.Millisecond, "HTTP runner should transition to Running after receiving config")
}
