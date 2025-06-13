//go:build e2e
// +build e2e

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

//go:embed testdata/test_config.toml
var testConfigContent string

//go:embed testdata/updated_config.toml
var updatedConfigContent string

func TestApplyConfig(t *testing.T) {
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

func TestGetConfig(t *testing.T) {
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
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Server should become ready")

	// Test GetConfig with output to file
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	outputPath := filepath.Join(tempDir, "output_config.toml")
	err = GetConfig(ctx, grpcAddr, outputPath)
	assert.NoError(t, err, "Should get config successfully")

	// Verify output file exists and has content
	assert.FileExists(t, outputPath, "Output file should exist")
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Version = 'v1'", "Output should contain config version")

	// Test GetConfig without output path (prints to stdout)
	err = GetConfig(ctx, grpcAddr, "")
	assert.NoError(t, err, "Should get config and print to stdout")

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

func TestTransactionOperations(t *testing.T) {
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

	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	// Test GetCurrentTransaction (should work without error)
	t.Run("GetCurrentTransaction", func(t *testing.T) {
		err := GetCurrentTransaction(ctx, grpcAddr, "text")
		assert.NoError(t, err, "Should get current transaction")

		// Test with JSON format
		err = GetCurrentTransaction(ctx, grpcAddr, "json")
		assert.NoError(t, err, "Should get current transaction in JSON format")

		// Test with TOML format
		err = GetCurrentTransaction(ctx, grpcAddr, "toml")
		assert.NoError(t, err, "Should get current transaction in TOML format")
	})

	// Test ListTransactions (should work without error)
	t.Run("ListTransactions", func(t *testing.T) {
		err := ListTransactions(ctx, grpcAddr, 10, "", "", "")
		assert.NoError(t, err, "Should list transactions")

		// Test with pagination
		err = ListTransactions(ctx, grpcAddr, 1, "", "", "")
		assert.NoError(t, err, "Should list transactions with page size limit")

		// Test with filters
		err = ListTransactions(ctx, grpcAddr, 10, "", "COMMITTED", "")
		assert.NoError(t, err, "Should list transactions with state filter")
	})

	// Apply a config update to create transactions
	err = ApplyConfig(ctx, updatedConfigPath, grpcAddr, 5*time.Second)
	require.NoError(t, err, "Should apply config successfully")

	// Wait for the update to complete
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Updated endpoint should become available")

	// Test GetCurrentTransaction (should exist after update)
	t.Run("GetCurrentTransaction_AfterUpdate", func(t *testing.T) {
		err := GetCurrentTransaction(ctx, grpcAddr, "text")
		assert.NoError(t, err, "Should get current transaction after update")
	})

	// Test ListTransactions (should have transactions after update)
	t.Run("ListTransactions_AfterUpdate", func(t *testing.T) {
		err := ListTransactions(ctx, grpcAddr, 10, "", "", "")
		assert.NoError(t, err, "Should list transactions after update")
	})

	// Test ClearTransactions
	t.Run("ClearTransactions", func(t *testing.T) {
		err := ClearTransactions(ctx, grpcAddr, 1)
		assert.NoError(t, err, "Should clear transactions")
	})

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

func TestApplyConfigTimeout(t *testing.T) {
	ctx := t.Context()

	// Test with invalid server address to trigger timeout
	err := ApplyConfig(ctx, "nonexistent.toml", "invalid:1234", 1*time.Second)
	assert.Error(t, err, "Should fail with invalid server")
}

func TestGetConfigInvalidServer(t *testing.T) {
	ctx := t.Context()

	// Test with invalid server address
	err := GetConfig(ctx, "invalid:1234", "")
	assert.Error(t, err, "Should fail with invalid server")
}

func TestTransactionInvalidServer(t *testing.T) {
	ctx := t.Context()

	// Test GetCurrentTransaction with invalid server
	err := GetCurrentTransaction(ctx, "invalid:1234", "text")
	assert.Error(t, err, "Should fail with invalid server")

	// Test ListTransactions with invalid server
	err = ListTransactions(ctx, "invalid:1234", 10, "", "", "")
	assert.Error(t, err, "Should fail with invalid server")

	// Test GetTransaction with invalid server
	err = GetTransaction(ctx, "invalid:1234", "test-id", "text")
	assert.Error(t, err, "Should fail with invalid server")

	// Test ClearTransactions with invalid server
	err = ClearTransactions(ctx, "invalid:1234", 1)
	assert.Error(t, err, "Should fail with invalid server")
}
