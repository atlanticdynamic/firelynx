//go:build integration
// +build integration

package http_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPListenerMinimalSaga tests the minimal saga flow with an empty configuration
func TestHTTPListenerMinimalSaga(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage
	txStore := txstorage.NewTransactionStorage()

	// Create saga orchestrator
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	assert.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Create a minimal configuration using string
	configData := `version = "v1"`

	// Create config from string
	testConfig, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err)

	// Create a config transaction
	tx, err := transaction.FromFile(t.Name(), testConfig, nil)
	require.NoError(t, err)

	// Validate the transaction
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process the transaction through the orchestrator
	err = saga.ProcessTransaction(ctx, tx)
	require.NoError(t, err)

	// Verify the transaction completed successfully
	assert.Equal(t, "completed", tx.GetState())

	// Wait a moment for the runner state to stabilize
	assert.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, 100*time.Millisecond, 10*time.Millisecond, "HTTP runner should be running after transaction completes")

	// Stop the HTTP runner
	httpRunner.Stop()

	// Wait for runner to stop
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Check that the runner didn't error
	select {
	case err := <-runnerErrCh:
		if err != nil && err != context.Canceled {
			t.Logf("Runner error: %v", err)
		}
	default:
		// Runner might still be shutting down, that's ok
	}
}

// TestHTTPListenerConfigUpdate tests updating configuration through saga
func TestHTTPListenerConfigUpdate(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage
	txStore := txstorage.NewTransactionStorage()

	// Create saga orchestrator
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	assert.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// First configuration - empty
	config1Data := `version = "v1"`
	config1, err := config.NewConfigFromBytes([]byte(config1Data))
	require.NoError(t, err)

	tx1, err := transaction.FromFile("config-1", config1, nil)
	require.NoError(t, err)
	err = tx1.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx1)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx1.GetState())

	// Second configuration - add a listener
	config2Data := `
version = "v1"

[[listeners]]
id = "http-1"
type = "http"
address = "127.0.0.1:0"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"
`
	config2, err := config.NewConfigFromBytes([]byte(config2Data))
	require.NoError(t, err)

	tx2, err := transaction.FromFile("config-2", config2, nil)
	require.NoError(t, err)
	err = tx2.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx2)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx2.GetState())

	// Verify runner is still running
	assert.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Stop the HTTP runner
	httpRunner.Stop()

	// Wait for runner to stop
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}
