package cfgservice

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun tests all Run method functionality with different configurations
func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("basic_functionality", func(t *testing.T) {
		// Create a Runner instance with a listen address
		r, err := NewRunner(
			WithListenAddr(testutil.GetRandomListeningPort(t)),
		)
		require.NoError(t, err)

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run the Runner in a goroutine
		runErr := make(chan error)
		go func() {
			runErr <- r.Run(ctx)
		}()

		// Wait for the context to time out
		chanErr := <-runErr
		assert.NoError(t, chanErr)
	})

	t.Run("with_invalid_address", func(t *testing.T) {
		// Create a Runner with an invalid listen address that will cause NewGRPCManager to fail
		listenAddr := "invalid:address:with:too:many:colons"
		r, err := NewRunner(WithListenAddr(listenAddr))
		require.NoError(t, err)

		// Run should return the error from NewGRPCManager
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = r.Run(ctx)
		// Simply verify that there's an error, as the exact error may change
		assert.Error(
			t,
			err,
			"Run should return an error when NewGRPCManager fails with an invalid address",
		)
	})

	t.Run("with_config_path", func(t *testing.T) {
		// Create a temporary directory that's automatically cleaned up
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write default config to the file
		err := os.WriteFile(configPath, []byte("version = \"v1\"\n"), 0o644)
		require.NoError(t, err)

		// Create a Runner with the config path
		listenAddr := testutil.GetRandomListeningPort(t)
		r, err := NewRunner(
			WithListenAddr(listenAddr),
			WithConfigPath(configPath),
		)
		require.NoError(t, err)

		// Run for a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = r.Run(ctx)
		assert.NoError(t, err) // Should return nil on clean shutdown with supervisor pattern
	})

	t.Run("with_listen_addr_only", func(t *testing.T) {
		// Use a random port to avoid conflicts
		r, err := NewRunner(
			WithListenAddr(testutil.GetRandomListeningPort(t)),
		)
		require.NoError(t, err)

		// Run for a short time
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- r.Run(ctx)
		}()

		// Wait for the context to time out
		select {
		case err := <-errCh:
			assert.NoError(t, err) // Should return nil on clean shutdown with supervisor pattern
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for Runner to run")
		}
	})

	t.Run("with_config_path_only", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write a valid config
		err := os.WriteFile(configPath, []byte(`version = "v1"`), 0o644)
		require.NoError(t, err)

		// Create Runner with config path only
		r, err := NewRunner(WithConfigPath(configPath))
		require.NoError(t, err)

		// Run for a short time (should block on ctx.Done)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = r.Run(ctx)
		assert.NoError(t, err)
	})
}

// TestLoadInitialConfig tests the LoadInitialConfig method
func TestLoadInitialConfig(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write a valid config
		err := os.WriteFile(configPath, []byte(`version = "v1"`), 0o644)
		require.NoError(t, err)

		// Create Runner with config path
		r, err := NewRunner(WithConfigPath(configPath))
		require.NoError(t, err)

		// Set logger to discard output for cleaner test logs
		r.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

		// Directly call LoadInitialConfig
		err = r.LoadInitialConfig()
		assert.NoError(t, err)

		// Check that config was loaded
		cfg := r.GetPbConfigClone()
		assert.NotNil(t, cfg)
		assert.Equal(t, "v1", *cfg.Version)
	})

	t.Run("failure_with_listen_addr", func(t *testing.T) {
		// Create a Runner with non-existent config path
		nonExistentPath := "/tmp/non-existent-config-file.toml"

		// Ensure the file doesn't exist
		if _, err := os.Stat(nonExistentPath); err == nil {
			err = os.Remove(nonExistentPath)
			require.NoError(t, err, "Failed to remove test file")
		}

		r, err := NewRunner(
			WithConfigPath(nonExistentPath),
			WithListenAddr(testutil.GetRandomListeningPort(t)),
		)
		require.NoError(t, err)

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run should not return an error since we have a listen address
		err = r.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("failure_without_listen_addr", func(t *testing.T) {
		// Create a Runner with non-existent config path
		nonExistentPath := "/tmp/non-existent-config-file.toml"

		// Ensure the file doesn't exist
		if _, err := os.Stat(nonExistentPath); err == nil {
			err = os.Remove(nonExistentPath)
			require.NoError(t, err, "Failed to remove test file")
		}

		r, err := NewRunner(
			WithConfigPath(nonExistentPath),
		)
		require.NoError(t, err)

		// Create a context that will cancel after a short time
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run should return an error since we don't have a listen address
		err = r.Run(ctx)
		assert.Error(
			t,
			err,
			"Run should return an error when config loading fails and no listen address is provided",
		)
	})
}
