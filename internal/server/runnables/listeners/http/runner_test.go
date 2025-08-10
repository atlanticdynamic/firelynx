package http

import (
	"context"
	"log/slog"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// For testing only - a minimal config
func mockConfig() *config.Config {
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	if err != nil {
		panic("failed to create mock config: " + err.Error())
	}
	cfg.Version = config.VersionLatest
	return cfg
}

// setupAppsInTransaction runs validation to create apps in the transaction
// This uses the normal validation flow instead of reflection
func setupAppsInTransaction(t *testing.T, tx *transaction.ConfigTransaction) {
	t.Helper()
	// Run validation which creates the app instances
	err := tx.RunValidation()
	require.NoError(t, err, "Failed to run validation to set up apps")
}

// createMockTransaction creates a test transaction with apps set up
func createMockTransaction(t *testing.T) *transaction.ConfigTransaction {
	t.Helper()
	cfg := mockConfig()
	tx, err := transaction.FromTest(t.Name(), cfg, nil)
	require.NoError(t, err)

	// Set up apps using normal validation flow
	setupAppsInTransaction(t, tx)

	return tx
}

func TestNewRunner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)
		assert.NotNil(t, runner)
		assert.Equal(t, "HTTPRunner", runner.String())
	})

	t.Run("with custom logger", func(t *testing.T) {
		customLogger := slog.Default().With("test", "custom")
		runner, err := NewRunner(WithLogger(customLogger))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
		// The logger should be set (can't easily verify internals but we know it's set)
	})

	t.Run("with custom handler", func(t *testing.T) {
		handler := slog.Default().Handler()
		runner, err := NewRunner(WithLogHandler(handler))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("with siphon timeout", func(t *testing.T) {
		runner, err := NewRunner(WithSiphonTimeout(5 * time.Second))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("with multiple options", func(t *testing.T) {
		logger := slog.Default().With("test", "logger")
		runner, err := NewRunner(
			WithLogger(logger),
		)
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})
}

func TestRunner_RunAndStop(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Run in a goroutine
	errChan := make(chan error)
	go func() {
		errChan <- runner.Run(ctx)
	}()

	// Wait for the runner to start
	assert.Eventually(t, func() bool {
		return runner.IsRunning()
	}, 1*time.Second, 10*time.Millisecond)

	// Stop the runner
	runner.Stop()

	// Wait for Run to return
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	// Verify it's stopped
	assert.Eventually(t, func() bool {
		return !runner.IsRunning()
	}, 1*time.Second, 10*time.Millisecond, "runner should stop")
}
