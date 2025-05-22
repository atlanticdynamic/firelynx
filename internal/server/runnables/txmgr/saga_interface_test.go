package txmgr_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSagaParticipant is a mock implementation of the SagaParticipant interface
type MockSagaParticipant struct {
	mock.Mock
	name string
}

func (m *MockSagaParticipant) String() string {
	return m.name
}

func (m *MockSagaParticipant) Run(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSagaParticipant) Stop() {
	m.Called()
}

func (m *MockSagaParticipant) GetState() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSagaParticipant) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSagaParticipant) GetStateChan(ctx context.Context) <-chan string {
	args := m.Called(ctx)
	return args.Get(0).(<-chan string)
}

func (m *MockSagaParticipant) ExecuteConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockSagaParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockSagaParticipant) ApplyPendingConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ConflictingParticipant implements both SagaParticipant and Reloadable (conflict)
type ConflictingParticipant struct {
	name string
}

func (p *ConflictingParticipant) String() string                { return p.name }
func (p *ConflictingParticipant) Run(ctx context.Context) error { return nil }
func (p *ConflictingParticipant) Stop()                         {}
func (p *ConflictingParticipant) GetState() string              { return "running" }
func (p *ConflictingParticipant) IsRunning() bool               { return true }
func (p *ConflictingParticipant) GetStateChan(ctx context.Context) <-chan string {
	ch := make(chan string, 1)
	ch <- "running"
	close(ch)
	return ch
}

func (p *ConflictingParticipant) ExecuteConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return nil
}

func (p *ConflictingParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return nil
}
func (p *ConflictingParticipant) ApplyPendingConfig(ctx context.Context) error { return nil }

func (p *ConflictingParticipant) Reload() {} // This causes the conflict

// TestSagaParticipantInterface_ReloadConflict tests that registering a participant
// that implements both SagaParticipant and Reloadable interfaces is detected and rejected
func TestSagaParticipantInterface_ReloadConflict(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()
	orchestrator := txmgr.NewSagaOrchestrator(storage, handler)

	// Create a participant that also implements Reloadable (conflict)
	conflictParticipant := &ConflictingParticipant{name: "conflict"}

	// Registration should return an error
	err := orchestrator.RegisterParticipant(conflictParticipant)
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"implements supervisor.Reloadable which conflicts with SagaParticipant",
	)
}

// TestSagaParticipantInterface_ApplyPendingConfig tests the new ApplyPendingConfig method
func TestSagaParticipantInterface_ApplyPendingConfig(t *testing.T) {
	ctx := context.Background()
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()
	orchestrator := txmgr.NewSagaOrchestrator(storage, handler)

	// Create participant
	participant := &MockSagaParticipant{name: "test-participant"}

	// Setup expectations - we need to be more specific about what will be called
	participant.On("ApplyPendingConfig", mock.Anything).Return(nil)
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

	// Verify ApplyPendingConfig was called on participant
	participant.AssertCalled(t, "ApplyPendingConfig", mock.Anything)
}
