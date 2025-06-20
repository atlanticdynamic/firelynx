package orchestrator

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestSagaParticipantInterface_ReloadConflictFromMocks tests that registering a participant that implements both
// SagaParticipant and Reloadable interfaces is detected and rejected using mocks from the mocks package
func TestSagaParticipantInterface_ReloadConflictFromMocks(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Create a participant that also implements Reloadable (conflict)
	conflictParticipant := NewConflictingParticipant("conflict")

	// Registration should return an error
	err := orchestrator.RegisterParticipant(conflictParticipant)
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"implements supervisor.Reloadable which conflicts with SagaParticipant",
	)
}

// TestSagaParticipantInterface_ApplyPendingConfigFromMocks tests the CommitConfig method
func TestSagaParticipantInterface_ApplyPendingConfigFromMocks(t *testing.T) {
	ctx := t.Context()
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Create participant
	participant := newMockSagaParticipant("test-participant")

	// Setup expectations
	participant.On("CommitConfig", mock.Anything).Return(nil)
	participant.On("GetState").Return("running")
	participant.On("IsRunning").Return(true)

	// Register participant
	err := orchestrator.RegisterParticipant(participant)
	assert.NoError(t, err)

	// Create and prepare transaction with a non-nil config
	testConfig := &config.Config{
		Version: "test",
	}
	tx, err := transaction.New(transaction.SourceTest, "test", "test-id", testConfig, handler)
	assert.NoError(t, err)

	assert.NoError(t, tx.BeginValidation())
	tx.IsValid.Store(true)
	assert.NoError(t, tx.MarkValidated())
	assert.NoError(t, tx.BeginExecution())
	assert.NoError(t, tx.MarkSucceeded())

	// Store transaction
	storage.SetCurrent(tx)

	// Call TriggerReload
	err = orchestrator.TriggerReload(ctx)
	assert.NoError(t, err)

	// Verify transaction is completed
	assert.Equal(t, finitestate.StateCompleted, tx.GetState())

	// Verify CommitConfig was called on participant
	participant.AssertCalled(t, "CommitConfig", mock.Anything)
}

func TestTriggerReload_SuccessFromMocks(t *testing.T) {
	// Create a transaction storage with a current transaction
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()

	// Create a test config
	testConfig := &config.Config{
		Version: "test",
	}

	tx, err := transaction.New(transaction.SourceTest, "test", "test-request", testConfig, handler)
	assert.NoError(t, err)

	// Mark transaction as succeeded (prerequisite for reload)
	assert.NoError(t, tx.BeginValidation())
	tx.IsValid.Store(true)
	assert.NoError(t, tx.MarkValidated())
	assert.NoError(t, tx.BeginExecution())
	assert.NoError(t, tx.MarkSucceeded())

	// Store transaction
	storage.SetCurrent(tx)

	// Create the orchestrator
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Register mock participants
	participant1 := NewMockReloadParticipant("participant1")
	participant2 := NewMockReloadParticipant("participant2")

	// Set up mocks
	participant1.On("CommitConfig", mock.Anything).Return(nil)
	participant2.On("CommitConfig", mock.Anything).Return(nil)
	participant1.On("GetState").Return("running")
	participant2.On("GetState").Return("running")
	participant1.On("IsRunning").Return(true)
	participant2.On("IsRunning").Return(true)

	// Register participants
	err = orchestrator.RegisterParticipant(participant1)
	assert.NoError(t, err)
	err = orchestrator.RegisterParticipant(participant2)
	assert.NoError(t, err)

	// Trigger reload
	err = orchestrator.TriggerReload(t.Context())
	assert.NoError(t, err)

	// Verify transaction state
	assert.Equal(t, finitestate.StateCompleted, tx.GetState())

	// Verify mocks
	participant1.AssertExpectations(t)
	participant2.AssertExpectations(t)
}

func TestTriggerReload_FailureFromMocks(t *testing.T) {
	// Create a transaction storage with a current transaction
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()

	// Create a test config
	testConfig := &config.Config{
		Version: "test",
	}

	tx, err := transaction.New(transaction.SourceTest, "test", "test-request", testConfig, handler)
	assert.NoError(t, err)

	// Mark transaction as succeeded (prerequisite for reload)
	assert.NoError(t, tx.BeginValidation())
	tx.IsValid.Store(true)
	assert.NoError(t, tx.MarkValidated())
	assert.NoError(t, tx.BeginExecution())
	assert.NoError(t, tx.MarkSucceeded())

	// Store transaction
	storage.SetCurrent(tx)

	// Create the orchestrator
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Register mock participants
	participant1 := NewMockReloadParticipant("participant1")
	participant2 := NewMockReloadParticipant("participant2")

	// Configure participant2 to fail
	participant2.SetFailReload()

	// Set up mocks - the expectations need to match what TriggerReload actually calls
	participant1.On("CommitConfig", mock.Anything).Return(nil)
	participant2.On("CommitConfig", mock.Anything).
		Return(fmt.Errorf("failed to apply pending config"))
	participant1.On("GetState").Return("running")
	participant2.On("GetState").Maybe().Return("failed")
	participant1.On("IsRunning").Maybe().Return(true)
	participant2.On("IsRunning").Maybe().Return(false)

	// Register participants
	err = orchestrator.RegisterParticipant(participant1)
	assert.NoError(t, err)
	err = orchestrator.RegisterParticipant(participant2)
	assert.NoError(t, err)

	// Trigger reload and verify it returns an error
	err = orchestrator.TriggerReload(t.Context())
	assert.Error(t, err)

	// Verify transaction state is now error
	assert.Equal(t, finitestate.StateError, tx.GetState())

	// Verify mocks
	participant1.AssertExpectations(t)
	participant2.AssertExpectations(t)
}
