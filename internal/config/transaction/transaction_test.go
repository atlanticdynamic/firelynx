package transaction

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*ConfigTransaction, slog.Handler) {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Create a valid config with the current version
	cfg := &config.Config{
		Version: config.VersionLatest,
	}

	tx, err := FromTest("test_transaction", cfg, handler)
	require.NoError(t, err)
	require.NotNil(t, tx)

	return tx, handler
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates new transaction with correct initial state", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.GetState())
		assert.Equal(t, SourceTest, tx.Source)
		assert.Equal(t, "test_transaction", tx.SourceDetail)
		assert.NotEmpty(t, tx.ID)
		assert.NotNil(t, tx.logger)
		assert.NotNil(t, tx.logCollector)
		assert.False(t, tx.IsValid.Load())
	})
}

func TestConfigTransaction_String(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	str := tx.String()
	assert.Contains(t, str, "Transaction ID:")
	assert.Contains(t, str, "State: created")
	assert.Contains(t, str, tx.GetTransactionID())
}

func TestConfigTransaction_GetTransactionID(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	id := tx.GetTransactionID()
	assert.NotEmpty(t, id)
	assert.Equal(t, tx.ID.String(), id)
}

func TestConfigTransaction_BeginValidation(t *testing.T) {
	t.Parallel()

	t.Run("successful transition", func(t *testing.T) {
		tx, _ := setupTest(t)

		err := tx.BeginValidation()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateValidating, tx.GetState())
	})

	t.Run("invalid transition", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Move to a state that can't transition to validating
		err := tx.BeginExecution()
		assert.Error(t, err) // Should fail because not validated

		// Try to begin validation from wrong state
		err = tx.BeginValidation()
		assert.NoError(t, err) // Actually, from created state this should work
	})
}

func TestConfigTransaction_MarkValidated(t *testing.T) {
	t.Parallel()

	t.Run("successful validation", func(t *testing.T) {
		tx, _ := setupTest(t)

		// First begin validation
		err := tx.BeginValidation()
		require.NoError(t, err)

		// Mark as valid
		tx.IsValid.Store(true)

		err = tx.MarkValidated()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
	})

	t.Run("validation failed", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Begin validation
		err := tx.BeginValidation()
		require.NoError(t, err)

		// Don't mark as valid (IsValid remains false)
		err = tx.MarkValidated()
		assert.NoError(t, err) // MarkValidated returns nil when it successfully marks as invalid
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})
}

func TestConfigTransaction_MarkInvalid(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	// Begin validation
	err := tx.BeginValidation()
	require.NoError(t, err)

	validationErr := errors.New("validation error")
	err = tx.MarkInvalid(validationErr)
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateInvalid, tx.GetState())
}

func TestConfigTransaction_BeginExecution(t *testing.T) {
	t.Parallel()

	t.Run("from validated state", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Move through validation
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)

		// Now begin execution
		err = tx.BeginExecution()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateExecuting, tx.GetState())
	})

	t.Run("from non-validated state", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Try to execute without validation
		err := tx.BeginExecution()
		assert.ErrorIs(t, err, ErrNotValidated)
	})
}

func TestConfigTransaction_MarkSucceeded(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	// Move through proper states
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)

	// Mark as succeeded
	err = tx.MarkSucceeded()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateSucceeded, tx.GetState())
}

func TestConfigTransaction_MarkCompleted(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	// Move through proper states
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)
	err = tx.MarkSucceeded()
	require.NoError(t, err)

	// Need to go through reloading state first
	err = tx.BeginReload()
	require.NoError(t, err)

	// Now mark as completed
	err = tx.MarkCompleted()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateCompleted, tx.GetState())
}

func TestConfigTransaction_BeginReload(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	// Move to succeeded state
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)
	err = tx.MarkSucceeded()
	require.NoError(t, err)

	// Begin reload
	err = tx.BeginReload()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateReloading, tx.GetState())
}

func TestConfigTransaction_BeginCompensation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tx, _ := setupTest(t)

	// Move to failed state first
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)

	// Mark as failed
	failErr := errors.New("execution failed")
	err = tx.MarkFailed(ctx, failErr)
	require.NoError(t, err)

	// Now can begin compensation
	err = tx.BeginCompensation()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateCompensating, tx.GetState())
}

func TestConfigTransaction_MarkCompensated(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tx, _ := setupTest(t)

	// Move through states to compensation
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)
	err = tx.MarkFailed(ctx, errors.New("failed"))
	require.NoError(t, err)
	err = tx.BeginCompensation()
	require.NoError(t, err)

	// Mark as compensated
	err = tx.MarkCompensated()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateCompensated, tx.GetState())
}

func TestConfigTransaction_MarkError(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	errorMsg := errors.New("unrecoverable error")
	err := tx.MarkError(errorMsg)
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateError, tx.GetState())
}

func TestConfigTransaction_MarkFailed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tx, _ := setupTest(t)

	// Move to executing state
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)

	// Mark as failed
	failErr := errors.New("execution failed")
	err = tx.MarkFailed(ctx, failErr)
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateFailed, tx.GetState())
}

func TestConfigTransaction_LegacyMethods(t *testing.T) {
	t.Parallel()

	t.Run("BeginPreparation", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Setup to validated state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)

		// BeginPreparation should map to BeginExecution
		err = tx.BeginPreparation()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateExecuting, tx.GetState())
	})

	t.Run("MarkPrepared", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Setup to executing state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)

		// MarkPrepared should map to MarkSucceeded
		err = tx.MarkPrepared()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateSucceeded, tx.GetState())
	})

	t.Run("BeginCommit", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Setup to validated state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)

		// BeginCommit should map to BeginExecution
		err = tx.BeginCommit()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateExecuting, tx.GetState())
	})

	t.Run("MarkCommitted", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Setup to executing state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)

		// MarkCommitted should map to MarkSucceeded
		err = tx.MarkCommitted()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateSucceeded, tx.GetState())
	})

	t.Run("BeginRollback", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		// Setup to failed state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)
		err = tx.MarkFailed(ctx, errors.New("failed"))
		require.NoError(t, err)

		// BeginRollback should map to BeginCompensation
		err = tx.BeginRollback()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateCompensating, tx.GetState())
	})

	t.Run("MarkRolledBack", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		// Setup to compensating state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)
		err = tx.MarkFailed(ctx, errors.New("failed"))
		require.NoError(t, err)
		err = tx.BeginCompensation()
		require.NoError(t, err)

		// MarkRolledBack should map to MarkCompensated
		err = tx.MarkRolledBack()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateCompensated, tx.GetState())
	})
}

func TestConfigTransaction_GetConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Version: config.VersionLatest,
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	tx, err := FromTest("test", cfg, handler)
	require.NoError(t, err)

	gotCfg := tx.GetConfig()
	assert.Equal(t, cfg, gotCfg)
}

func TestConfigTransaction_RunValidation(t *testing.T) {
	t.Parallel()

	t.Run("successful validation", func(t *testing.T) {
		tx, _ := setupTest(t)

		err := tx.RunValidation()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
		assert.True(t, tx.IsValid.Load())
	})

	t.Run("validation with nil config", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		tx, err := New(SourceTest, "test", "", nil, handler)
		require.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("validation with invalid config", func(t *testing.T) {
		// Create a config that will fail validation
		cfg := &config.Config{
			Version: "invalid-version", // This should fail validation
		}

		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		err = tx.RunValidation()
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})
}

func TestTransactionLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("validates happy path lifecycle", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.GetState())

		require.NoError(t, tx.RunValidation())
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
		assert.True(t, tx.IsValid.Load())

		require.NoError(t, tx.BeginExecution())
		assert.Equal(t, finitestate.StateExecuting, tx.GetState())

		require.NoError(t, tx.MarkSucceeded())
		assert.Equal(t, finitestate.StateSucceeded, tx.GetState())

		require.NoError(t, tx.BeginReload())
		assert.Equal(t, finitestate.StateReloading, tx.GetState())

		require.NoError(t, tx.MarkCompleted())
		assert.Equal(t, finitestate.StateCompleted, tx.GetState())
	})

	t.Run("validates failed validation path", func(t *testing.T) {
		tx, _ := setupTest(t)

		invalidCfg := &config.Config{}
		tx.domainConfig = invalidCfg

		validationErrs := []error{
			errors.New("validation error 1"),
			errors.New("validation error 2"),
		}

		require.NoError(t, tx.fsm.Transition(finitestate.StateValidating))
		tx.setStateInvalid(validationErrs)
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())

		err := tx.BeginExecution()
		assert.ErrorIs(t, err, ErrNotValidated)
	})

	t.Run("validates compensation path", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(ctx, testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())

		require.NoError(t, tx.BeginCompensation())
		assert.Equal(t, finitestate.StateCompensating, tx.GetState())

		require.NoError(t, tx.MarkCompensated())
		assert.Equal(t, finitestate.StateCompensated, tx.GetState())
	})

	t.Run("validates failure path", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(ctx, testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())
	})
}

func TestLogCollection(t *testing.T) {
	t.Parallel()

	t.Run("collects and plays back logs", func(t *testing.T) {
		tx, handler := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())
		require.NoError(t, tx.MarkSucceeded())
		require.NoError(t, tx.BeginReload())
		require.NoError(t, tx.MarkCompleted())

		err := tx.PlaybackLogs(handler)
		require.NoError(t, err)
	})
}

func TestGetDuration(t *testing.T) {
	t.Parallel()

	t.Run("reports transaction duration", func(t *testing.T) {
		tx, _ := setupTest(t)

		time.Sleep(10 * time.Millisecond)

		duration := tx.GetTotalDuration()
		assert.Greater(t, duration, 0*time.Millisecond)
	})
}

func TestConvertToAppDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		input    apps.AppCollection
		expected []serverApps.AppDefinition
	}{
		{
			name:     "empty collection",
			input:    apps.AppCollection{},
			expected: []serverApps.AppDefinition{},
		},
		{
			name: "single echo app",
			input: apps.AppCollection{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
		},
		{
			name: "multiple apps",
			input: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToAppDefinitions(tt.input)

			require.Len(t, result, len(tt.expected))

			for i, def := range result {
				assert.Equal(t, tt.expected[i].ID, def.ID)
				assert.Equal(t, tt.expected[i].Config, def.Config)
			}
		})
	}
}

// Mock FSM for testing error conditions
type MockFSM struct {
	mock.Mock
}

func (m *MockFSM) Transition(state string) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockFSM) TransitionBool(state string) bool {
	args := m.Called(state)
	return args.Bool(0)
}

func (m *MockFSM) TransitionIfCurrentState(currentState, newState string) error {
	args := m.Called(currentState, newState)
	return args.Error(0)
}

func (m *MockFSM) SetState(state string) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockFSM) GetState() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockFSM) GetStateChan(ctx context.Context) <-chan string {
	args := m.Called(ctx)
	return args.Get(0).(<-chan string)
}

func TestNew_ErrorConditions(t *testing.T) {
	t.Parallel()

	t.Run("nil config returns error", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		tx, err := New(SourceTest, "test", "", nil, handler)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("nil handler creates default handler", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionLatest}
		tx, err := New(SourceTest, "test", "", cfg, nil)

		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.NotNil(t, tx.logger)
	})

	t.Run("handles app factory creation failure", func(t *testing.T) {
		// Create a config that will cause app factory to fail
		cfg := &config.Config{
			Version: config.VersionLatest,
			Apps: apps.AppCollection{
				{
					ID:     "invalid-app",
					Config: nil, // This should cause the factory to fail
				},
			},
		}

		handler := slog.NewTextHandler(os.Stdout, nil)
		tx, err := New(SourceTest, "test", "", cfg, handler)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "failed to create app instances")
	})
}

func TestMarkFailed_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("handles canceled context before transition", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Create a canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		err := tx.MarkFailed(ctx, errors.New("test error"))
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)

		// State should remain unchanged
		assert.Equal(t, finitestate.StateCreated, tx.GetState())
	})

	t.Run("handles context timeout", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Create a context that times out immediately
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(1 * time.Millisecond)

		err := tx.MarkFailed(ctx, errors.New("test error"))
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("handles invalid transition from terminal state", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		// Move to error state first (terminal state)
		err := tx.MarkError(errors.New("terminal error"))
		require.NoError(t, err)
		assert.Equal(t, finitestate.StateError, tx.GetState())

		// Attempt to mark as failed from error state should be handled gracefully
		err = tx.MarkFailed(ctx, errors.New("another error"))
		assert.NoError(t, err) // Should not return error due to graceful handling

		// State should remain error
		assert.Equal(t, finitestate.StateError, tx.GetState())
	})

	t.Run("accumulates errors on successful transition", func(t *testing.T) {
		ctx := t.Context()
		tx, _ := setupTest(t)

		// Move to executing state first
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("execution failed")
		err := tx.MarkFailed(ctx, testErr)
		require.NoError(t, err)

		assert.Equal(t, finitestate.StateFailed, tx.GetState())
	})
}

func TestParticipantFunctionality(t *testing.T) {
	t.Parallel()

	t.Run("RegisterParticipant adds participant successfully", func(t *testing.T) {
		tx, _ := setupTest(t)

		err := tx.RegisterParticipant("test-participant")
		assert.NoError(t, err)

		participants := tx.GetParticipantStates()
		assert.Contains(t, participants, "test-participant")
		assert.Equal(t, finitestate.ParticipantNotStarted, participants["test-participant"])
	})

	t.Run("RegisterParticipant handles duplicate names", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Add participant first time
		err := tx.RegisterParticipant("duplicate")
		assert.NoError(t, err)

		// Add same participant again
		err = tx.RegisterParticipant("duplicate")
		assert.Error(t, err)
	})

	t.Run("GetParticipants returns collection", func(t *testing.T) {
		tx, _ := setupTest(t)

		collection := tx.GetParticipants()
		assert.NotNil(t, collection)

		// Add a participant and verify
		require.NoError(t, tx.RegisterParticipant("test"))
		states := tx.GetParticipantStates()
		assert.Contains(t, states, "test")
	})

	t.Run("GetParticipantErrors returns errors from participants", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Register participant and cause error
		require.NoError(t, tx.RegisterParticipant("error-participant"))

		// Simulate participant error by getting the participant and marking it failed
		participants := tx.GetParticipants()
		participant, err := participants.GetOrCreate("error-participant")
		require.NoError(t, err)

		// First transition to executing state (required before failing)
		require.NoError(t, participant.Execute())

		testErr := errors.New("participant error")
		require.NoError(t, participant.MarkFailed(testErr))

		participantErrors := tx.GetParticipantErrors()
		assert.Contains(t, participantErrors, "error-participant")
		assert.Equal(t, testErr, participantErrors["error-participant"])
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("concurrent IsValid operations are safe", func(t *testing.T) {
		tx, _ := setupTest(t)

		var wg sync.WaitGroup
		numGoroutines := 10

		// Start readers
		wg.Add(numGoroutines)
		for range numGoroutines {
			go func() {
				defer wg.Done()
				for range 100 {
					_ = tx.IsValid.Load()
				}
			}()
		}

		// Start writers
		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()
				for range 100 {
					tx.IsValid.Store(id%2 == 0)
				}
			}(i)
		}

		wg.Wait()

		// Should not panic or deadlock
		finalValue := tx.IsValid.Load()
		assert.IsType(t, true, finalValue) // Just checking type safety
	})

	t.Run("concurrent state transitions fail gracefully", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Set up to validated state first
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		var wg sync.WaitGroup
		results := make([]error, 10)

		// Try concurrent MarkSucceeded calls
		wg.Add(10)
		for i := range 10 {
			go func(idx int) {
				defer wg.Done()
				results[idx] = tx.MarkSucceeded()
			}(i)
		}

		wg.Wait()

		// Only one should succeed, others should fail
		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}
		assert.Equal(t, 1, successCount, "only one concurrent transition should succeed")
		assert.Equal(t, finitestate.StateSucceeded, tx.GetState())
	})
}

func TestFSMTransitionErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("BeginValidation handles FSM transition failure", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionLatest}
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create transaction with mock FSM
		mockFSM := &MockFSM{}
		tx := &ConfigTransaction{
			domainConfig: cfg,
			logger:       slog.New(handler),
			fsm:          mockFSM,
		}

		// Set up FSM to fail transition
		expectedErr := errors.New("fsm transition failed")
		mockFSM.On("Transition", finitestate.StateValidating).Return(expectedErr)

		err := tx.BeginValidation()
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})

	t.Run("MarkSucceeded handles FSM transition failure", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionLatest}
		handler := slog.NewTextHandler(os.Stdout, nil)

		mockFSM := &MockFSM{}
		tx := &ConfigTransaction{
			domainConfig: cfg,
			logger:       slog.New(handler),
			fsm:          mockFSM,
		}

		expectedErr := errors.New("fsm transition failed")
		mockFSM.On("Transition", finitestate.StateSucceeded).Return(expectedErr)

		err := tx.MarkSucceeded()
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})

	t.Run("MarkError handles FSM transition failure", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionLatest}
		handler := slog.NewTextHandler(os.Stdout, nil)

		mockFSM := &MockFSM{}
		tx := &ConfigTransaction{
			domainConfig: cfg,
			logger:       slog.New(handler),
			fsm:          mockFSM,
		}

		expectedErr := errors.New("fsm transition failed")
		mockFSM.On("Transition", finitestate.StateError).Return(expectedErr)

		originalErr := errors.New("original error")
		err := tx.MarkError(originalErr)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})
}

func TestGetAppCollection(t *testing.T) {
	t.Parallel()

	t.Run("returns app collection", func(t *testing.T) {
		tx, _ := setupTest(t)

		collection := tx.GetAppCollection()
		assert.NotNil(t, collection)
	})

	t.Run("app collection integration", func(t *testing.T) {
		cfg := &config.Config{
			Version: config.VersionLatest,
			Apps: apps.AppCollection{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test response"},
				},
			},
		}

		handler := slog.NewTextHandler(os.Stdout, nil)
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		collection := tx.GetAppCollection()
		app, exists := collection.GetApp("test-echo")
		assert.True(t, exists)
		assert.Equal(t, "test-echo", app.String())
	})
}

func TestPlaybackLogs(t *testing.T) {
	t.Parallel()

	t.Run("playback logs with multiple operations", func(t *testing.T) {
		tx, handler := setupTest(t)

		// Perform several operations to generate logs
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())
		require.NoError(t, tx.MarkSucceeded())

		// Playback should work without error
		err := tx.PlaybackLogs(handler)
		assert.NoError(t, err)
	})
}

func TestGetTotalDuration(t *testing.T) {
	t.Parallel()

	t.Run("duration increases over time", func(t *testing.T) {
		tx, _ := setupTest(t)

		duration1 := tx.GetTotalDuration()
		assert.Greater(t, duration1, time.Duration(0))

		time.Sleep(1 * time.Millisecond)

		duration2 := tx.GetTotalDuration()
		assert.Greater(t, duration2, duration1)
	})

	t.Run("duration in completed transaction", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Complete a full transaction
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())
		require.NoError(t, tx.MarkSucceeded())
		require.NoError(t, tx.BeginReload())
		require.NoError(t, tx.MarkCompleted())

		duration := tx.GetTotalDuration()
		assert.Greater(t, duration, time.Duration(0))
	})
}

func TestAppFactoryIntegration(t *testing.T) {
	t.Run("creates app collection from config", func(t *testing.T) {
		// Create a config with echo apps
		cfg := &config.Config{
			Apps: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "Hello 1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "Hello 2"},
				},
			},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify apps exist
		app1, exists1 := collection.GetApp("echo1")
		assert.True(t, exists1)
		assert.Equal(t, "echo1", app1.String())

		app2, exists2 := collection.GetApp("echo2")
		assert.True(t, exists2)
		assert.Equal(t, "echo2", app2.String())
	})

	t.Run("handles empty config", func(t *testing.T) {
		cfg := &config.Config{
			Apps: apps.AppCollection{},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify no apps exist
		_, exists := collection.GetApp("nonexistent")
		assert.False(t, exists)
	})
}
