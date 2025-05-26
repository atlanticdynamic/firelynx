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

// Define a custom type for context keys to avoid collisions
type contextKey string

const testContextKey contextKey = "test"

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

// Helper to manually set the app collection field in the transaction using reflection
// This is only for testing purposes - we wouldn't do this in production code
func setAppCollection(tx *transaction.ConfigTransaction, collection apps.AppLookup) {
	// Use reflection to set the private field
	txValue := reflect.ValueOf(tx).Elem()
	collectionField := txValue.FieldByName("appCollection")

	// Create a new reflect.Value from our collection
	collectionValue := reflect.ValueOf(collection)

	// Check if the field is valid and can be set
	if collectionField.IsValid() && collectionField.CanSet() {
		collectionField.Set(collectionValue)
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
	setAppCollection(tx, registry)

	return tx
}

func TestNewRunner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)
		assert.NotNil(t, runner)
		assert.Equal(t, "HTTPRunner", runner.String())
	})

	t.Run("with custom logger", func(t *testing.T) {
		customLogger := slog.Default().With("test", "custom")
		runner, err := NewRunner(WithLogger(customLogger))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
		// The logger should be set (can't easily verify internals but we know it's set)
	})

	t.Run("with custom context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), testContextKey, "value")
		runner, err := NewRunner(WithContext(ctx))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("with custom handler", func(t *testing.T) {
		handler := slog.Default().Handler()
		runner, err := NewRunner(WithLogHandler(handler))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("with siphon timeout", func(t *testing.T) {
		runner, err := NewRunner(WithSiphonTimeout(5 * time.Second))
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("with multiple options", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), testContextKey, "value")
		logger := slog.Default().With("test", "logger")
		runner, err := NewRunner(
			WithContext(ctx),
			WithLogger(logger),
		)
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})
}

func TestRunner_ApplyPendingConfig(t *testing.T) {
	t.Run("no pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		// Apply pending config when there are no pending changes
		err = runner.ApplyPendingConfig(context.Background())
		assert.NoError(t, err)
	})
}

func TestRunner_GetState(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	// Initial state should be "New" based on the implementation
	assert.Equal(t, "New", runner.GetState())
	assert.False(t, runner.IsRunning())
}

func TestRunner_ExecuteConfig(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	// Test with empty config
	tx := createMockTransaction(t)

	err = runner.ExecuteConfig(context.Background(), tx)
	assert.NoError(t, err)
}

func TestRunner_CompensateConfig(t *testing.T) {
	runner, err := NewRunner()
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
	runner, err := NewRunner()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Run in a goroutine
	errChan := make(chan error)
	go func() {
		errChan <- runner.Run(ctx)
	}()

	// Wait for the runner to start
	assert.Eventually(t, func() bool {
		return runner.IsRunning()
	}, 1*time.Second, 10*time.Millisecond)

	// Stop the runner
	runner.Stop()

	// Wait for Run to return
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	// Verify it's stopped
	assert.False(t, runner.IsRunning())
}

func TestRunner_GetStateChan(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Get state channel
	stateChan := runner.GetStateChan(ctx)
	assert.NotNil(t, stateChan)
}
