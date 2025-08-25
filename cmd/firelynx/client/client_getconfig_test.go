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

func TestGetCurrentConfigEdgeCases(t *testing.T) {
	ctx := t.Context()

	// Test GetCurrentConfig with toml format and output path
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.toml")
	err := GetCurrentConfig(ctx, "invalid:1234", "toml", outputPath)
	require.Error(t, err, "Should fail with invalid server when using toml format with output path")

	// Test GetCurrentConfig with toml format but no output path
	err = GetCurrentConfig(ctx, "invalid:1234", "toml", "")
	require.Error(
		t,
		err,
		"Should fail with invalid server when using toml format without output path",
	)

	// Test GetCurrentConfig with json format
	err = GetCurrentConfig(ctx, "invalid:1234", "json", "")
	require.Error(t, err, "Should fail with invalid server when using json format")

	// Test GetCurrentConfig with text format
	err = GetCurrentConfig(ctx, "invalid:1234", "text", outputPath)
	require.Error(t, err, "Should fail with invalid server when using text format")
}

func TestGetConfigEdgeCases(t *testing.T) {
	ctx := t.Context()

	// Test GetConfig with invalid server and empty output path
	err := GetConfig(ctx, "invalid:1234", "")
	require.Error(t, err, "Should fail with invalid server when printing to stdout")

	// Test GetConfig with invalid server and output path
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.toml")
	err = GetConfig(ctx, "invalid:1234", outputPath)
	require.Error(t, err, "Should fail with invalid server when saving to file")
}

func TestGetConfigInvalidServer(t *testing.T) {
	ctx := t.Context()

	// Test with invalid server address
	err := GetConfig(ctx, "invalid:1234", "")
	require.Error(t, err, "Should fail with invalid server")
}

func TestGetConfigE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	// Get available ports
	httpPort := testutil.GetRandomPort(t)
	grpcPort := testutil.GetRandomPort(t)

	// Create config file with dynamic port
	tempDir := t.TempDir()
	testConfig := strings.ReplaceAll(testConfigContent, ":8080", fmt.Sprintf(":%d", httpPort))
	configPath := filepath.Join(tempDir, "config.toml")
	err := os.WriteFile(configPath, []byte(testConfig), 0o644)
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
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Server should become ready")

	// Test GetConfig with output to file
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	outputPath := filepath.Join(tempDir, "output_config.toml")
	err = GetConfig(ctx, grpcAddr, outputPath)
	require.NoError(t, err, "Should get config successfully")

	// Verify output file exists and has content
	assert.FileExists(t, outputPath, "Output file should exist")
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Version = 'v1'", "Output should contain config version")

	// Test GetConfig without output path (prints to stdout)
	err = GetConfig(ctx, grpcAddr, "")
	require.NoError(t, err, "Should get config and print to stdout")

	// Shutdown server
	serverCancel()

	// Wait for clean shutdown
	select {
	case err := <-errCh:
		require.NoError(t, err, "Server should shut down cleanly")
	case <-time.After(30 * time.Second):
		t.Error("Server did not shut down within timeout")
	}
}
