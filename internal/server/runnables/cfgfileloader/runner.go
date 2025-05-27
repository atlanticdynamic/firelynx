package cfgfileloader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/robbyt/go-supervisor/supervisor"
)

var (
	_ supervisor.Runnable   = (*Runner)(nil)
	_ supervisor.Reloadable = (*Runner)(nil)
	_ supervisor.Stateable  = (*Runner)(nil)
)

type Runner struct {
	filePath             string
	lastValidTransaction atomic.Pointer[transaction.ConfigTransaction]

	// runCtx is passed in to Run, and is used to cancel the Run loop
	runCtx    context.Context
	runCancel context.CancelFunc
	parentCtx context.Context
	txSiphon  chan<- *transaction.ConfigTransaction
	fsm       finitestate.Machine
	logger    *slog.Logger
}

// NewRunner creates a new Runner instance used for loading cfg files from disk
func NewRunner(
	filePath string,
	txSiphon chan<- *transaction.ConfigTransaction,
	opts ...Option,
) (*Runner, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	if txSiphon == nil {
		return nil, fmt.Errorf("transaction siphon cannot be nil")
	}

	runner := &Runner{
		filePath:             filePath,
		txSiphon:             txSiphon,
		logger:               slog.Default().WithGroup("cfgfileloader.Runner"),
		lastValidTransaction: atomic.Pointer[transaction.ConfigTransaction]{},
		parentCtx:            context.Background(),
		runCtx:               context.Background(),
	}

	// Apply functional options
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

// String implements the supervisor.Runnable interface
func (r *Runner) String() string {
	return "cfgfileloader.Runner"
}

// Run implements the supervisor.Runnable interface
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Debug("Starting Runner")

	if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
		return fmt.Errorf("failed to transition to booting state: %w", err)
	}

	r.runCtx, r.runCancel = context.WithCancel(ctx)

	// Load the initial configuration
	if err := r.boot(); err != nil {
		if stateErr := r.fsm.Transition(finitestate.StatusError); stateErr != nil {
			r.logger.Error("Failed to transition to error state", "error", stateErr)
		}
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Transition to running state
	if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// block here waiting for a context cancellation
	select {
	case <-r.parentCtx.Done():
		r.logger.Debug("Parent context canceled")
	case <-r.runCtx.Done():
		r.logger.Debug("Run context canceled")
	}

	r.logger.Info("Runner shutting down")

	// Ensure we transition to stopping state first
	if r.fsm.GetState() != finitestate.StatusStopping {
		if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
			r.logger.Error("Failed to transition to stopping state", "error", err)
		}
	}

	// Then transition to stopped
	if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
		return fmt.Errorf("failed to transition to stopped state: %w", err)
	}

	// Clear the last loaded config
	r.lastValidTransaction.Store(nil)

	return nil
}

// boot loads the initial configuration from disk
func (r *Runner) boot() error {
	if r.filePath == "" {
		return errors.New("no config path set")
	}

	cfg, err := r.loadConfigFromDisk()
	if err != nil {
		return err
	}
	if cfg == nil {
		return errors.New("no config loaded")
	}

	tx, err := r.validate(cfg)
	if err != nil {
		return err
	}

	r.lastValidTransaction.Store(tx)
	// Send transaction to siphon in a goroutine to avoid blocking startup
	// TODO: consider running this without the goroutine
	go func() {
		select {
		case r.txSiphon <- tx:
			r.logger.Debug("Initial transaction sent to siphon", "id", tx.ID)
		case <-r.runCtx.Done():
			r.logger.Debug("Context cancelled while sending initial transaction")
		}
	}()

	return nil
}

// loadConfigFromDisk loads the configuration from disk
func (r *Runner) loadConfigFromDisk() (*config.Config, error) {
	return config.NewConfig(r.filePath)
}

// validate validates the domain config and returns a config transaction, ready for future processing.
func (r *Runner) validate(cfg *config.Config) (*transaction.ConfigTransaction, error) {
	tx, err := transaction.FromFile(r.filePath, cfg, r.logger.Handler())
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	if err := tx.RunValidation(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return tx, nil
}

// Stop implements the supervisor.Runnable interface
func (r *Runner) Stop() {
	r.logger.Debug("Stopping Runner")
	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
		// Continue with shutdown despite the state transition error
	}
	r.runCancel()
}

// Reload implements the supervisor.Reloadable interface
func (r *Runner) Reload() {
	r.logger.Debug("Starting Reload...")
	defer func() {
		r.logger.Debug("Reload completed")
	}()

	if r.filePath == "" {
		r.logger.Warn("No config path set, skipping reload")
		return
	}

	newCfg, err := r.loadConfigFromDisk()
	if err != nil {
		r.logger.Error("Failed to reload config", "error", err)
		return
	}

	if newCfg == nil {
		r.logger.Error("No config loaded, skipping reload")
		return
	}

	// Only broadcast if config has changed
	reloadNeeded := true
	oldCfg := r.getConfig()
	if oldCfg != nil {
		reloadNeeded = !oldCfg.Equals(newCfg)
	}
	if !reloadNeeded {
		r.logger.Debug("Config unchanged, skipping broadcast")
		return
	}

	tx, err := r.validate(newCfg)
	if err != nil {
		r.logger.Error("Failed to validate config", "error", err)
		return
	}
	if tx == nil {
		r.logger.Error("No valid transaction created, skipping broadcast")
		return
	}

	r.lastValidTransaction.Store(tx)

	select {
	case r.txSiphon <- tx:
		r.logger.Debug("Config changed, transaction sent to siphon", "id", tx.ID)
	case <-r.runCtx.Done():
		r.logger.Debug("Context cancelled while sending transaction")
	}
}

// getConfig returns the last config successfully loaded and validated, or nil if none
func (r *Runner) getConfig() *config.Config {
	tx := r.lastValidTransaction.Load()
	if tx == nil {
		return nil
	}
	return tx.GetConfig()
}
