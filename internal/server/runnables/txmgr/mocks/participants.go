package mocks

import (
	"context"
	"fmt"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/stretchr/testify/mock"
)

// MockSagaParticipant is a mock implementation of the SagaParticipant interface
type MockSagaParticipant struct {
	mock.Mock
	name string
}

// NewMockSagaParticipant creates a new MockSagaParticipant with the given name
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

// NewConflictingParticipant creates a new ConflictingParticipant with the given name
func NewConflictingParticipant(name string) *ConflictingParticipant {
	return &ConflictingParticipant{
		name: name,
	}
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
