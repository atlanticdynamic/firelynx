package http

import (
	"context"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// StageConfig implements SagaParticipant.StageConfig
func (r *Runner) StageConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}
	logger := r.logger.WithGroup("StageConfig").With("tx_id", tx.GetTransactionID())
	logger.Debug("Executing HTTP configuration")

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Create adapter from transaction (transaction implements ConfigProvider)
	adapter, err := cfg.NewAdapter(tx, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create HTTP adapter: %w", err)
	}

	// Store as pending configuration
	r.configMgr.SetPending(adapter)
	logger.Debug("HTTP configuration prepared successfully")

	return nil
}

// CompensateConfig implements SagaParticipant.CompensateConfig
func (r *Runner) CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	logger := r.logger.WithGroup("CompensateConfig").With("tx_id", tx.GetTransactionID())
	logger.Debug("Compensating HTTP configuration")

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Discard pending configuration
	r.configMgr.RollbackPending()
	logger.Debug("HTTP configuration compensated successfully")
	return nil
}

// CommitConfig applies the pending configuration
// This should only be called by the saga orchestrator during TriggerReload
func (r *Runner) CommitConfig(ctx context.Context) error {
	logger := r.logger.WithGroup("CommitConfig")
	logger.Debug("Applying pending HTTP configuration")
	defer logger.Debug("CommitConfig completed")

	r.mutex.Lock()
	defer r.mutex.Unlock()

	hasPending := r.configMgr.HasPendingChanges()
	if !hasPending {
		logger.Debug("No pending HTTP configuration to apply")
		return nil
	}

	// Commit pending configuration
	r.configMgr.CommitPending()
	cfg := r.configMgr.GetCurrent()

	// Send the new configuration to the cluster
	return r.sendConfigToCluster(ctx, cfg)
}

// sendConfigToCluster converts the current adapter configuration to httpserver configs
// and sends them through the siphon channel, then waits for cluster to be ready
func (r *Runner) sendConfigToCluster(ctx context.Context, cfg *cfg.Adapter) error {
	configs := r.prepConfigPayload(cfg)

	keys := make([]string, 0, len(configs))
	for k := range configs {
		keys = append(keys, k)
	}

	if len(configs) == 0 {
		r.logger.Warn("No HTTP listeners configured")
	}

	r.logger.Debug("Sending configuration to cluster", "config", keys)

	// Send configuration through siphon with configurable timeout
	siphonCtx, siphonCancel := context.WithTimeout(ctx, r.siphonTimeout)
	defer siphonCancel()

	select {
	case r.cluster.GetConfigSiphon() <- configs:
		r.logger.Debug("Sent configuration to cluster", "config", keys)
	case <-siphonCtx.Done():
		return fmt.Errorf("timeout sending configuration to cluster after %v", r.siphonTimeout)
	}

	// Wait for cluster to finish processing and return to running state
	err := r.waitForClusterRunning(ctx, r.clusterReadyTimeout)
	if err != nil {
		return err
	}

	// Log each HTTP server that's now ready at INFO level (matching gRPC format)
	for listenerID, cfg := range configs {
		r.logger.Info("HTTP listener is ready", "id", listenerID, "addr", cfg.ListenAddr)
	}

	return nil
}

// prepConfigPayload converts the adapters into httpserver.Config objects
func (r *Runner) prepConfigPayload(cfg *cfg.Adapter) map[string]*httpserver.Config {
	configs := make(map[string]*httpserver.Config)

	if cfg != nil {
		// Convert adapter to httpserver.Config for each listener
		for _, listenerID := range cfg.GetListenerIDs() {
			logger := r.logger.WithGroup("prepConfigPayload").With("listener_id", listenerID)
			listenerCfg, ok := cfg.GetListenerConfig(listenerID)
			if !ok {
				logger.Warn("Listener config not found")
				continue
			}

			// Get routes for this listener
			adapterRoutes := cfg.GetRoutesForListener(listenerID)
			routes := r.convertRoutes(adapterRoutes)

			logger.Debug("Routes for listener",
				"adapter_routes_count", len(adapterRoutes),
				"converted_routes_count", len(routes))

			// Skip listeners without routes (httpserver requires at least one route)
			if len(routes) == 0 {
				logger.Debug("Skipping listener without routes")
				continue
			}

			logger.Debug("Configuring listener with routes",
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

	return configs
}

// convertRoutes converts adapter routes to httpserver.Route format
func (r *Runner) convertRoutes(adapterRoutes []httpserver.Route) httpserver.Routes {
	// The adapter already provides httpserver.Route objects
	// Just return them as Routes (which is []Route)
	return httpserver.Routes(adapterRoutes)
}
