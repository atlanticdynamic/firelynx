//go:build integration

package configupdates

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateConfigShutdownTimeout verifies that ValidateConfig does not create transactions
// that get stuck in non-terminal states, which would cause shutdown timeouts.
// This test was created to reproduce and verify the fix for the bug where ValidateConfig
// created transactions in StateValidated (non-terminal), causing WaitForCompletion to timeout.
func TestValidateConfigShutdownTimeout(t *testing.T) {
	ctx := t.Context()

	// Create transaction siphon
	txSiphon := make(chan *transaction.ConfigTransaction, 10)

	// Create gRPC config service runner
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Start config service
	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(ctx)
	}()

	// Wait for config service to start
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should start")

	// Create client
	testClient := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Wait for gRPC server to be ready
	assert.Eventually(t, func() bool {
		_, err := testClient.GetConfig(ctx)
		return err == nil || strings.Contains(err.Error(), "no configuration") ||
			strings.Contains(err.Error(), "not found")
	}, 5*time.Second, 200*time.Millisecond, "gRPC server should be ready")

	// Call ValidateConfig with a valid config
	// With the fix: This should NOT create any transactions that remain in memory
	// Before the fix: This would create transactions stuck in StateValidated (non-terminal)
	httpPort := testutil.GetRandomPort(t)
	configPath := createTempEchoAppConfigFile(t, httpPort)

	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	require.NoError(t, err)

	cfg, err := configLoader.LoadProto()
	require.NoError(t, err)

	// Validate the config - this should validate directly without creating lingering transactions
	isValid, validationErr := testClient.ValidateConfig(ctx, cfg)
	require.True(t, isValid, "Config should be valid")
	require.NoError(t, validationErr, "Validation should succeed")

	// Verify no transaction was sent to the siphon (this is expected behavior)
	select {
	case tx := <-txSiphon:
		t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
	case <-time.After(100 * time.Millisecond):
		// This is expected - ValidateConfig should not send to siphon
	}

	// Now attempt graceful shutdown - this should complete quickly
	// but will timeout if transactions are stuck in non-terminal states
	shutdownStartTime := time.Now()

	// Stop the config service and measure shutdown time
	cfgServiceRunner.Stop()

	// Wait for shutdown to complete with a reasonable timeout
	shutdownComplete := make(chan bool, 1)
	go func() {
		assert.Eventually(t, func() bool {
			return !cfgServiceRunner.IsRunning()
		}, 3*time.Second, 10*time.Millisecond, "gRPC config service should stop")
		shutdownComplete <- true
	}()

	select {
	case <-shutdownComplete:
		shutdownDuration := time.Since(shutdownStartTime)
		t.Logf("Shutdown completed in %v", shutdownDuration)

		// With the fix: Shutdown should complete quickly since no transactions are created
		// Before the fix: Shutdown would timeout due to transactions stuck in StateValidated
		if shutdownDuration > 2*time.Second {
			t.Errorf(
				"Shutdown took too long (%v), may indicate transactions stuck in non-terminal states",
				shutdownDuration,
			)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown timeout - likely caused by transactions stuck in non-terminal states")
	}

	// Check for any error in the config service
	select {
	case err := <-cfgServiceErrCh:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Config service returned unexpected error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		// No error is fine too
	}
}

// TestValidateConfigMultipleCallsShutdown tests that multiple ValidateConfig calls
// don't accumulate stuck transactions that would worsen shutdown timeouts
func TestValidateConfigMultipleCallsShutdown(t *testing.T) {
	ctx := t.Context()

	// Create transaction siphon
	txSiphon := make(chan *transaction.ConfigTransaction, 10)

	// Create gRPC config service runner
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Start config service
	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(ctx)
	}()

	// Wait for config service to start
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should start")

	// Create client
	testClient := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Wait for gRPC server to be ready
	assert.Eventually(t, func() bool {
		_, err := testClient.GetConfig(ctx)
		return err == nil || strings.Contains(err.Error(), "no configuration") ||
			strings.Contains(err.Error(), "not found")
	}, 5*time.Second, 200*time.Millisecond, "gRPC server should be ready")

	// Load test config
	httpPort := testutil.GetRandomPort(t)
	configPath := createTempEchoAppConfigFile(t, httpPort)

	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	require.NoError(t, err)

	cfg, err := configLoader.LoadProto()
	require.NoError(t, err)

	// Call ValidateConfig multiple times to simulate real usage
	for i := 0; i < 5; i++ {
		isValid, validationErr := testClient.ValidateConfig(ctx, cfg)
		require.True(t, isValid, "Config should be valid on call %d", i+1)
		require.NoError(t, validationErr, "Validation should succeed on call %d", i+1)
	}

	// Verify no transactions were sent to the siphon
	select {
	case tx := <-txSiphon:
		t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
	case <-time.After(100 * time.Millisecond):
		// This is expected
	}

	// Shutdown should still be fast even after multiple validation calls
	shutdownStartTime := time.Now()

	cfgServiceRunner.Stop()

	shutdownComplete := make(chan bool, 1)
	go func() {
		assert.Eventually(t, func() bool {
			return !cfgServiceRunner.IsRunning()
		}, 3*time.Second, 10*time.Millisecond, "gRPC config service should stop")
		shutdownComplete <- true
	}()

	select {
	case <-shutdownComplete:
		shutdownDuration := time.Since(shutdownStartTime)
		t.Logf("Shutdown after %d validations completed in %v", 5, shutdownDuration)

		// Multiple validations shouldn't make shutdown slower if we fix the terminal state issue
		if shutdownDuration > 2*time.Second {
			t.Errorf(
				"Shutdown took too long (%v) after multiple validations, may indicate accumulating stuck transactions",
				shutdownDuration,
			)
		}
	case <-time.After(5 * time.Second):
		t.Fatal(
			"Shutdown timeout after multiple validations - likely caused by accumulating transactions in non-terminal states",
		)
	}
}
