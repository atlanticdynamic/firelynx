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

func TestNewRunner(t *testing.T) {
	t.Parallel()
	t.Run("creates runner with default options", func(t *testing.T) {
		runner, err := NewRunner("/test/path")
		require.NoError(t, err)
		assert.NotNil(t, runner)
		assert.Equal(t, "/test/path", runner.filePath)
		assert.NotNil(t, runner.logger)
		assert.NotNil(t, runner.fsm)
		assert.Equal(t, context.Background(), runner.parentCtx)
	})

	t.Run("applies custom options", func(t *testing.T) {
		type testKey string
		customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		customCtx := context.WithValue(context.Background(), testKey("test"), "value")

		runner, err := NewRunner("/test/path",
			WithLogger(customLogger),
			WithContext(customCtx),
		)
		require.NoError(t, err)
		assert.Equal(t, customLogger, runner.logger)
		assert.Equal(t, customCtx, runner.parentCtx)
	})
}

func TestRunner_String(t *testing.T) {
	t.Parallel()
	runner, err := NewRunner("/test/path")
	require.NoError(t, err)
	assert.Equal(t, "cfgfileloader.Runner", runner.String())
}

func TestRunner_Run(t *testing.T) {
	t.Parallel()
	t.Run("successful run with empty config path", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		assert.Eventually(t, func() bool {
			return runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)

		cancel()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, runner.GetState())
	})

	t.Run("run with valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		assert.Eventually(t, func() bool {
			return runner.GetState() == finitestate.StatusRunning && runner.getConfig() != nil
		}, time.Second, 10*time.Millisecond)

		cfg := runner.getConfig()
		assert.NotNil(t, cfg)

		cancel()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, runner.GetState())
		assert.Nil(t, runner.getConfig())
	})

	t.Run("run with invalid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid_config.toml")
		err := os.WriteFile(configPath, invalidConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = runner.Run(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize configuration")
		assert.Equal(t, finitestate.StatusError, runner.GetState())
	})

	t.Run("run with non-existent config file", func(t *testing.T) {
		runner, err := NewRunner("/non/existent/path.toml")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = runner.Run(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize configuration")
	})
}

func TestRunner_Stop(t *testing.T) {
	t.Parallel()
	t.Run("stop transitions to stopping state", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		assert.Eventually(t, func() bool {
			return runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)

		runner.Stop()

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Equal(t, finitestate.StatusStopped, runner.GetState())
	})
}

func TestRunner_Reload(t *testing.T) {
	t.Parallel()
	t.Run("reload with empty config path", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)

		runner.Reload()
		assert.Nil(t, runner.getConfig())
	})

	t.Run("reload with valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath)
		require.NoError(t, err)

		assert.Nil(t, runner.getConfig())

		runner.Reload()
		cfg := runner.getConfig()
		assert.NotNil(t, cfg)

		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)

		runner.Reload()
		newCfg := runner.getConfig()
		assert.NotNil(t, newCfg)

		assert.NotSame(t, cfg, newCfg)
	})

	t.Run("reload with invalid config file logs error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid_config.toml")
		err := os.WriteFile(configPath, invalidConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath)
		require.NoError(t, err)

		runner.Reload()
		assert.Nil(t, runner.getConfig())
	})
}

func TestRunner_GetConfig(t *testing.T) {
	t.Parallel()
	t.Run("returns nil when no config loaded", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)
		assert.Nil(t, runner.getConfig())
	})

	t.Run("returns loaded config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath)
		require.NoError(t, err)

		runner.Reload()
		cfg := runner.getConfig()
		assert.NotNil(t, cfg)
	})
}

func TestRunner_StateInterfaces(t *testing.T) {
	t.Parallel()
	t.Run("implements Stateable interface", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)

		state := runner.GetState()
		assert.Equal(t, finitestate.StatusNew, state)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stateCh := runner.GetStateChan(ctx)
		assert.NotNil(t, stateCh)

		assert.False(t, runner.IsRunning())
	})

	t.Run("state changes during lifecycle", func(t *testing.T) {
		runner, err := NewRunner("")
		require.NoError(t, err)

		// Use separate contexts for state channel and runner
		stateCtx, stateCancel := context.WithCancel(context.Background())
		defer stateCancel()
		stateCh := runner.GetStateChan(stateCtx)

		runCtx, runCancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(runCtx)
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
		assert.True(t, runner.IsRunning())

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
		assert.Equal(t, finitestate.StatusStopped, runner.GetState())
		assert.False(t, runner.IsRunning())
	})
}

func TestRunner_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")
	err := os.WriteFile(configPath, validConfigTOML, 0o644)
	require.NoError(t, err)

	runner, err := NewRunner(configPath)
	require.NoError(t, err)

	done := make(chan bool, 10)
	for range 10 {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				runner.Reload()
				cfg := runner.getConfig()
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

	assert.NotNil(t, runner.getConfig())
}

func TestRunner_GetConfigChan(t *testing.T) {
	t.Parallel()

	t.Run("sends initial config and updates on reload", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")

		// Write initial config
		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath, WithContext(t.Context()))
		require.NoError(t, err)

		// Get config channel before starting
		configCh := runner.GetConfigChan()

		// Start runner
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(t.Context())
		}()

		// Should receive initial config transaction after boot
		var initialTx *transaction.ConfigTransaction
		select {
		case tx := <-configCh:
			initialTx = tx
			assert.NotNil(t, tx)
			cfg := tx.GetConfig()
			assert.NotNil(t, cfg)
			assert.Equal(t, "v1", cfg.Version)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Did not receive initial config transaction")
		}

		// Write updated config
		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)

		// Trigger reload
		runner.Reload()

		// Should receive updated config transaction
		select {
		case tx := <-configCh:
			assert.NotNil(t, tx)
			cfg := tx.GetConfig()
			assert.NotNil(t, cfg)
			initialCfg := initialTx.GetConfig()
			assert.False(t, initialCfg.Equals(cfg), "Config should have changed")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Did not receive updated config transaction")
		}

		// Write same config again (no change)
		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)

		// Trigger reload again
		runner.Reload()

		// Should NOT receive transaction (unchanged)
		select {
		case <-configCh:
			t.Fatal("Should not receive transaction when unchanged")
		case <-time.After(100 * time.Millisecond):
			// Expected - no transaction sent
		}

		// Runner will stop when test context ends
	})

	t.Run("multiple subscribers receive updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")

		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		runner, err := NewRunner(configPath, WithContext(t.Context()))
		require.NoError(t, err)

		// Create multiple subscribers
		configCh1 := runner.GetConfigChan()
		configCh2 := runner.GetConfigChan()
		configCh3 := runner.GetConfigChan()

		// Start runner
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(t.Context())
		}()

		// All subscribers should receive initial config transaction
		transactions := make([]*transaction.ConfigTransaction, 3)
		for i, ch := range []<-chan *transaction.ConfigTransaction{configCh1, configCh2, configCh3} {
			select {
			case tx := <-ch:
				transactions[i] = tx
				assert.NotNil(t, tx)
				cfg := tx.GetConfig()
				assert.NotNil(t, cfg)
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("Subscriber %d did not receive initial config transaction", i+1)
			}
		}

		// Update config
		err = os.WriteFile(configPath, updatedConfigTOML, 0o644)
		require.NoError(t, err)
		runner.Reload()

		// All subscribers should receive updated config transaction
		for i, ch := range []<-chan *transaction.ConfigTransaction{configCh1, configCh2, configCh3} {
			select {
			case tx := <-ch:
				assert.NotNil(t, tx)
				cfg := tx.GetConfig()
				assert.NotNil(t, cfg)
				oldCfg := transactions[i].GetConfig()
				assert.False(
					t,
					oldCfg.Equals(cfg),
					"Config should have changed for subscriber %d",
					i+1,
				)
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("Subscriber %d did not receive updated config transaction", i+1)
			}
		}
	})

	t.Run("channel cleanup on context cancellation", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.toml")

		err := os.WriteFile(configPath, validConfigTOML, 0o644)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		runner, err := NewRunner(configPath, WithContext(ctx))
		require.NoError(t, err)

		// Get config channel
		configCh := runner.GetConfigChan()

		// Start runner
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		// Receive initial config transaction
		select {
		case tx := <-configCh:
			assert.NotNil(t, tx)
			cfg := tx.GetConfig()
			assert.NotNil(t, cfg)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Did not receive initial config transaction")
		}

		// Cancel context (shutdown runner)
		cancel()

		// Channel should be closed
		select {
		case tx, ok := <-configCh:
			assert.False(t, ok, "Channel should be closed, got transaction: %v", tx)
		case <-time.After(time.Second):
			t.Fatal("Channel was not closed within timeout")
		}

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}
	})
}
