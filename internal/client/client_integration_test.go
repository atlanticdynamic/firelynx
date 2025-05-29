//go:build integration
// +build integration

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
	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/test_config.toml
var baseConfigContent string

//go:embed testdata/updated_config.toml
var updatedConfigContent string

func TestApplyConfigFromPath_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get available ports
	httpPort := testutil.GetRandomPort(t)
	grpcPort := testutil.GetRandomPort(t)

	// Create config files with dynamic ports
	tempDir := t.TempDir()

	// Replace port in base config
	baseConfig := strings.ReplaceAll(baseConfigContent, ":8080", fmt.Sprintf(":%d", httpPort))
	configPath := filepath.Join(tempDir, "config.toml")
	err := os.WriteFile(configPath, []byte(baseConfig), 0o644)
	require.NoError(t, err)

	// Replace port in updated config
	updatedConfig := strings.ReplaceAll(updatedConfigContent, ":8080", fmt.Sprintf(":%d", httpPort))
	updatedConfigPath := filepath.Join(tempDir, "updated_config.toml")
	err = os.WriteFile(updatedConfigPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	logging.SetupLogger("debug")
	logger := slog.Default()

	errCh := make(chan error, 1)
	go func() {
		err := server.Run(serverCtx, logger, configPath, fmt.Sprintf(":%d", grpcPort))
		errCh <- err
	}()

	// Wait for server to be ready
	assert.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", httpPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Server should become ready")

	// Create client
	client := New(Config{
		ServerAddr: fmt.Sprintf("localhost:%d", grpcPort),
	})

	// Apply the updated config via gRPC
	err = client.ApplyConfigFromPath(ctx, updatedConfigPath)
	assert.NoError(t, err, "Should apply config successfully")

	// Test that the new endpoint becomes available
	assert.Eventually(t, func() bool {
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
