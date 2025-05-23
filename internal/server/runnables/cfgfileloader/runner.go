package cfgfileloader

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	filePath         string
	lastConfigLoaded atomic.Pointer[transaction.ConfigTransaction]

	logger *slog.Logger
	fsm    finitestate.Machine

	runCtx    context.Context
	runCancel context.CancelFunc
	parentCtx context.Context

	configSubscribers sync.Map
	subscriberCounter atomic.Uint64
}

func NewRunner(filePath string, opts ...Option) (*Runner, error) {
	// Initialize with default options
	runner := &Runner{
		filePath:         filePath,
		logger:           slog.Default().WithGroup("cfgfileloader.Runner"),
		lastConfigLoaded: atomic.Pointer[transaction.ConfigTransaction]{},
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

func (r *Runner) String() string {
	return "cfgfileloader.Runner"
}

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
	r.lastConfigLoaded.Store(nil)

	return nil
}

func (r *Runner) boot() error {
	if r.filePath == "" {
		r.logger.Warn("No config path set, skipping boot")
		return nil
	}

	cfg, err := r.loadConfigFromDisk()
	if err != nil {
		return err
	}

	if cfg != nil {
		tx, err := r.validate(cfg)
		if err != nil {
			return err
		}

		r.lastConfigLoaded.Store(tx)
		r.broadcastConfigTransaction(tx)
	}

	return nil
}

func (r *Runner) loadConfigFromDisk() (*config.Config, error) {
	return config.NewConfig(r.filePath)
}

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

func (r *Runner) Stop() {
	r.logger.Debug("Stopping Runner")
	if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
		r.logger.Error("Failed to transition to stopping state", "error", err)
		// Continue with shutdown despite the state transition error
	}
	r.runCancel()
}

func (r *Runner) Reload() {
	r.logger.Debug("Starting Reload...")
	if r.filePath == "" {
		r.logger.Warn("No config path set, skipping reload")
		return
	}

	newCfg, err := r.loadConfigFromDisk()
	if err != nil {
		r.logger.Error("Failed to reload config", "error", err)
		return
	}

	if newCfg != nil {
		// Only broadcast if config has changed
		oldTx := r.lastConfigLoaded.Load()
		var configChanged bool
		if oldTx == nil {
			configChanged = true
		} else {
			oldCfg := oldTx.GetConfig()
			configChanged = oldCfg == nil || !oldCfg.Equals(newCfg)
		}

		if configChanged {
			tx, err := r.validate(newCfg)
			if err != nil {
				r.logger.Error("Failed to validate config", "error", err)
				return
			}

			r.lastConfigLoaded.Store(tx)
			r.broadcastConfigTransaction(tx)
			r.logger.Debug("Config changed, broadcasted to subscribers")
		} else {
			r.logger.Debug("Config unchanged, skipping broadcast")
		}
	}
	r.logger.Debug("Reload completed")
}

func (r *Runner) GetConfig() *config.Config {
	tx := r.lastConfigLoaded.Load()
	if tx == nil {
		return nil
	}
	return tx.GetConfig()
}

func (r *Runner) GetConfigChan() <-chan *transaction.ConfigTransaction {
	// TODO: Consider removing buffer or making it configurable for better backpressure control
	ch := make(chan *transaction.ConfigTransaction, 1)

	// Send current transaction immediately if available
	if current := r.lastConfigLoaded.Load(); current != nil {
		select {
		case ch <- current:
		default: // channel full, skip
		}
	}

	// Register for future updates
	id := r.subscriberCounter.Add(1)
	r.configSubscribers.Store(id, ch)

	// Cleanup when Runner's parent context is done
	go func() {
		<-r.parentCtx.Done()
		r.configSubscribers.Delete(id)
		close(ch)
	}()

	return ch
}

func (r *Runner) broadcastConfigTransaction(tx *transaction.ConfigTransaction) {
	if tx == nil {
		return
	}

	r.configSubscribers.Range(func(key, value interface{}) bool {
		ch, ok := value.(chan *transaction.ConfigTransaction)
		if !ok {
			r.logger.Error("Invalid subscriber channel type", "key", key)
			r.configSubscribers.Delete(key)
			return true
		}

		select {
		case ch <- tx:
			r.logger.Debug("Config transaction sent to subscriber", "subscriber_id", key)
		default:
			r.logger.Warn("Subscriber channel full, skipping", "subscriber_id", key)
		}
		return true
	})
}
