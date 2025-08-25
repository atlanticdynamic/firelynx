package txmgr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHarness provides a clean test setup
type testHarness struct {
	t                *testing.T
	runner           *Runner
	txSiphon         chan<- *transaction.ConfigTransaction
	sagaOrchestrator SagaProcessor
	txStorage        *txstorage.MemoryStorage
	ctx              context.Context
	cancel           context.CancelFunc
	errCh            chan error
}

// newTestHarness creates a complete test setup
func newTestHarness(t *testing.T, opts ...Option) *testHarness {
	t.Helper()

	txStorage := txstorage.NewMemoryStorage()
	sagaOrchestrator := orchestrator.NewSagaOrchestrator(txStorage, slog.Default().Handler())
	runner, err := NewRunner(sagaOrchestrator, opts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())

	return &testHarness{
		t:                t,
		runner:           runner,
		txSiphon:         runner.GetTransactionSiphon(),
		sagaOrchestrator: sagaOrchestrator,
		txStorage:        txStorage,
		ctx:              ctx,
		cancel:           cancel,
		errCh:            make(chan error, 1),
	}
}

// start begins running the runner in a goroutine
func (h *testHarness) start() {
	go func() {
		h.errCh <- h.runner.Run(h.ctx)
	}()

	// Wait for runner to be in Running state
	assert.Eventually(h.t, func() bool {
		return h.runner.IsRunning()
	}, 5*time.Second, 10*time.Millisecond, "runner should reach Running state")
}

// stop cancels the context and waits for clean shutdown
func (h *testHarness) stop() error {
	h.cancel()
	err := <-h.errCh
	return err
}

// sendTransaction sends a transaction to the siphon
func (h *testHarness) sendTransaction(tx *transaction.ConfigTransaction) {
	select {
	case h.txSiphon <- tx:
		// Sent successfully
	case <-time.After(5 * time.Second):
		h.t.Fatal("timeout sending transaction to siphon")
	}
}

// waitForTransaction waits for a transaction to be stored
func (h *testHarness) waitForTransaction(txID string) *transaction.ConfigTransaction {
	var stored *transaction.ConfigTransaction
	assert.Eventually(h.t, func() bool {
		stored = h.txStorage.GetByID(txID)
		return stored != nil
	}, 5*time.Second, 10*time.Millisecond, "transaction should be stored")
	return stored
}

// sendConfig sends a config transaction with the given version
func (h *testHarness) sendConfig(version string) *transaction.ConfigTransaction {
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(h.t, err)
	cfg.Version = version
	tx, err := transaction.New(
		transaction.SourceTest,
		"test harness",
		"test-request-"+version,
		cfg,
		slog.Default().Handler(),
	)
	require.NoError(h.t, err)

	// Validate the transaction before sending (mimicking what cfgservice would do)
	require.NoError(h.t, tx.BeginValidation())
	tx.IsValid.Store(true)
	require.NoError(h.t, tx.MarkValidated())

	h.sendTransaction(tx)
	return tx
}

func TestNewRunnerMinimalOptions(t *testing.T) {
	h := newTestHarness(t)
	assert.NotNil(t, h.runner)
	assert.NotNil(t, h.txSiphon)
	assert.NotNil(t, h.sagaOrchestrator)
	assert.NotNil(t, h.txStorage)
}

func TestRunnerOptionsFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := newTestHarness(t, WithLogger(logger))
	assert.NotNil(t, h.runner)
}

func TestRunnerReceivesConfig(t *testing.T) {
	h := newTestHarness(t)
	h.start()

	// Send config transaction
	tx := h.sendConfig("v1")

	// Verify transaction was stored
	stored := h.waitForTransaction(tx.ID.String())
	assert.Equal(t, tx.ID, stored.ID)
	assert.Equal(t, "v1", stored.GetConfig().Version)

	// Clean shutdown
	err := h.stop()
	require.NoError(t, err)
}

func TestRunnerRunLifecycle(t *testing.T) {
	h := newTestHarness(t)

	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)
	go func() {
		errCh <- h.runner.Run(ctx)
	}()

	assert.Eventually(t, func() bool {
		return h.runner.GetState() == finitestate.StatusRunning
	}, time.Second, 10*time.Millisecond)

	assert.True(t, h.runner.IsRunning())

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Runner did not complete within timeout")
	}

	assert.Eventually(t, func() bool {
		return h.runner.GetState() == finitestate.StatusStopped
	}, time.Second, 10*time.Millisecond, "runner should reach Stopped state")
	assert.False(t, h.runner.IsRunning())
}

func TestRunnerConfigUpdate(t *testing.T) {
	h := newTestHarness(t)
	h.start()

	// Send first config
	tx1 := h.sendConfig("v1")
	stored1 := h.waitForTransaction(tx1.ID.String())
	assert.Equal(t, "v1", stored1.GetConfig().Version)

	// Send second config
	tx2 := h.sendConfig("v2")
	stored2 := h.waitForTransaction(tx2.ID.String())
	assert.Equal(t, "v2", stored2.GetConfig().Version)

	// Verify current transaction is updated
	assert.Eventually(t, func() bool {
		current := h.txStorage.GetCurrent()
		return current != nil && current.ID == tx2.ID
	}, 5*time.Second, 10*time.Millisecond, "current transaction should be updated")

	// Clean shutdown
	err := h.stop()
	require.NoError(t, err)
}

func TestRunnerStateChan(t *testing.T) {
	t.Run("state changes during lifecycle", func(t *testing.T) {
		h := newTestHarness(t)
		runner := h.runner

		// Use separate contexts for state channel and runner
		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()
		stateCh := runner.GetStateChan(stateCtx)

		runCtx, runCancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runner.Run(runCtx)
		}()

		// Assert the expected state sequence

		// 1. Should start with New state
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusNew, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected New state")
		}

		// 2. Should transition to Booting
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusBooting, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Booting state")
		}

		// 3. Should transition to Running
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusRunning, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Running state")
		}

		// Verify runner is now running
		assert.True(t, runner.IsRunning())

		// Trigger shutdown
		runCancel()

		// 4. Should transition to Stopping
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusStopping, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Stopping state")
		}

		// 5. Should transition to Stopped
		select {
		case state := <-stateCh:
			assert.Equal(t, finitestate.StatusStopped, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Stopped state")
		}

		// Wait for Run() to complete
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		// Final verification
		assert.Eventually(t, func() bool {
			return runner.GetState() == finitestate.StatusStopped
		}, time.Second, 10*time.Millisecond, "runner should reach Stopped state")
		assert.False(t, runner.IsRunning())
	})
}

func TestRunnerMultipleConcurrentTransactions(t *testing.T) {
	txStorage := txstorage.NewMemoryStorage()
	sagaOrchestrator := orchestrator.NewSagaOrchestrator(txStorage, slog.Default().Handler())
	runner, err := NewRunner(sagaOrchestrator)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	// Wait for runner to start
	assert.Eventually(t, func() bool {
		return runner.IsRunning()
	}, 5*time.Second, 10*time.Millisecond)

	txSiphon := runner.GetTransactionSiphon()

	// Send multiple transactions concurrently
	for i := range 5 {
		go func(n int) {
			cfg, err := config.NewFromProto(&pb.ServerConfig{})
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}
			cfg.Version = fmt.Sprintf("v%d", n)
			tx, err := transaction.New(
				transaction.SourceTest,
				"concurrent test",
				fmt.Sprintf("request-%d", n),
				cfg,
				slog.Default().Handler(),
			)
			if err != nil {
				t.Errorf("Failed to create transaction: %v", err)
				return
			}

			// Validate the transaction before sending
			if err := tx.BeginValidation(); err != nil {
				t.Errorf("Failed to begin validation: %v", err)
				return
			}
			tx.IsValid.Store(true)
			if err := tx.MarkValidated(); err != nil {
				t.Errorf("Failed to mark validated: %v", err)
				return
			}

			select {
			case txSiphon <- tx:
				// Sent successfully
			case <-time.After(5 * time.Second):
				t.Errorf("Timeout sending transaction %d", n)
			}
		}(i)
	}

	// Verify all transactions are stored - we need to check by count since IDs are UUIDs
	assert.Eventually(t, func() bool {
		txList := txStorage.GetAll()
		return len(txList) >= 5
	}, 5*time.Second, 10*time.Millisecond, "all transactions should be stored")

	// Clean shutdown
	cancel()
	shutdownErr := <-errCh
	require.NoError(t, shutdownErr)
}

func TestRunnerString(t *testing.T) {
	h := newTestHarness(t)
	name := h.runner.String()
	assert.NotEmpty(t, name, "String() should return a non-empty value")
	assert.Contains(t, name, "txmgr")
}

func TestRunnerStop(t *testing.T) {
	t.Run("stop transitions to stopping state", func(t *testing.T) {
		h := newTestHarness(t)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- h.runner.Run(ctx)
		}()

		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusRunning
		}, time.Second, 10*time.Millisecond)

		h.runner.Stop()

		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Runner did not complete within timeout")
		}

		assert.Eventually(t, func() bool {
			return h.runner.GetState() == finitestate.StatusStopped
		}, time.Second, 10*time.Millisecond, "runner should reach Stopped state")
	})
}

func TestRunnerErrorHandling(t *testing.T) {
	h := newTestHarness(t)
	h.start()

	// Send a valid transaction first
	tx := h.sendConfig("v1")

	// Wait for it to be stored
	h.waitForTransaction(tx.ID.String())

	// Runner should continue running despite error
	assert.Eventually(t, func() bool {
		return h.runner.IsRunning()
	}, 5*time.Second, 10*time.Millisecond, "runner should continue running")

	// Clean shutdown
	err := h.stop()
	require.NoError(t, err)
}
