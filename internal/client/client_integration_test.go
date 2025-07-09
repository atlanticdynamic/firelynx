//go:build integration

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
	require.Eventually(t, func() bool {
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
	require.Eventually(t, func() bool {
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
	require.Eventually(t, func() bool {
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
		require.NotNil(t, transaction, "Transaction should not be nil")
		assert.NotEmpty(t, transaction.GetId(), "Transaction should have an ID")
		assert.NotNil(t, transaction.GetCreatedAt(), "Transaction should have creation time")
		require.NotNil(t, transaction.GetConfig(), "Transaction should have config")
		assert.NotEmpty(
			t,
			transaction.GetConfig().GetVersion(),
			"Transaction config should have version",
		)
	})

	// Test ListConfigTransactions (should have transactions after update)
	var transactionID string
	t.Run("ListConfigTransactions_AfterUpdate", func(t *testing.T) {
		transactions, nextPageToken, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		assert.NoError(t, err)
		require.NotEmpty(t, transactions, "Should have transactions after update")
		assert.Empty(t, nextPageToken, "Should have no next page token with small dataset")

		// We know there's at least one transaction after the update
		transactionID = transactions[0].GetId()
		assert.NotEmpty(t, transactionID, "Transaction should have an ID")
	})

	// Test GetConfigTransaction with specific ID
	t.Run("GetConfigTransaction_SpecificID", func(t *testing.T) {
		// transactionID is guaranteed to be set from previous test
		require.NotEmpty(t, transactionID, "Transaction ID should be available from previous test")

		transaction, err := client.GetConfigTransaction(ctx, transactionID)
		assert.NoError(t, err)
		assert.NotNil(t, transaction, "Should retrieve specific transaction")
		assert.Equal(t, transactionID, transaction.GetId(), "Should return correct transaction")
		require.NotNil(t, transaction.GetConfig(), "Transaction should have config")
		assert.NotEmpty(
			t,
			transaction.GetConfig().GetVersion(),
			"Transaction config should have version",
		)
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

		// We know we have at least one transaction from the config update
		require.GreaterOrEqual(
			t,
			initialCount,
			1,
			"Should have at least one transaction from config update",
		)

		// Clear all but the last one
		clearedCount, err := client.ClearConfigTransactions(ctx, 1)
		assert.NoError(t, err)

		// Verify the clear operation
		if initialCount > 1 {
			assert.Equal(
				t,
				int32(initialCount-1),
				clearedCount,
				"Should have cleared exactly initialCount-1 transactions",
			)
		} else {
			// If we only had 1 transaction, nothing should be cleared (keeping last 1)
			assert.Equal(t, int32(0), clearedCount, "Should not clear anything when keeping last 1 with only 1 transaction")
		}

		// Verify remaining transactions
		transactions, _, err = client.ListConfigTransactions(ctx, "", 100, "", "")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(transactions), "Should have exactly 1 transaction remaining")
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

func TestApplyConfigFromTransaction_Integration(t *testing.T) {
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
	require.Eventually(t, func() bool {
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

	// Get the initial transaction (from the base config load)
	var initialTransactionID string
	t.Run("CaptureInitialTransaction", func(t *testing.T) {
		transactions, _, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		require.NoError(t, err)
		require.NotEmpty(t, transactions, "Should have initial transaction from server startup")

		// Get the first transaction (should be the initial config)
		initialTransactionID = transactions[0].GetId()
		assert.NotEmpty(t, initialTransactionID, "Initial transaction should have ID")

		// Verify it has config
		tx, err := client.GetConfigTransaction(ctx, initialTransactionID)
		require.NoError(t, err)
		require.NotNil(t, tx.GetConfig(), "Initial transaction should have config")
	})

	// Apply updated configuration to create a second transaction
	t.Run("ApplyUpdatedConfig", func(t *testing.T) {
		err := client.ApplyConfigFromPath(ctx, updatedConfigPath)
		require.NoError(t, err, "Should apply updated config successfully")

		// Wait for the update to take effect
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 10*time.Second, 200*time.Millisecond, "Updated endpoint should become available")

		// Verify we now have 2 transactions
		transactions, _, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		require.NoError(t, err)
		assert.Len(t, transactions, 2, "Should have 2 transactions after update")
	})

	// Test successful rollback to initial transaction
	t.Run("SuccessfulRollback", func(t *testing.T) {
		require.NotEmpty(t, initialTransactionID, "Initial transaction ID should be available")

		// Verify /updated endpoint is currently available
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
		require.NoError(t, err)
		require.Equal(
			t,
			http.StatusOK,
			resp.StatusCode,
			"/updated endpoint should be available before rollback",
		)
		resp.Body.Close()

		// Perform rollback
		err = client.ApplyConfigFromTransaction(ctx, initialTransactionID)
		require.NoError(t, err, "Rollback should succeed")

		// Verify /updated endpoint is no longer available
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return true // Connection error means endpoint is gone
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusNotFound
		}, 10*time.Second, 200*time.Millisecond, "/updated endpoint should be removed after rollback")

		// Verify /test endpoint is still available
		resp, err = http.Get(fmt.Sprintf("http://localhost:%d/test", httpPort))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "/test endpoint should still be available")
		resp.Body.Close()

		// Verify we now have 3 transactions (initial, update, rollback)
		transactions, _, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		require.NoError(t, err)
		assert.Len(t, transactions, 3, "Should have 3 transactions after rollback")
	})

	// Test rollback error cases
	t.Run("RollbackErrorCases", func(t *testing.T) {
		// Test non-existent transaction ID
		err := client.ApplyConfigFromTransaction(ctx, "non-existent-transaction-id")
		assert.Error(t, err, "Should fail with non-existent transaction ID")
		assert.Contains(
			t,
			err.Error(),
			"transaction not found",
			"Error should indicate transaction not found",
		)

		// Test empty transaction ID
		err = client.ApplyConfigFromTransaction(ctx, "")
		assert.Error(t, err, "Should fail with empty transaction ID")

		// Test malformed transaction ID
		err = client.ApplyConfigFromTransaction(ctx, "invalid-format")
		assert.Error(t, err, "Should fail with malformed transaction ID")
	})

	// Test rolling back to the most recent transaction (should be a no-op but valid)
	t.Run("RollbackToCurrentTransaction", func(t *testing.T) {
		// Get current transaction
		currentTx, err := client.GetCurrentConfigTransaction(ctx)
		require.NoError(t, err)
		require.NotNil(t, currentTx, "Should have current transaction")

		currentID := currentTx.GetId()
		require.NotEmpty(t, currentID, "Current transaction should have ID")

		// Roll back to current transaction (should succeed)
		err = client.ApplyConfigFromTransaction(ctx, currentID)
		assert.NoError(t, err, "Rollback to current transaction should succeed")

		// Verify we have one more transaction (the rollback itself)
		transactions, _, err := client.ListConfigTransactions(ctx, "", 10, "", "")
		require.NoError(t, err)
		assert.Len(t, transactions, 4, "Should have 4 transactions after rollback to current")
	})

	// Test round-trip rollback: initial -> updated -> initial -> updated
	t.Run("RoundTripRollback", func(t *testing.T) {
		// Apply updated config again
		err := client.ApplyConfigFromPath(ctx, updatedConfigPath)
		require.NoError(t, err, "Should apply updated config again")

		// Wait for /updated endpoint to become available
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 10*time.Second, 200*time.Millisecond, "/updated endpoint should be available again")

		// Get all transactions - we expect exactly 5 at this point:
		// 1. Initial config, 2. Update config, 3. Rollback to initial, 4. No-op rollback to current, 5. Update config again
		transactions, _, err := client.ListConfigTransactions(ctx, "", 20, "", "")
		require.NoError(t, err)
		require.Len(t, transactions, 5, "Should have exactly 5 transactions at this point")

		// The most recent transaction (index 4) should be the updated config with /updated endpoint
		updatedTransaction := transactions[4]
		require.NotNil(
			t,
			updatedTransaction.GetConfig(),
			"Most recent transaction should have config",
		)
		updatedTransactionID := updatedTransaction.GetId()

		// Rollback to initial (no /updated endpoint)
		err = client.ApplyConfigFromTransaction(ctx, initialTransactionID)
		require.NoError(t, err, "Should rollback to initial config")

		// Verify /updated is gone
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return true
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusNotFound
		}, 10*time.Second, 200*time.Millisecond, "/updated should be gone after rollback")

		// Rollback to updated config again
		err = client.ApplyConfigFromTransaction(ctx, updatedTransactionID)
		require.NoError(t, err, "Should rollback to updated config")

		// Verify /updated is back
		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/updated", httpPort))
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 10*time.Second, 200*time.Millisecond, "/updated should be back after second rollback")
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
