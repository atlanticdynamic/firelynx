package http

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// For testing only - a minimal config
func mockConfig() *config.Config {
	return &config.Config{
		Version: "v1",
	}
}

// createTestRegistry creates a mock registry with one app for testing
func createTestRegistry() *mocks.MockRegistry {
	registry := mocks.NewMockRegistry()
	mockApp := mocks.NewMockApp("test-app")
	mockApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	registry.On("GetApp", mock.Anything).Return(mockApp, true)

	return registry
}

// Helper to manually set the app registry field in the transaction using reflection
// This is only for testing purposes - we wouldn't do this in production code
func setAppRegistry(tx *transaction.ConfigTransaction, registry apps.Registry) {
	// Use reflection to set the private field
	txValue := reflect.ValueOf(tx).Elem()
	registryField := txValue.FieldByName("appRegistry")

	// Create a new reflect.Value from our registry
	registryValue := reflect.ValueOf(registry)

	// Check if the field is valid and can be set
	if registryField.IsValid() && registryField.CanSet() {
		registryField.Set(registryValue)
	}
}

// createMockTransaction creates a test transaction with a mock app registry
func createMockTransaction(t *testing.T) *transaction.ConfigTransaction {
	t.Helper()
	cfg := mockConfig()
	tx, err := transaction.FromTest(t.Name(), cfg, nil)
	require.NoError(t, err)

	// Create and set a mock registry using reflection
	registry := createTestRegistry()
	setAppRegistry(tx, registry)

	return tx
}

func TestNewRunner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		logger := slog.Default()

		runner, err := NewRunner(logger)
		require.NoError(t, err)
		assert.NotNil(t, runner)
		assert.Equal(t, "HTTPRunner", runner.String())
	})

	t.Run("with nil logger", func(t *testing.T) {
		runner, err := NewRunner(nil)
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})
}

func TestRunner_ApplyPendingConfig(t *testing.T) {
	t.Run("no pending changes", func(t *testing.T) {
		logger := slog.Default()

		runner, err := NewRunner(logger)
		require.NoError(t, err)

		// Apply pending config when there are no pending changes
		err = runner.ApplyPendingConfig(context.Background())
		assert.NoError(t, err)
	})
}

func TestRunner_GetState(t *testing.T) {
	logger := slog.Default()

	runner, err := NewRunner(logger)
	require.NoError(t, err)

	// Initial state should be "New" based on the implementation
	assert.Equal(t, "New", runner.GetState())
	assert.False(t, runner.IsRunning())
}

func TestRunner_ExecuteConfig(t *testing.T) {
	logger := slog.Default()
	runner, err := NewRunner(logger)
	require.NoError(t, err)

	// Test with empty config
	tx := createMockTransaction(t)

	err = runner.ExecuteConfig(context.Background(), tx)
	assert.NoError(t, err)
}

func TestRunner_CompensateConfig(t *testing.T) {
	logger := slog.Default()
	runner, err := NewRunner(logger)
	require.NoError(t, err)

	// Test compensation
	tx := createMockTransaction(t)

	// First set something pending
	err = runner.ExecuteConfig(context.Background(), tx)
	require.NoError(t, err)

	// Then compensate
	err = runner.CompensateConfig(context.Background(), tx)
	assert.NoError(t, err)
}

func TestRunner_RunAndStop(t *testing.T) {
	logger := slog.Default()
	runner, err := NewRunner(logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run in a goroutine
	go func() {
		err := runner.Run(ctx)
		assert.NoError(t, err)
	}()

	// Stop the runner
	runner.Stop()

	// Verify it's stopped using assert.Eventually
	assert.Eventually(t, func() bool {
		return !runner.IsRunning()
	}, 1*time.Second, 10*time.Millisecond)
}

func TestRunner_GetStateChan(t *testing.T) {
	logger := slog.Default()
	runner, err := NewRunner(logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get state channel
	stateChan := runner.GetStateChan(ctx)
	assert.NotNil(t, stateChan)
}
