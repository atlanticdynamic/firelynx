package cfgfileloader

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/valid_config.toml
var validConfigTOML []byte

//go:embed testdata/updated_config.toml
var updatedConfigTOML []byte

//go:embed testdata/invalid_config.toml
var invalidConfigTOML []byte

const (
	validConfigFilename = "valid_config.toml"
)

// testHarness provides a clean test setup for cfgfileloader
type testHarness struct {
	t        *testing.T
	runner   *Runner
	txSiphon chan *transaction.ConfigTransaction
	ctx      context.Context
	cancel   context.CancelFunc
}

// newTestHarness creates a test harness with a buffered siphon channel
func newTestHarness(t *testing.T, filePath string, opts ...Option) *testHarness {
	t.Helper()
	if filePath == "" {
		tmpDir := t.TempDir()
		filePath = filepath.Join(tmpDir, validConfigFilename)
		err := os.WriteFile(filePath, validConfigTOML, 0o644)
		require.NoError(t, err)
		t.Logf("Created temporary config file: %s", filePath)
	}

	// Use buffered channel for tests to avoid blocking
	txSiphon := make(chan *transaction.ConfigTransaction, 10)
	runner, err := NewRunner(filePath, txSiphon, opts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())

	return &testHarness{
		t:        t,
		runner:   runner,
		txSiphon: txSiphon,
		ctx:      ctx,
		cancel:   cancel,
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

func TestNewRunner(t *testing.T) {
	t.Parallel()
	t.Run("creates runner with default options", func(t *testing.T) {
		h := newTestHarness(t, "")
		assert.NotNil(t, h.runner)
		assert.Contains(t, h.runner.filePath, validConfigFilename)
		assert.NotNil(t, h.runner.logger)
		assert.NotNil(t, h.runner.fsm)
	})

	t.Run("applies custom options", func(t *testing.T) {
		l := slog.Default()
		h := newTestHarness(t, "/test/path",
			WithLogger(l),
		)
		assert.Equal(t, l, h.runner.logger)
	})

	t.Run("errors on nil siphon", func(t *testing.T) {
		_, err := NewRunner("/test/path", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction siphon cannot be nil")
	})
}

func TestRunner_String(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t, "/test/path")
	assert.Equal(t, "cfgfileloader.Runner", h.runner.String())
}

func TestRunner_Run(t *testing.T) {
	t.Parallel()
	t.Run("successful run with empty config path", func(t *testing.T) {
		h := newTestHarness(t, "")

		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(h.ctx)
		}()

		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)

		h.cancel()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
	})

	t.Run("run with valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(h.ctx)
		}()

		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusRunning && h.runner.getConfig() != nil
		}, time.Second, 10*time.Millisecond)

		// Should receive initial transaction
		tx := h.receiveTransaction()
		assert.NotNil(t, tx)

		cfg := h.runner.getConfig()
		assert.NotNil(t, cfg)

		h.cancel()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
		assert.Nil(t, h.runner.getConfig())
	})

	t.Run("run with invalid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid_config.toml")
		err := os.WriteFile(configPath, invalidConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)
		defer h.cancel()

		err = h.runner.Run(h.ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize configuration")
		assert.Equal(t, finitestate.StatusError, h.runner.GetState())
	})

	t.Run("run with non-existent config file", func(t *testing.T) {
		h := newTestHarness(t, "/non/existent/path.toml")
		defer h.cancel()

		err := h.runner.Run(h.ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize configuration")
	})
}

func TestRunner_Stop(t *testing.T) {
	t.Parallel()
	t.Run("stop transitions to stopping state", func(t *testing.T) {
		h := newTestHarness(t, "")
		defer h.cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(h.ctx)
		}()

		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)

		h.runner.Stop()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
	})
}

func TestRunner_Reload(t *testing.T) {
	t.Parallel()
	t.Run("reload with default config", func(t *testing.T) {
		h := newTestHarness(t, "")

		h.runner.Reload()
		// Now that we create a valid config file by default, it should load successfully
		assert.NotNil(t, h.runner.getConfig())
	})

	t.Run("reload with valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		assert.Nil(t, h.runner.getConfig())

		// Reload when not running - config should be stored but no transaction sent
		h.runner.Reload()
		cfg := h.runner.getConfig()
		assert.NotNil(t, cfg)

		// Update config file
		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)

		// Reload again - config should be updated but still no transaction
		h.runner.Reload()
		newCfg := h.runner.getConfig()
		assert.NotNil(t, newCfg)
		assert.NotSame(t, cfg, newCfg)
	})

	t.Run("reload with invalid config file logs error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid_config.toml")
		err := os.WriteFile(configPath, invalidConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		h.runner.Reload()
		assert.Nil(t, h.runner.getConfig())
	})

	t.Run("reload while running sends transactions", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)
		defer h.cancel()

		// Start the runner
		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(h.ctx)
		}()

		// Wait for runner to be running and receive initial transaction
		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)
		tx1 := h.receiveTransaction()
		assert.NotNil(t, tx1)

		// Update config and reload
		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)

		h.runner.Reload()
		tx2 := h.receiveTransaction()
		assert.NotNil(t, tx2)

		// Verify config was updated
		cfg := h.runner.getConfig()
		assert.NotNil(t, cfg)

		// Stop the runner
		h.cancel()
		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}
	})
}

func TestRunner_GetConfig(t *testing.T) {
	t.Parallel()
	t.Run("returns nil when no config loaded", func(t *testing.T) {
		h := newTestHarness(t, "")
		assert.Nil(t, h.runner.getConfig())
	})

	t.Run("returns loaded config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		h.runner.Reload()
		cfg := h.runner.getConfig()
		assert.NotNil(t, cfg)
		// No transaction expected when not running
	})
}

func TestRunner_StateInterfaces(t *testing.T) {
	t.Parallel()
	t.Run("implements Stateable interface", func(t *testing.T) {
		h := newTestHarness(t, "")

		state := h.runner.GetState()
		assert.Equal(t, finitestate.StatusNew, state)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		stateCh := h.runner.GetStateChan(ctx)
		assert.NotNil(t, stateCh)

		assert.False(t, h.runner.IsRunning())
	})

	t.Run("state changes during lifecycle", func(t *testing.T) {
		h := newTestHarness(t, "")

		// Use separate contexts for state channel and runner
		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()
		stateCh := h.runner.GetStateChan(stateCtx)

		runCtx, runCancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(runCtx)
		}()

		// Assert the expected state sequence

		// 1. Should start with New state
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusNew, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected New state")
		}

		// 2. Should transition to Booting
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusBooting, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Booting state")
		}

		// 3. Should transition to Running
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusRunning, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Running state")
		}

		// Verify runner is now running
		assert.True(t, h.runner.IsRunning())

		// Trigger shutdown
		runCancel()

		// 4. Should transition to Stopping
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusStopping, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Stopping state")
		}

		// 5. Should transition to Stopped
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusStopped, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Stopped state")
		}

		// Wait for Run() to complete
		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		// Final verification
		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
		assert.False(t, h.runner.IsRunning())
	})
}

func TestRunner_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown transitions states correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		// Set the FSM to Running state (via proper transition path)
		err = h.runner.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = h.runner.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Load a config to verify it gets cleared
		h.runner.Reload()
		assert.NotNil(t, h.runner.getConfig())

		// Call shutdown directly
		err = h.runner.shutdown()
		assert.NoError(t, err)

		// Verify state transitions and cleanup
		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
		assert.Nil(t, h.runner.getConfig())
	})

	t.Run("shutdown clears config even without prior loading", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		// Transition to Running without loading config
		err = h.runner.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)
		err = h.runner.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		// Verify no config is loaded
		assert.Nil(t, h.runner.getConfig())

		// Call shutdown - should work fine even without config
		err = h.runner.shutdown()
		assert.NoError(t, err)

		// Verify state transitions
		assert.Equal(t, finitestate.StatusStopped, h.runner.GetState())
		assert.Nil(t, h.runner.getConfig())
	})

	t.Run("shutdown handles invalid FSM transitions gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		h := newTestHarness(t, configPath)

		// Runner starts in New state
		assert.Equal(t, finitestate.StatusNew, h.runner.GetState())

		// Load a config to verify it gets cleared even when FSM transition fails
		h.runner.Reload()
		assert.NotNil(t, h.runner.getConfig())

		// Call shutdown from New state - this will fail FSM transitions but should still clear config
		err = h.runner.shutdown()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to transition to stopped state")

		// Config should still be cleared even though FSM transition failed
		assert.Nil(t, h.runner.getConfig())
		// FSM should remain in New state since transition failed
		assert.Equal(t, finitestate.StatusNew, h.runner.GetState())
	})
}

func TestRunner_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")
	err := os.WriteFile(configPath, validConfigTOML, 0o644)
	require.NoError(t, err)

	h := newTestHarness(t, configPath)
	defer h.cancel()

	// Start the runner first
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.runner.Run(h.ctx)
	}()

	// Wait for runner to be running
	assert.Eventually(t, func() bool {
		return h.runner.GetState() == finitestate.StatusRunning
	}, time.Second, 10*time.Millisecond)

	// Drain the initial transaction
	h.receiveTransaction()

	done := make(chan bool, 10)
	for range 10 {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				h.runner.Reload()
				cfg := h.runner.getConfig()
				if cfg != nil {
					_ = cfg.String()
				}
			}
		}()
	}

	for range 10 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}

	assert.NotNil(t, h.runner.getConfig())

	// Stop the runner
	h.cancel()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Runner did not complete within timeout")
	}
}
