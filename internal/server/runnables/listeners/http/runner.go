// Package http provides the HTTP listener implementation with SagaParticipant support.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/robbyt/go-supervisor/runnables/httpcluster"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Runner manages HTTP listeners using the httpcluster runnable with saga participant support
type Runner struct {
	cluster   *httpcluster.Runner
	configMgr *cfg.Manager
	logger    *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	// Configuration options
	siphonTimeout       time.Duration
	clusterReadyTimeout time.Duration
}

// Interface guards
var (
	_ supervisor.Runnable          = (*Runner)(nil)
	_ supervisor.Stateable         = (*Runner)(nil)
	_ supervisor.Readiness         = (*Runner)(nil)
	_ orchestrator.SagaParticipant = (*Runner)(nil)
)

// NewRunner creates a new HTTP cluster runner
func NewRunner(options ...Option) (*Runner, error) {
	r := &Runner{
		logger:              slog.Default().WithGroup("http.Runner"),
		siphonTimeout:       60 * time.Second, // timeout for sending config through cluster siphon channel
		clusterReadyTimeout: 30 * time.Second, // timeout for waiting for cluster to become ready
	}

	// Apply functional options
	for _, option := range options {
		option(r)
	}

	// Create config manager
	r.configMgr = cfg.NewManager(r.logger)

	// Create httpcluster with default unbuffered siphon channel
	cluster, err := httpcluster.NewRunner(
		httpcluster.WithLogger(r.logger.WithGroup("cluster")),
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
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()
	r.mutex.Lock()
	r.ctx = ctx
	r.cancel = ctxCancel

	// The httpcluster will start with no servers and wait for configuration
	go func() {
		if err := r.cluster.Run(ctx); err != nil {
			r.logger.Error("HTTP cluster failed", "error", err)
		}
	}()

	err := r.waitForClusterReady(ctx, r.clusterReadyTimeout)
	r.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("failed to wait for HTTP cluster to become ready: %w", err)
	}

	// block here until the run context is canceled
	<-ctx.Done()

	return r.shutdown()
}

// Stop stops the HTTP cluster runner
func (r *Runner) Stop() {
	r.logger.Debug("Stopping HTTP runner")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
}

// shutdown performs graceful shutdown of the HTTP runner
func (r *Runner) shutdown() error {
	logger := r.logger.WithGroup("shutdown")
	logger.Debug("Shutting down HTTP runner")

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Clean up any pending configuration
	if r.configMgr != nil && r.configMgr.HasPendingChanges() {
		logger.Debug("Rolling back pending configuration during shutdown")
		r.configMgr.RollbackPending()
	}

	logger.Debug("HTTP runner shutdown complete")
	return nil
}

// waitForClusterReady waits for the cluster to return a positive IsReady()
func (r *Runner) waitForClusterReady(ctx context.Context, timeout time.Duration) error {
	logger := r.logger.WithGroup("waitForClusterReady")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, timerCancel := context.WithTimeout(ctx, timeout)
	defer timerCancel()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				logger.Warn("Timeout waiting for HTTP cluster to become ready")
			}
			return timeoutCtx.Err()
		case <-ctx.Done():
			logger.Debug("Run context canceled")
			return ctx.Err()
		case <-ticker.C:
			// periodically check if the cluster is ready
			if r.cluster.IsReady() {
				logger.Debug("HTTP cluster is now ready")
				return nil
			}
		}
	}
}
