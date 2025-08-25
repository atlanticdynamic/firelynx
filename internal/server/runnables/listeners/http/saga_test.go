package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
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
		err = runner.StageConfig(t.Context(), tx)
		require.NoError(t, err)
	})

	t.Run("StageConfig with nil transaction", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.StageConfig(t.Context(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is nil")
	})

	t.Run("CommitConfig with no pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.CommitConfig(t.Context())
		require.NoError(t, err)
	})

	t.Run("CommitConfig with pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)
		err = runner.StageConfig(t.Context(), tx)
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

		err = runner.StageConfig(t.Context(), tx)
		require.NoError(t, err)

		err = runner.CompensateConfig(t.Context(), tx)
		require.NoError(t, err)
	})

	t.Run("CompensateConfig with nil transaction", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		err = runner.CompensateConfig(t.Context(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is nil")
	})
}

func TestRunner_PrepConfigPayload(t *testing.T) {
	t.Run("with listener configs and routes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		tx := createMockTransaction(t)
		err = runner.StageConfig(t.Context(), tx)
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

	t.Run("with multiple listeners and routes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Create routes using NewRouteFromHandlerFunc
		route1, err := httpserver.NewRouteFromHandlerFunc("route1", "/api/v1", testHandler)
		require.NoError(t, err)
		route2, err := httpserver.NewRouteFromHandlerFunc("route2", "/api/v2", testHandler)
		require.NoError(t, err)
		route3, err := httpserver.NewRouteFromHandlerFunc("route3", "/health", testHandler)
		require.NoError(t, err)

		// Create adapter with multiple listeners
		adapter := &cfg.Adapter{
			TxID: "test-tx-123",
			Listeners: map[string]cfg.ListenerConfig{
				"listener1": {
					ID:           "listener1",
					Address:      ":8080",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  60 * time.Second,
					DrainTimeout: 5 * time.Second,
				},
				"listener2": {
					ID:           "listener2",
					Address:      ":8081",
					ReadTimeout:  45 * time.Second,
					WriteTimeout: 45 * time.Second,
					IdleTimeout:  90 * time.Second,
					DrainTimeout: 10 * time.Second,
				},
			},
			Routes: map[string][]httpserver.Route{
				"listener1": {*route1, *route2},
				"listener2": {*route3},
			},
		}

		configs := runner.prepConfigPayload(adapter)

		// Verify we get configs for both listeners
		assert.Len(t, configs, 2)

		// Verify listener1 config
		cfg1, ok := configs["listener1"]
		assert.True(t, ok)
		assert.Equal(t, ":8080", cfg1.ListenAddr)
		assert.Len(t, cfg1.Routes, 2)
		assert.Equal(t, "/api/v1", cfg1.Routes[0].Path)
		assert.Equal(t, "/api/v2", cfg1.Routes[1].Path)
		assert.Equal(t, 30*time.Second, cfg1.ReadTimeout)
		assert.Equal(t, 30*time.Second, cfg1.WriteTimeout)
		assert.Equal(t, 60*time.Second, cfg1.IdleTimeout)
		assert.Equal(t, 5*time.Second, cfg1.DrainTimeout)

		// Verify listener2 config
		cfg2, ok := configs["listener2"]
		assert.True(t, ok)
		assert.Equal(t, ":8081", cfg2.ListenAddr)
		assert.Len(t, cfg2.Routes, 1)
		assert.Equal(t, "/health", cfg2.Routes[0].Path)
		assert.Equal(t, 45*time.Second, cfg2.ReadTimeout)
		assert.Equal(t, 45*time.Second, cfg2.WriteTimeout)
		assert.Equal(t, 90*time.Second, cfg2.IdleTimeout)
		assert.Equal(t, 10*time.Second, cfg2.DrainTimeout)
	})

	t.Run("listener with no routes is skipped", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		route1, err := httpserver.NewRouteFromHandlerFunc("route1", "/api", testHandler)
		require.NoError(t, err)

		adapter := &cfg.Adapter{
			TxID: "test-tx-456",
			Listeners: map[string]cfg.ListenerConfig{
				"listener1": {
					ID:      "listener1",
					Address: ":8080",
				},
				"listener2": {
					ID:      "listener2",
					Address: ":8081",
				},
			},
			Routes: map[string][]httpserver.Route{
				"listener1": {*route1},
				"listener2": {}, // Empty routes for listener2
			},
		}

		configs := runner.prepConfigPayload(adapter)

		// Only listener1 should be included
		assert.Len(t, configs, 1)
		_, ok := configs["listener1"]
		assert.True(t, ok)
		_, ok = configs["listener2"]
		assert.False(t, ok, "listener2 should be skipped due to no routes")
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
		route1, err := httpserver.NewRouteFromHandlerFunc("test1", "/test1", testHandler)
		require.NoError(t, err)
		route2, err := httpserver.NewRouteFromHandlerFunc("test2", "/test2", testHandler)
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

		ctx := t.Context()
		err = runner.waitForClusterRunning(ctx, 50*time.Millisecond)
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("context cancelled while waiting", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		// Use already cancelled context for deterministic behavior
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		err = runner.waitForClusterRunning(ctx, 1*time.Second)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("cluster becomes ready", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// Start cluster and capture error
		clusterErr := make(chan error, 1)
		go func() {
			clusterErr <- runner.cluster.Run(ctx)
		}()

		// Use assert.Eventually to wait for cluster to be ready
		assert.Eventually(t, func() bool {
			return runner.cluster.IsRunning()
		}, 1*time.Second, 10*time.Millisecond)

		// Now waitForClusterRunning should succeed immediately
		err = runner.waitForClusterRunning(ctx, 100*time.Millisecond)
		require.NoError(t, err)

		// Clean shutdown
		cancel()
		select {
		case err := <-clusterErr:
			require.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("cluster.Run should have returned after context cancellation")
		}
	})
}

func TestRunner_SendConfigToCluster(t *testing.T) {
	t.Run("siphon timeout", func(t *testing.T) {
		runner, err := NewRunner(WithSiphonTimeout(50 * time.Millisecond))
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		route1, err := httpserver.NewRouteFromHandlerFunc("route1", "/test", testHandler)
		require.NoError(t, err)

		adapter := &cfg.Adapter{
			TxID: "test-tx-002",
			Listeners: map[string]cfg.ListenerConfig{
				"listener1": {
					ID:      "listener1",
					Address: ":8080",
				},
			},
			Routes: map[string][]httpserver.Route{
				"listener1": {*route1},
			},
		}

		ctx := t.Context()

		// Don't consume from siphon to trigger timeout
		err = runner.sendConfigToCluster(ctx, adapter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout sending configuration to cluster")
	})

	t.Run("context cancelled during send", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		route1, err := httpserver.NewRouteFromHandlerFunc("route1", "/test", testHandler)
		require.NoError(t, err)

		adapter := &cfg.Adapter{
			TxID: "test-tx-003",
			Listeners: map[string]cfg.ListenerConfig{
				"listener1": {
					ID:      "listener1",
					Address: ":8080",
				},
			},
			Routes: map[string][]httpserver.Route{
				"listener1": {*route1},
			},
		}

		// Use already cancelled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err = runner.sendConfigToCluster(ctx, adapter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout sending configuration to cluster")
	})
}
