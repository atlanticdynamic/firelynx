// Package txmgr implements the transaction manager for configuration updates
// and adapters between domain config and runtime components.
//
// HTTP Listener Rewrite Plan:
// According to the HTTP listener rewrite plan, HTTP-specific configuration logic
// in this package will be moved to the HTTP listener package where it will implement
// the SagaParticipant interface. This will allow each SagaParticipant to handle
// its own configuration extraction and management, keeping this package focused
// on orchestration rather than HTTP-specific details.
package txmgr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Interface guards: ensure Runner implements these interfaces
var (
	_ supervisor.Runnable  = (*Runner)(nil)
	_ supervisor.Stateable = (*Runner)(nil)
)

// These are injected by goreleaser and correspond to the version of the build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

// Version returns a formatted string with version information.
func Version() string {
	return fmt.Sprintf("version %s (commit %s) built by %s on %s", version, commit, builtBy, date)
}

// ConfigChannelProvider defines the interface for getting a channel of validated config transactions
type ConfigChannelProvider interface {
	GetConfigChan() <-chan *transaction.ConfigTransaction
}

// Runner implements the core server coordinator that manages configuration
// lifecycle and app collection.
//
// Note: The HTTP-specific references in this comment are outdated. According to the
// HTTP listener rewrite plan, HTTP server management functionality will be moved
// to the HTTP listener package as a dedicated SagaParticipant implementation.
type Runner struct {
	// Required dependencies
	appCollection *apps.AppCollection
	logger        *slog.Logger

	// Configuration transaction management
	sagaOrchestrator *SagaOrchestrator
	configProvider   ConfigChannelProvider

	// Internal state

	// Control channels
	serverErrors chan error

	// Synchronization
	wg sync.WaitGroup

	// Context handling
	runCtx    context.Context
	runCancel context.CancelFunc
	parentCtx context.Context

	// State management
	fsm finitestate.Machine
}

// NewRunner creates a new core runner that coordinates configuration and services.
// It follows the functional options pattern for configuration.
func NewRunner(
	sagaOrchestrator *SagaOrchestrator,
	configProvider ConfigChannelProvider,
	opts ...Option,
) (*Runner, error) {
	if sagaOrchestrator == nil {
		return nil, errors.New("saga orchestrator cannot be nil")
	}
	if configProvider == nil {
		return nil, errors.New("config provider cannot be nil")
	}
	// Create initial empty app collection
	initialApps, err := apps.NewAppCollection([]apps.App{})
	if err != nil {
		return nil, fmt.Errorf("failed to create initial app collection: %w", err)
	}

	// Initialize with default options
	runner := &Runner{
		appCollection:    initialApps,
		logger:           slog.Default().WithGroup("txmgr.Runner"),
		sagaOrchestrator: sagaOrchestrator,
		configProvider:   configProvider,
		serverErrors:     make(chan error, 10),
		parentCtx:        context.Background(),
	}

	// Apply options
	for _, opt := range opts {
		opt(runner)
	}

	// Initialize the finite state machine
	fsmLogger := runner.logger.WithGroup("fsm")
	fsm, err := finitestate.New(fsmLogger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}
	runner.fsm = fsm

	return runner, nil
}

// Run implements the supervisor.Runnable interface.
// It initializes and starts all server components, blocking until
// the context is cancelled or Stop is called.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
		return fmt.Errorf("failed to transition to booting state: %w", err)
	}

	// Create a cancellable context
	r.runCtx, r.runCancel = context.WithCancel(ctx)

	// Start monitoring for errors
	r.wg.Add(1)
	go r.monitorErrors()

	// Start monitoring config transactions from cfgservice
	r.wg.Add(1)
	go r.monitorConfigTransactions()

	// Transition to running state
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// Block until context is cancelled or Stop is called
	select {
	case <-r.parentCtx.Done():
		r.logger.Debug("Parent context canceled")
		// Cancel run context to stop goroutines since Stop() wasn't called
		r.runCancel()
	case <-r.runCtx.Done():
		r.logger.Debug("Run context canceled")
	}

	r.logger.Info("Transaction manager shutting down")

	// Ensure we transition to stopping state first
	if r.fsm.GetState() != finitestate.StatusStopping {
		if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
			r.logger.Error("Failed to transition to stopping state", "error", err)
		}
	}

	// Wait for all goroutines to finish
	r.wg.Wait()

	// Then transition to stopped
	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		return fmt.Errorf("failed to transition to stopped state: %w", err)
	}

	return nil
}

// monitorErrors watches the error channel and logs errors.
func (r *Runner) monitorErrors() {
	defer r.wg.Done()

	for {
		select {
		case <-r.runCtx.Done():
			return
		case err := <-r.serverErrors:
			if err != nil {
				r.logger.Error("Server error", "error", err)
			}
		}
	}
}

// monitorConfigTransactions watches for validated config transactions from cfgservice
// and processes them through the saga orchestrator.
func (r *Runner) monitorConfigTransactions() {
	defer r.wg.Done()

	configChan := r.configProvider.GetConfigChan()
	for {
		select {
		case <-r.runCtx.Done():
			return
		case tx, ok := <-configChan:
			if !ok {
				r.logger.Info("Config channel closed, stopping config transaction monitoring")
				return
			}
			if tx == nil {
				continue
			}

			r.logger.Debug("Received validated config transaction", "id", tx.ID)

			// Add transaction to storage first
			if err := r.sagaOrchestrator.txStorage.Add(tx); err != nil {
				r.logger.Error("Failed to add config transaction to storage",
					"id", tx.ID, "error", err)
				r.serverErrors <- fmt.Errorf("failed to store transaction %s: %w", tx.ID, err)
				continue
			}

			// Process the transaction through the saga orchestrator
			if err := r.sagaOrchestrator.ProcessTransaction(r.runCtx, tx); err != nil {
				r.logger.Error("Failed to process config transaction via saga orchestrator",
					"id", tx.ID, "error", err)
				r.serverErrors <- fmt.Errorf("saga processing failed for transaction %s: %w", tx.ID, err)
			} else {
				r.logger.Info("Successfully processed config transaction via saga orchestrator", "id", tx.ID)
			}
		}
	}
}

// String returns the name of this runnable component.
func (r *Runner) String() string {
	return "txmgr.Runner"
}

// Stop gracefully stops all server components.
func (r *Runner) Stop() {
	r.logger.Debug("Stopping transaction manager")
	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
		// Continue with shutdown despite the state transition error
	}
	r.runCancel()
}
