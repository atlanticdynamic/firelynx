package txmgr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSagaParticipant is a mock implementation of the SagaParticipant interface
type MockSagaParticipant struct {
	mock.Mock
	name string
}

func NewMockSagaParticipant(name string) *MockSagaParticipant {
	return &MockSagaParticipant{
		name: name,
	}
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

func NewConflictingParticipant(name string) *ConflictingParticipant {
	return &ConflictingParticipant{name: name}
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

// MockReloadParticipant is a mock implementation of SagaParticipant that can be configured to fail reload
type MockReloadParticipant struct {
	mock.Mock
	name    string
	running bool
	state   string
}

// NewMockReloadParticipant creates a new MockReloadParticipant with the given name
func NewMockReloadParticipant(name string) *MockReloadParticipant {
	return &MockReloadParticipant{
		name:    name,
		running: true,
		state:   "running",
	}
}

// String returns the name of the participant
func (m *MockReloadParticipant) String() string {
	return m.name
}

// Run implements supervisor.Runnable
func (m *MockReloadParticipant) Run(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Stop implements supervisor.Runnable
func (m *MockReloadParticipant) Stop() {
	m.Called()
}

// ApplyPendingConfig implements SagaParticipant - replaces Reload()
func (m *MockReloadParticipant) ApplyPendingConfig(ctx context.Context) error {
	args := m.Called(ctx)

	// Simulate a reload by briefly setting running to false
	if m.running {
		m.running = false
		m.state = "reloading"

		// Wait a short time to simulate reload work
		time.Sleep(10 * time.Millisecond)

		// Set back to running if not configured to fail
		if m.state != "failed" {
			m.running = true
			m.state = "running"
			return nil
		}
		return fmt.Errorf("failed to apply pending config")
	}

	return args.Error(0)
}

// IsRunning implements supervisor.Readiness
func (m *MockReloadParticipant) IsRunning() bool {
	m.Called()
	return m.running
}

// GetState implements supervisor.Stateable
func (m *MockReloadParticipant) GetState() string {
	m.Called()
	return m.state
}

// GetStateChan implements supervisor.Stateable
func (m *MockReloadParticipant) GetStateChan(ctx context.Context) <-chan string {
	m.Called(ctx)
	ch := make(chan string, 1)
	ch <- m.state
	return ch
}

// ExecuteConfig implements SagaParticipant
func (m *MockReloadParticipant) ExecuteConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// CompensateConfig implements SagaParticipant
func (m *MockReloadParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// SetFailReload configures the participant to fail during reload
func (m *MockReloadParticipant) SetFailReload() {
	m.running = false
	m.state = "failed"
}

// TestSagaParticipantInterface_ReloadConflict tests that registering a participant that implements both
// SagaParticipant and Reloadable interfaces is detected and rejected
func TestSagaParticipantInterface_ReloadConflict(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()
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

// TestSagaParticipantInterface_ApplyPendingConfig tests the ApplyPendingConfig method
func TestSagaParticipantInterface_ApplyPendingConfig(t *testing.T) {
	ctx := context.Background()
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()
	orchestrator := NewSagaOrchestrator(storage, handler)

	// Create participant
	participant := NewMockSagaParticipant("test-participant")

	// Setup expectations
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

func TestTriggerReload_Success(t *testing.T) {
	// Create a transaction storage with a current transaction
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()

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
	participant1.On("ApplyPendingConfig", mock.Anything).Return(nil)
	participant2.On("ApplyPendingConfig", mock.Anything).Return(nil)
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
	err = orchestrator.TriggerReload(context.Background())
	assert.NoError(t, err)

	// Verify transaction state
	assert.Equal(t, finitestate.StateCompleted, tx.GetState())

	// Verify mocks
	participant1.AssertExpectations(t)
	participant2.AssertExpectations(t)
}

func TestTriggerReload_Failure(t *testing.T) {
	// Create a transaction storage with a current transaction
	handler := slog.NewTextHandler(os.Stdout, nil)
	storage := txstorage.NewTransactionStorage()

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
	participant1.On("ApplyPendingConfig", mock.Anything).Return(nil)
	participant2.On("ApplyPendingConfig", mock.Anything).
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
	err = orchestrator.TriggerReload(context.Background())
	assert.Error(t, err)

	// Verify transaction state is now error
	assert.Equal(t, finitestate.StateError, tx.GetState())

	// Verify mocks
	participant1.AssertExpectations(t)
	participant2.AssertExpectations(t)
}
