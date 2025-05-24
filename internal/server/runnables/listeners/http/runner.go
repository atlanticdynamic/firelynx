// Package http provides the HTTP listener implementation with SagaParticipant support.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/httpserver"
	"github.com/robbyt/go-supervisor/runnables/composite"
)

// Runner manages HTTP listeners and participates in the saga pattern
type Runner struct {
	configMgr *cfg.Manager
	runner    *composite.Runner[*httpserver.HTTPServer]
	logger    *slog.Logger

	parentCtx context.Context
	mutex     sync.RWMutex
}

// NewRunner creates a new HTTP runner with optional configuration
func NewRunner(options ...Option) (*Runner, error) {
	r := &Runner{
		logger:    slog.Default().WithGroup("http.Runner"),
		parentCtx: context.Background(),
	}

	// Apply functional options
	for _, option := range options {
		option(r)
	}

	// Create config manager using the configured logger
	r.configMgr = cfg.NewManager(r.logger)

	// Create config callback for composite runner
	configCallback := func() (*composite.Config[*httpserver.HTTPServer], error) {
		return r.buildCompositeConfig()
	}

	// Create composite runner
	var err error
	r.runner, err = composite.NewRunner(configCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite runner: %w", err)
	}

	return r, nil
}

// buildCompositeConfig builds a configuration for the composite runner
func (r *Runner) buildCompositeConfig() (*composite.Config[*httpserver.HTTPServer], error) {
	r.mutex.RLock()
	adapter := r.configMgr.GetCurrent()
	r.mutex.RUnlock()

	if adapter == nil {
		// Create empty config - no HTTP listeners initially
		config, err := composite.NewConfig[*httpserver.HTTPServer]("http-listeners", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create empty config: %w", err)
		}
		return config, nil
	}

	// Create entries from adapter
	var entries []composite.RunnableEntry[*httpserver.HTTPServer]

	for _, listenerID := range adapter.GetListenerIDs() {
		listenerCfg, ok := adapter.GetListenerConfig(listenerID)
		if !ok {
			r.logger.Warn("Listener config not found", "listener_id", listenerID)
			continue
		}

		routes := adapter.GetRoutesForListener(listenerID)

		// Convert timeout fields to HTTPTimeoutOptions
		timeouts := httpserver.HTTPTimeoutOptions{
			ReadTimeout:  listenerCfg.ReadTimeout,
			WriteTimeout: listenerCfg.WriteTimeout,
			IdleTimeout:  listenerCfg.IdleTimeout,
			DrainTimeout: listenerCfg.DrainTimeout,
		}

		// Create HTTP server
		server, err := httpserver.NewHTTPServer(
			listenerCfg.ID,
			listenerCfg.Address,
			routes,
			timeouts,
			r.logger.With("listener", listenerID),
		)
		if err != nil {
			r.logger.Error("Failed to create HTTP server", "id", listenerID, "error", err)
			continue
		}

		// Add to entries
		entry := composite.RunnableEntry[*httpserver.HTTPServer]{
			Runnable: server,
			Config:   nil, // No additional config needed
		}
		entries = append(entries, entry)
	}

	// Create composite config
	return composite.NewConfig("http-listeners", entries)
}

// String returns a unique identifier for this runner
func (r *Runner) String() string {
	return "HTTPRunner"
}

// Run starts the HTTP runner
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting HTTP runner")
	return r.runner.Run(ctx)
}

// Stop stops the HTTP runner
func (r *Runner) Stop() {
	r.logger.Debug("Stopping HTTP runner")
	r.runner.Stop()
}

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
	return r.runner.GetState()
}

// IsRunning returns whether the runner is running
func (r *Runner) IsRunning() bool {
	return r.runner.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
	return r.runner.GetStateChan(ctx)
}

// ExecuteConfig implements SagaParticipant.ExecuteConfig
func (r *Runner) ExecuteConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Executing HTTP configuration", "tx_id", tx.GetTransactionID())

	// Create adapter from transaction
	adapter, err := cfg.NewAdapter(tx, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create HTTP adapter: %w", err)
	}

	// Validate the adapter configuration
	if err := r.validateAdapter(adapter); err != nil {
		return fmt.Errorf("invalid HTTP configuration: %w", err)
	}

	// Store as pending configuration
	r.configMgr.SetPending(adapter)
	r.logger.Debug("HTTP configuration prepared successfully", "tx_id", tx.GetTransactionID())

	return nil
}

// validateAdapter validates the adapter configuration
func (r *Runner) validateAdapter(adapter *cfg.Adapter) error {
	// For now, we just check that the adapter has valid listeners and routes
	// In a real implementation, we would validate addresses, routes, etc.
	if len(adapter.GetListenerIDs()) == 0 {
		r.logger.Warn("Adapter has no HTTP listeners")
		// Allow empty configuration (for removing all HTTP listeners)
	}

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

// ApplyPendingConfig applies the pending configuration
// This should only be called by the saga orchestrator during TriggerReload
func (r *Runner) ApplyPendingConfig(ctx context.Context) error {
	// Check if we have pending changes while holding the lock
	r.mutex.Lock()
	hasPending := r.configMgr.HasPendingChanges()
	if !hasPending {
		r.mutex.Unlock()
		return nil
	}

	r.logger.Debug("Applying pending HTTP configuration")

	// Commit pending configuration to make it current
	// This will cause the composite runner to reload on next getConfig call
	r.configMgr.CommitPending()
	r.mutex.Unlock()

	// Force reload of composite runner without holding the lock
	// This avoids deadlock when buildCompositeConfig is called
	r.runner.Reload()

	return nil
}
