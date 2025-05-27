// Package http provides the HTTP listener implementation with SagaParticipant support.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/robbyt/go-supervisor/runnables/httpcluster"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Runner manages HTTP listeners using the httpcluster runnable with saga participant support
type Runner struct {
	cluster   *httpcluster.Runner
	configMgr *cfg.Manager
	logger    *slog.Logger

	parentCtx context.Context
	mutex     sync.RWMutex

	// Configuration options
	siphonTimeout time.Duration
}

// Interface guards
var (
	_ supervisor.Runnable          = (*Runner)(nil)
	_ supervisor.Stateable         = (*Runner)(nil)
	_ orchestrator.SagaParticipant = (*Runner)(nil)
)

// NewRunner creates a new HTTP cluster runner
func NewRunner(options ...Option) (*Runner, error) {
	r := &Runner{
		logger:        slog.Default().WithGroup("http.Runner"),
		parentCtx:     context.Background(),
		siphonTimeout: 30 * time.Second, // Default timeout
	}

	// Apply functional options
	for _, option := range options {
		option(r)
	}

	// Create config manager
	r.configMgr = cfg.NewManager(r.logger)

	// Create httpcluster with default unbuffered siphon channel
	cluster, err := httpcluster.NewRunner(
		httpcluster.WithContext(r.parentCtx),
		httpcluster.WithLogger(r.logger.WithGroup("cluster")),
		// Siphon buffer defaults to 0 (unbuffered)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create httpcluster runner: %w", err)
	}

	r.cluster = cluster
	return r, nil
}

// String returns a unique identifier for this runner
func (r *Runner) String() string {
	return "HTTPRunner"
}

// Run starts the HTTP cluster runner
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting HTTP runner")

	// The httpcluster will start with no servers and wait for configuration
	// through the siphon channel via the saga pattern
	return r.cluster.Run(ctx)
}

// Stop stops the HTTP cluster runner
func (r *Runner) Stop() {
	r.logger.Debug("Stopping HTTP runner")
	r.cluster.Stop()
}

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.cluster.GetState()
}

// IsRunning returns whether the runner is running
func (r *Runner) IsRunning() bool {
	return r.cluster.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.cluster.GetStateChan(ctx)
}

// StageConfig implements SagaParticipant.StageConfig
func (r *Runner) StageConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Executing HTTP configuration", "tx_id", tx.GetTransactionID())

	// Create adapter from transaction (transaction implements ConfigProvider)
	adapter, err := cfg.NewAdapter(tx, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create HTTP adapter: %w", err)
	}

	// Store as pending configuration
	r.configMgr.SetPending(adapter)
	r.logger.Debug("HTTP configuration prepared successfully", "tx_id", tx.GetTransactionID())

	return nil
}

// CompensateConfig implements SagaParticipant.CompensateConfig
func (r *Runner) CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Compensating HTTP configuration", "tx_id", tx.GetTransactionID())

	// Discard pending configuration
	r.configMgr.RollbackPending()

	return nil
}

// CommitConfig applies the pending configuration
// This should only be called by the saga orchestrator during TriggerReload
func (r *Runner) CommitConfig(ctx context.Context) error {
	r.mutex.Lock()
	hasPending := r.configMgr.HasPendingChanges()
	if !hasPending {
		r.mutex.Unlock()
		return nil
	}

	r.logger.Debug("Applying pending HTTP configuration")

	// Commit pending configuration
	r.configMgr.CommitPending()
	r.mutex.Unlock()

	// Send the new configuration to the cluster
	return r.sendCurrentConfig(ctx)
}

// sendCurrentConfig converts the current adapter configuration to httpserver configs
// and sends them through the siphon channel
func (r *Runner) sendCurrentConfig(ctx context.Context) error {
	r.mutex.RLock()
	adapter := r.configMgr.GetCurrent()
	r.mutex.RUnlock()

	configs := make(map[string]*httpserver.Config)

	if adapter != nil {
		// Convert adapter to httpserver.Config for each listener
		for _, listenerID := range adapter.GetListenerIDs() {
			listenerCfg, ok := adapter.GetListenerConfig(listenerID)
			if !ok {
				r.logger.Warn("Listener config not found", "listener_id", listenerID)
				continue
			}

			// Get routes for this listener
			adapterRoutes := adapter.GetRoutesForListener(listenerID)
			routes := r.convertRoutes(adapterRoutes)

			r.logger.Debug("Routes for listener",
				"listener_id", listenerID,
				"adapter_routes_count", len(adapterRoutes),
				"converted_routes_count", len(routes))

			// Skip listeners without routes (httpserver requires at least one route)
			if len(routes) == 0 {
				r.logger.Debug("Skipping listener without routes", "listener_id", listenerID)
				continue
			}

			r.logger.Debug("Configuring listener with routes",
				"listener_id", listenerID,
				"address", listenerCfg.Address,
				"route_count", len(routes))

			// Create httpserver.Config
			serverCfg := &httpserver.Config{
				ListenAddr:   listenerCfg.Address,
				Routes:       routes,
				ReadTimeout:  listenerCfg.ReadTimeout,
				WriteTimeout: listenerCfg.WriteTimeout,
				IdleTimeout:  listenerCfg.IdleTimeout,
				DrainTimeout: listenerCfg.DrainTimeout,
			}

			configs[listenerID] = serverCfg
		}
	}

	// Send configuration through siphon with configurable timeout
	ctx, cancel := context.WithTimeout(ctx, r.siphonTimeout)
	defer cancel()

	select {
	case r.cluster.GetConfigSiphon() <- configs:
		r.logger.Debug("Sent configuration to cluster", "listeners", len(configs))
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout sending configuration to cluster after %v", r.siphonTimeout)
	}
}

// convertRoutes converts adapter routes to httpserver.Route format
func (r *Runner) convertRoutes(adapterRoutes []httpserver.Route) httpserver.Routes {
	// The adapter already provides httpserver.Route objects
	// Just return them as Routes (which is []Route)
	return httpserver.Routes(adapterRoutes)
}
