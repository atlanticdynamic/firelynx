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
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
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

func TestConfigTransactions_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
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

	// Test GetCurrentConfigTransaction (should work without error)
	t.Run("GetCurrentConfigTransaction_Basic", func(t *testing.T) {
		transaction, err := client.GetCurrentConfigTransaction(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, transaction, "Transaction should not be nil")
		assert.NotNil(t, transaction.GetCreatedAt(), "Transaction should have creation time")
	})

	// Test ListConfigTransactions (should work without error)
	t.Run("ListConfigTransactions_Basic", func(t *testing.T) {
		transactions, nextPageToken, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, transactions, "Should return transactions slice (may be empty)")
		assert.Empty(t, nextPageToken, "Should have no next page token with small dataset")
	})

	// Apply a config update to create transactions
	err = client.ApplyConfigFromPath(ctx, updatedConfigPath)
	require.NoError(t, err, "Should apply config successfully")

	// Wait for the update to complete
	assert.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "Updated endpoint should become available")

	// Test GetCurrentConfigTransaction (should exist after update)
	t.Run("GetCurrentConfigTransaction_AfterUpdate", func(t *testing.T) {
		transaction, err := client.GetCurrentConfigTransaction(ctx)
		assert.NoError(t, err)
		if transaction != nil {
			assert.NotEmpty(t, transaction.GetId(), "Transaction should have an ID")
			assert.NotNil(t, transaction.GetCreatedAt(), "Transaction should have creation time")
		}
	})

	// Test ListConfigTransactions (should have transactions after update)
	var transactionID string
	t.Run("ListConfigTransactions_AfterUpdate", func(t *testing.T) {
		transactions, nextPageToken, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, transactions, "Should have transactions after update")
		assert.Empty(t, nextPageToken, "Should have no next page token with small dataset")

		if len(transactions) > 0 {
			transactionID = transactions[0].GetId()
			assert.NotEmpty(t, transactionID, "Transaction should have an ID")
		}
	})

	// Test GetConfigTransaction with specific ID
	t.Run("GetConfigTransaction_SpecificID", func(t *testing.T) {
		if transactionID == "" {
			t.Skip("No transaction ID available for test")
		}

		transaction, err := client.GetConfigTransaction(ctx, transactionID)
		assert.NoError(t, err)
		assert.NotNil(t, transaction, "Should retrieve specific transaction")
		assert.Equal(t, transactionID, transaction.GetId(), "Should return correct transaction")
	})

	// Test ListConfigTransactions with filters
	t.Run("ListConfigTransactions_WithFilters", func(t *testing.T) {
		// Test with page size
		transactions, _, err := client.ListConfigTransactions(ctx, "", 1, "", "")
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(transactions), 1, "Should respect page size limit")

		// Test with state filter (if transactions have states)
		transactions, _, err = client.ListConfigTransactions(ctx, "", 10, "COMMITTED", "")
		assert.NoError(t, err)
		// Don't assert on count since we don't know the exact state values
	})

	// Test ClearConfigTransactions
	t.Run("ClearConfigTransactions", func(t *testing.T) {
		// First, get the current count
		transactions, _, err := client.ListConfigTransactions(ctx, "", 100, "", "")
		require.NoError(t, err)
		initialCount := len(transactions)

		if initialCount > 1 {
			// Clear all but the last one
			clearedCount, err := client.ClearConfigTransactions(ctx, 1)
			assert.NoError(t, err)
			assert.Greater(t, clearedCount, int32(0), "Should have cleared some transactions")

			// Verify fewer transactions remain
			transactions, _, err = client.ListConfigTransactions(ctx, "", 100, "", "")
			assert.NoError(t, err)
			assert.Less(
				t,
				len(transactions),
				initialCount,
				"Should have fewer transactions after clearing",
			)
		} else {
			// If we only have 1 or 0 transactions, test that clearing works without error
			clearedCount, err := client.ClearConfigTransactions(ctx, 0)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, clearedCount, int32(0), "Should return non-negative cleared count")
		}
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
