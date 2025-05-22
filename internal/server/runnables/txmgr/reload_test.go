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
