package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/robbyt/go-supervisor/runnables/httpcluster"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_SagaOperations(t *testing.T) {
	t.Run("StageConfig", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)
		err = runner.StageConfig(context.Background(), tx)
		assert.NoError(t, err)
	})

	t.Run("StageConfig with nil transaction", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.StageConfig(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is nil")
	})

	t.Run("CommitConfig with no pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.CommitConfig(context.Background())
		assert.NoError(t, err)
	})

	t.Run("CommitConfig with pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)
		err = runner.StageConfig(context.Background(), tx)
		require.NoError(t, err)

		hasChanges := runner.configMgr.HasPendingChanges()
		assert.True(t, hasChanges, "Should have pending changes after StageConfig")

		runner.configMgr.CommitPending()

		current := runner.configMgr.GetCurrent()
		assert.NotNil(t, current, "Should have a current config after CommitPending")
	})

	t.Run("CompensateConfig", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)

		err = runner.StageConfig(context.Background(), tx)
		require.NoError(t, err)

		err = runner.CompensateConfig(context.Background(), tx)
		assert.NoError(t, err)
	})

	t.Run("CompensateConfig with nil transaction", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.CompensateConfig(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is nil")
	})
}

func TestRunner_PrepConfigPayload(t *testing.T) {
	t.Run("with listener configs and routes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)
		err = runner.StageConfig(context.Background(), tx)
		require.NoError(t, err)

		runner.configMgr.CommitPending()
		cfg := runner.configMgr.GetCurrent()
		require.NotNil(t, cfg)

		configs := runner.prepConfigPayload(cfg)
		assert.NotNil(t, configs)
	})

	t.Run("with empty adapter", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		configs := runner.prepConfigPayload(nil)
		assert.Empty(t, configs)
	})
}

func TestRunner_InternalHelpers(t *testing.T) {
	t.Run("prepConfigPayload with nil adapter", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		converted := runner.prepConfigPayload(nil)
		assert.Empty(t, converted, "Should handle nil adapter gracefully")
	})

	t.Run("convertRoutes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		route1, err := httpserver.NewRoute("test1", "/test1", testHandler)
		require.NoError(t, err)
		route2, err := httpserver.NewRoute("test2", "/test2", testHandler)
		require.NoError(t, err)

		adapterRoutes := []httpserver.Route{*route1, *route2}

		convertedRoutes := runner.convertRoutes(adapterRoutes)

		assert.Len(t, convertedRoutes, 2, "Should return the same number of routes")
		assert.Equal(t, "/test1", convertedRoutes[0].Path)
		assert.Equal(t, "/test2", convertedRoutes[1].Path)
	})
}

func TestRunner_WaitForClusterRunning(t *testing.T) {
	t.Run("timeout waiting for cluster", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		// Create a cluster that never becomes ready
		cluster, err := httpcluster.NewRunner()
		require.NoError(t, err)
		runner.cluster = cluster

		ctx := context.Background()
		err = runner.waitForClusterRunning(ctx, 50*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("context cancelled while waiting", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		cancelDone := make(chan struct{})
		go func() {
			defer close(cancelDone)
			assert.Eventually(t, func() bool {
				cancel()
				return true
			}, 50*time.Millisecond, 10*time.Millisecond)
		}()

		err = runner.waitForClusterRunning(ctx, 1*time.Second)
		assert.Equal(t, context.Canceled, err)

		// Wait for cancel goroutine to complete
		<-cancelDone
	})
}
