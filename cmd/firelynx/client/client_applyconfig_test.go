//go:build e2e

package client

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

	"github.com/atlanticdynamic/firelynx/cmd/firelynx/server"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyConfigWithTimeout(t *testing.T) {
	ctx := t.Context()

	// Test with zero timeout (should not set timeout)
	err := ApplyConfig(ctx, "nonexistent.toml", "invalid:1234", 0)
	assert.Error(t, err, "Should fail with invalid config file")

	// Test with positive timeout
	err = ApplyConfig(ctx, "nonexistent.toml", "invalid:1234", 5*time.Second)
	assert.Error(t, err, "Should fail with invalid config file")
}

func TestApplyConfigInvalidFile(t *testing.T) {
	ctx := t.Context()

	// Test with nonexistent file
	err := ApplyConfig(ctx, "nonexistent.toml", "localhost:50051", 0)
	assert.Error(t, err, "Should fail with nonexistent config file")

	// Test with invalid file path
	err = ApplyConfig(ctx, "/invalid/path/config.toml", "localhost:50051", 0)
	assert.Error(t, err, "Should fail with invalid file path")
}

func TestApplyConfigInvalidServer(t *testing.T) {
	ctx := t.Context()

	// Test with invalid server address to trigger timeout
	err := ApplyConfig(ctx, "nonexistent.toml", "invalid:1234", 1*time.Second)
	assert.Error(t, err, "Should fail with invalid server")
}

func TestApplyConfigE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	// Get available ports
	httpPort := testutil.GetRandomPort(t)
	grpcPort := testutil.GetRandomPort(t)

	// Create config files with dynamic ports
	tempDir := t.TempDir()

	// Replace port in test config
	testConfig := strings.ReplaceAll(testConfigContent, ":8080", fmt.Sprintf(":%d", httpPort))
	configPath := filepath.Join(tempDir, "config.toml")
	err := os.WriteFile(configPath, []byte(testConfig), 0o644)
	require.NoError(t, err)

	// Replace port in updated config
	updatedConfig := strings.ReplaceAll(updatedConfigContent, ":8080", fmt.Sprintf(":%d", httpPort))
	updatedConfigPath := filepath.Join(tempDir, "updated_config.toml")
	err = os.WriteFile(updatedConfigPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	errCh := make(chan error, 1)
	go func() {
		err := server.Run(serverCtx, logger, configPath, fmt.Sprintf(":%d", grpcPort))
		errCh <- err
	}()

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", httpPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Server should become ready")

	// Test ApplyConfig
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	err = ApplyConfig(ctx, updatedConfigPath, grpcAddr, 5*time.Second)
	assert.NoError(t, err, "Should apply config successfully")

	// Test that the new endpoint becomes available
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Updated endpoint should become available")

	// Shutdown server
	serverCancel()

	// Wait for clean shutdown
	select {
	case err := <-errCh:
		assert.NoError(t, err, "Server should shut down cleanly")
	case <-time.After(30 * time.Second):
		t.Error("Server did not shut down within timeout")
	}
}
