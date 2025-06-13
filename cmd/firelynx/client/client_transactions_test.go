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
	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentTransactionFormats(t *testing.T) {
	ctx := t.Context()

	// Test all format variations with invalid server to verify format handling
	formats := []string{"text", "json", "toml", "invalid"}

	for _, format := range formats {
		err := GetCurrentTransaction(ctx, "invalid:1234", format)
		assert.Error(t, err, "Should fail with invalid server for format: %s", format)
	}
}

func TestListTransactionsFormats(t *testing.T) {
	ctx := t.Context()

	// Test all format variations with invalid server to verify format handling
	formats := []string{"text", "json", "toml", "invalid"}

	for _, format := range formats {
		err := ListTransactions(ctx, "invalid:1234", 10, "", "", "", format)
		assert.Error(t, err, "Should fail with invalid server for format: %s", format)
	}
}

func TestListTransactionsPagination(t *testing.T) {
	ctx := t.Context()

	// Test with different page sizes
	pageSizes := []int32{1, 10, 100}

	for _, pageSize := range pageSizes {
		err := ListTransactions(ctx, "invalid:1234", pageSize, "", "", "", "text")
		assert.Error(t, err, "Should fail with invalid server for page size: %d", pageSize)
	}

	// Test with page token
	err := ListTransactions(ctx, "invalid:1234", 10, "next-page-token", "", "", "text")
	assert.Error(t, err, "Should fail with invalid server when using page token")

	// Test with state filter
	err = ListTransactions(ctx, "invalid:1234", 10, "", "COMMITTED", "", "text")
	assert.Error(t, err, "Should fail with invalid server when using state filter")

	// Test with source filter
	err = ListTransactions(ctx, "invalid:1234", 10, "", "", "file", "text")
	assert.Error(t, err, "Should fail with invalid server when using source filter")
}

func TestGetTransactionFormats(t *testing.T) {
	ctx := t.Context()

	// Test all format variations with invalid server to verify format handling
	formats := []string{"text", "json", "toml", "invalid"}

	for _, format := range formats {
		err := GetTransaction(ctx, "invalid:1234", "test-transaction-id", format)
		assert.Error(t, err, "Should fail with invalid server for format: %s", format)
	}
}

func TestGetTransactionWithEmptyID(t *testing.T) {
	ctx := t.Context()

	// Test with empty transaction ID
	err := GetTransaction(ctx, "invalid:1234", "", "text")
	assert.Error(t, err, "Should fail with invalid server even with empty transaction ID")
}

func TestClearTransactionsKeepLast(t *testing.T) {
	ctx := t.Context()

	// Test with different keepLast values
	keepLastValues := []int32{0, 1, 5, 10}

	for _, keepLast := range keepLastValues {
		err := ClearTransactions(ctx, "invalid:1234", keepLast)
		assert.Error(t, err, "Should fail with invalid server for keepLast: %d", keepLast)
	}
}

func TestTransactionInvalidServer(t *testing.T) {
	ctx := t.Context()

	// Test GetCurrentTransaction with invalid server
	err := GetCurrentTransaction(ctx, "invalid:1234", "text")
	assert.Error(t, err, "Should fail with invalid server")

	// Test ListTransactions with invalid server
	err = ListTransactions(ctx, "invalid:1234", 10, "", "", "", "text")
	assert.Error(t, err, "Should fail with invalid server")

	// Test GetTransaction with invalid server
	err = GetTransaction(ctx, "invalid:1234", "test-id", "text")
	assert.Error(t, err, "Should fail with invalid server")

	// Test ClearTransactions with invalid server
	err = ClearTransactions(ctx, "invalid:1234", 1)
	assert.Error(t, err, "Should fail with invalid server")

	// Test RollbackToTransaction with invalid server
	err = RollbackToTransaction(ctx, "invalid:1234", "test-id")
	assert.Error(t, err, "Should fail with invalid server")
}

func TestTransactionOperationsE2E(t *testing.T) {
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
		err := ListTransactions(ctx, grpcAddr, 10, "", "", "", "text")
		assert.NoError(t, err, "Should list transactions")

		// Test with pagination
		err = ListTransactions(ctx, grpcAddr, 1, "", "", "", "text")
		assert.NoError(t, err, "Should list transactions with page size limit")

		// Test with filters
		err = ListTransactions(ctx, grpcAddr, 10, "", "COMMITTED", "", "text")
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
		err := ListTransactions(ctx, grpcAddr, 10, "", "", "", "text")
		assert.NoError(t, err, "Should list transactions after update")
	})

	// Test RollbackToTransaction
	t.Run("RollbackToTransaction", func(t *testing.T) {
		err := RollbackToTransaction(ctx, grpcAddr, "non-existent-id")
		assert.Error(t, err, "Should fail with non-existent transaction ID")

		firelynxClient := client.New(client.Config{
			ServerAddr: grpcAddr,
			Logger: slog.New(
				slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
			),
		})

		transactions, _, err := firelynxClient.ListConfigTransactions(ctx, "", 10, "", "")
		require.NoError(t, err, "Should be able to list transactions")
		require.Len(t, transactions, 2, "Should have exactly 2 transactions at this point")

		// First transaction should be the initial config (oldest first in list)
		firstTransaction := transactions[0]
		require.NotNil(t, firstTransaction.GetConfig())

		// Verify it's the config without /updated endpoint
		hasUpdatedEndpoint := false
		for _, endpoint := range firstTransaction.GetConfig().GetEndpoints() {
			for _, route := range endpoint.GetRoutes() {
				if httpRule := route.GetHttp(); httpRule != nil &&
					httpRule.GetPathPrefix() == "/updated" {
					hasUpdatedEndpoint = true
					break
				}
			}
		}
		require.False(t, hasUpdatedEndpoint, "First transaction should not have /updated endpoint")

		// Verify /updated endpoint exists before rollback
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Perform rollback
		err = RollbackToTransaction(ctx, grpcAddr, firstTransaction.GetId())
		require.NoError(t, err, "Should successfully rollback to first transaction")

		// Verify /updated endpoint is gone after rollback
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return true
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusNotFound
		}, 10*time.Second, 200*time.Millisecond)
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
