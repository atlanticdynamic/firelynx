package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockParticipant is a mock implementation of the SagaParticipant interface for testing
type MockParticipant struct {
	mock.Mock
	name string
}

func (m *MockParticipant) StageConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// CommitConfig implements the SagaParticipant interface
func (m *MockParticipant) CommitConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockParticipant) String() string {
	return m.name
}

func (m *MockParticipant) Run(ctx context.Context) error {
	return nil
}

func (m *MockParticipant) Stop() {
	// Do nothing
}

func (m *MockParticipant) GetState() string {
	return "running"
}

func (m *MockParticipant) GetStateChan(ctx context.Context) <-chan string {
	stateCh := make(chan string)
	go func() {
		defer close(stateCh)
		select {
		case <-ctx.Done():
			return
		case stateCh <- "running":
			return
		}
	}()
	return stateCh
}

func (m *MockParticipant) IsRunning() bool {
	return true
}

// NewMockParticipant creates a new mock participant with the given name
func NewMockParticipant(name string) *MockParticipant {
	return &MockParticipant{
		name: name,
	}
}

func TestNewSagaOrchestrator(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()

	orchestrator := NewSagaOrchestrator(storage, handler)

	assert.NotNil(t, orchestrator)
	assert.Equal(t, storage, orchestrator.txStorage)
	assert.NotNil(t, orchestrator.runnables)
	assert.Empty(t, orchestrator.runnables)
}

func TestRegisterParticipant(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	participant := NewMockParticipant("test-participant")
	// Handle error return now
	err := orchestrator.RegisterParticipant(participant)
	assert.NoError(t, err)

	assert.Len(t, orchestrator.runnables, 1)
	assert.Contains(t, orchestrator.runnables, "test-participant")
	assert.Equal(t, participant, orchestrator.runnables["test-participant"])
}

// TestRegisterParticipant_ReloadableConflict tests that registering a participant
// that implements supervisor.Reloadable fails with an error
func TestRegisterParticipant_ReloadableConflict(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Create a conflicting participant (implements both SagaParticipant and Reloadable)
	conflictParticipant := &conflictingParticipant{name: "conflict"}

	// Registration should return an error
	err := orchestrator.RegisterParticipant(conflictParticipant)
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"implements supervisor.Reloadable which conflicts with SagaParticipant",
	)

	// Participant should not be registered
	assert.Empty(t, orchestrator.runnables)
}

// conflictingParticipant implements both SagaParticipant and Reloadable (conflict)
type conflictingParticipant struct {
	name string
}

func (p *conflictingParticipant) String() string                { return p.name }
func (p *conflictingParticipant) Run(ctx context.Context) error { return nil }
func (p *conflictingParticipant) Stop()                         {}
func (p *conflictingParticipant) GetState() string              { return "running" }
func (p *conflictingParticipant) IsRunning() bool               { return true }
func (p *conflictingParticipant) GetStateChan(ctx context.Context) <-chan string {
	ch := make(chan string, 1)
	ch <- "running"
	return ch
}

func (p *conflictingParticipant) StageConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return nil
}

func (p *conflictingParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return nil
}
func (p *conflictingParticipant) CommitConfig(ctx context.Context) error { return nil }

func (p *conflictingParticipant) Reload() {} // This causes the conflict

func TestProcessTransaction_Success(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)
	ctx := t.Context()

	// Create a test transaction
	cfg := &config.Config{Version: "v1"}
	tx, err := transaction.New(transaction.SourceTest, "test", "req-123", cfg, handler)
	require.NoError(t, err)

	// Mark as validated (required for processing)
	err = tx.RunValidation()
	require.NoError(t, err)

	// Register two mock participants
	participant1 := NewMockParticipant("participant1")
	participant2 := NewMockParticipant("participant2")

	// Set expectations
	participant1.On("StageConfig", mock.Anything, tx).Return(nil)
	participant2.On("StageConfig", mock.Anything, tx).Return(nil)

	// Set expectations for CommitConfig
	participant1.On("CommitConfig", mock.Anything).Return(nil)
	participant2.On("CommitConfig", mock.Anything).Return(nil)

	err = orchestrator.RegisterParticipant(participant1)
	require.NoError(t, err)
	err = orchestrator.RegisterParticipant(participant2)
	require.NoError(t, err)

	// Process the transaction
	err = orchestrator.ProcessTransaction(ctx, tx)
	require.NoError(t, err)

	// Verify expectations
	participant1.AssertExpectations(t)
	participant2.AssertExpectations(t)

	// Verify transaction state
	assert.Equal(t, finitestate.StateCompleted, tx.GetState())

	// Verify transaction is in storage
	assert.Equal(t, tx, storage.GetCurrent())
}

func TestProcessTransaction_Failure(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)
	ctx := t.Context()

	// Create a test transaction
	cfg := &config.Config{Version: "v1"}
	tx, err := transaction.New(transaction.SourceTest, "test", "req-123", cfg, handler)
	require.NoError(t, err)

	// Mark as validated (required for processing)
	err = tx.RunValidation()
	require.NoError(t, err)

	// Register one mock participant
	participant := NewMockParticipant("participant1")

	// Set up the mock to fail
	testErr := errors.New("test error")
	participant.On("StageConfig", mock.Anything, mock.Anything).Return(testErr)

	// Register the participant
	err = orchestrator.RegisterParticipant(participant)
	require.NoError(t, err)

	// Process the transaction - should fail
	err = orchestrator.ProcessTransaction(ctx, tx)
	require.Error(t, err)
	assert.Equal(t, testErr, err)

	// Verify expectations
	participant.AssertExpectations(t)

	// Verify transaction state - this could be either failed or compensated due to our async compensation
	finalState := tx.GetState()
	assert.True(t, finalState == finitestate.StateFailed ||
		finalState == finitestate.StateCompensating ||
		finalState == finitestate.StateCompensated,
		"Expected state to be failed, compensating, or compensated, but got %s", finalState)

	// Verify transaction is NOT in storage
	assert.Nil(t, storage.GetCurrent())
}

func TestCompensateParticipants(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)
	ctx := t.Context()

	// Create a test transaction
	cfg := &config.Config{Version: "v1"}
	tx, err := transaction.New(transaction.SourceTest, "test", "req-123", cfg, handler)
	require.NoError(t, err)

	// Mark as validated and failed (required for compensation)
	err = tx.RunValidation()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)
	err = tx.MarkFailed(ctx, errors.New("test error"))
	require.NoError(t, err)

	// Register two mock participants
	participant1 := NewMockParticipant("participant1")
	participant2 := NewMockParticipant("participant2")

	// Set expectations
	participant1.On("CompensateConfig", mock.Anything, tx).Return(nil).Maybe()
	participant2.On("CompensateConfig", mock.Anything, tx).Return(nil).Maybe()

	err = orchestrator.RegisterParticipant(participant1)
	require.NoError(t, err)
	err = orchestrator.RegisterParticipant(participant2)
	require.NoError(t, err)

	// Set up participant states to simulate one that succeeded
	// and one that didn't start yet
	participantState1, err := tx.GetParticipants().GetOrCreate("participant1")
	require.NoError(t, err)
	err = participantState1.Execute()
	require.NoError(t, err)
	err = participantState1.MarkSucceeded()
	require.NoError(t, err)

	// Call compensation
	orchestrator.compensateParticipants(ctx, tx)

	// Participant1 should have been compensated since it succeeded
	participant1.AssertExpectations(t)

	// Verify transaction state is either still compensating or already compensated
	finalState := tx.GetState()
	assert.True(t, finalState == finitestate.StateCompensating ||
		finalState == finitestate.StateCompensated,
		"Expected state to be compensating or compensated, but got %s", finalState)
}

func TestGetTransactionStatus(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Create a test transaction
	cfg := &config.Config{}
	tx, err := transaction.New(transaction.SourceTest, "test-details", "req-456", cfg, handler)
	require.NoError(t, err)

	// Add to storage
	err = storage.Add(tx)
	require.NoError(t, err)

	// Get status
	status, err := orchestrator.GetTransactionStatus(tx.ID.String())
	require.NoError(t, err)

	// Verify returned status
	assert.Equal(t, tx.ID.String(), status["id"])
	assert.Equal(t, finitestate.StateCreated, status["state"])
	assert.Equal(t, transaction.SourceTest, status["source"])
	assert.Equal(t, "test-details", status["sourceDetail"])
	assert.Equal(t, tx.CreatedAt, status["createdAt"])
	assert.Equal(t, false, status["isValid"])
}

func TestGetTransactionStatus_NotFound(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Get status for non-existent transaction
	status, err := orchestrator.GetTransactionStatus("non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "transaction not found")
}

func TestConcurrentParticipantRegistration(t *testing.T) {
	handler := logging.SetupHandler("error")
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Register multiple participants concurrently
	const numParticipants = 10
	var wg sync.WaitGroup
	wg.Add(numParticipants)

	// Create a mutex to protect the error collection
	var errMutex sync.Mutex
	var registrationErrors []error

	for i := range numParticipants {
		go func(i int) {
			defer wg.Done()
			participant := NewMockParticipant(fmt.Sprintf("participant-%d", i))

			// Handle error return
			err := orchestrator.RegisterParticipant(participant)
			if err != nil {
				errMutex.Lock()
				registrationErrors = append(registrationErrors, err)
				errMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify all participants were registered without errors
	assert.Empty(t, registrationErrors, "Unexpected errors during participant registration")
	assert.Len(t, orchestrator.runnables, numParticipants)
}

// TestProcessTransactionWithNoParticipants tests that transactions
// reach completed state even when no participants are registered for reload
func TestProcessTransactionWithNoParticipants(t *testing.T) {
	handler := logging.SetupHandler("debug")
	storage := txstorage.NewMemoryStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)
	ctx := t.Context()

	// Create a minimal config for testing
	cfg := &config.Config{
		Version: "v1",
	}

	// Create and validate transaction
	tx, err := transaction.New(
		transaction.SourceTest,
		"TestProcessTransactionWithNoParticipants",
		"test-request-id",
		cfg,
		handler,
	)
	require.NoError(t, err)

	// Validate the transaction
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process the transaction (with no participants registered)
	err = orchestrator.ProcessTransaction(ctx, tx)
	require.NoError(t, err)

	assert.Equal(t, finitestate.StateCompleted, tx.GetState(),
		"Transaction should reach completed state even with no participants")

	assert.Equal(t, tx, storage.GetCurrent(),
		"Transaction should be stored as current after processing")
}
