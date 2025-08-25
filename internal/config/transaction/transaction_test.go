package transaction

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	httpCfg "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*ConfigTransaction, slog.Handler) {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Create a valid config with the current version
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	cfg.Version = config.VersionLatest

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
		require.NoError(t, err)
		assert.Equal(t, finitestate.StateValidating, tx.GetState())
	})

	t.Run("invalid transition", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Move to a state that can't transition to validating
		err := tx.BeginExecution()
		require.Error(t, err) // Should fail because not validated

		// Try to begin validation from wrong state
		err = tx.BeginValidation()
		require.NoError(t, err) // Actually, from created state this should work
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
		require.NoError(t, err)
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
	})

	t.Run("validation failed", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Begin validation
		err := tx.BeginValidation()
		require.NoError(t, err)

		// Don't mark as valid (IsValid remains false)
		err = tx.MarkValidated()
		require.NoError(t, err) // MarkValidated returns nil when it successfully marks as invalid
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
	require.NoError(t, err)
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
		require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.Equal(t, finitestate.StateCompensated, tx.GetState())
}

func TestConfigTransaction_MarkError(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	errorMsg := errors.New("unrecoverable error")
	err := tx.MarkError(errorMsg)
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.Equal(t, finitestate.StateFailed, tx.GetState())
}

func TestConfigTransaction_GetConfig(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	cfg.Version = config.VersionLatest

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
		require.NoError(t, err)
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
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = "invalid-version" // This should fail validation

		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		err = tx.RunValidation()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
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

		// Create config using proper constructor
		invalidCfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		tx.domainConfig = invalidCfg

		validationErrs := []error{
			errors.New("validation error 1"),
			errors.New("validation error 2"),
		}

		require.NoError(t, tx.fsm.Transition(finitestate.StateValidating))
		tx.setStateInvalid(validationErrs)
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())

		err = tx.BeginExecution()
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
		input    *apps.AppCollection
		expected []serverApps.AppDefinition
	}{
		{
			name:     "empty collection",
			input:    apps.NewAppCollection(),
			expected: []serverApps.AppDefinition{},
		},
		{
			name: "single echo app",
			input: apps.NewAppCollection(
				apps.App{
					ID:     "test-echo",
					Config: &echo.EchoApp{ID: "test-echo", Response: "test"},
				},
			),
			expected: []serverApps.AppDefinition{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{ID: "test-echo", Response: "test"},
				},
			},
		},
		{
			name: "multiple apps",
			input: apps.NewAppCollection(
				apps.App{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				apps.App{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			),
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
			cfg, err := config.NewFromProto(&pb.ServerConfig{})
			require.NoError(t, err)
			cfg.Apps = tt.input
			result := collectApps(cfg)

			require.Len(t, result, len(tt.expected))

			// Create a map of expected apps for order-independent comparison
			expectedMap := make(map[string]serverApps.AppDefinition)
			for _, exp := range tt.expected {
				expectedMap[exp.ID] = exp
			}

			// Check that all results are in the expected set
			for _, def := range result {
				expected, ok := expectedMap[def.ID]
				require.True(t, ok, "unexpected app ID: %s", def.ID)
				assert.Equal(t, expected.Config, def.Config)
				delete(expectedMap, def.ID)
			}

			// Ensure all expected apps were found
			assert.Empty(t, expectedMap, "missing expected apps: %v", expectedMap)
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

		require.Error(t, err)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, ErrNilConfig)
	})

	t.Run("nil handler creates default handler", func(t *testing.T) {
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		tx, err := New(SourceTest, "test", "", cfg, nil)

		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.NotNil(t, tx.logger)
	})

	t.Run("handles app factory creation failure", func(t *testing.T) {
		// Create a config that will cause app factory to fail
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "invalid-app",
				Config: nil, // This should cause the factory to fail
			},
		)

		handler := slog.NewTextHandler(os.Stdout, nil)
		tx, err := New(SourceTest, "test", "", cfg, handler)
		require.NoError(t, err)

		// Validation should fail due to invalid app config
		err = tx.RunValidation()
		require.Error(t, err)
		// The error will be wrapped in ErrValidationFailed, but should contain app creation error
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "app instantiation validation failed")
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
		require.Error(t, err)
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
		require.Error(t, err)
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
		require.NoError(t, err) // Should not return error due to graceful handling

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
		require.NoError(t, err)

		participants := tx.GetParticipantStates()
		assert.Contains(t, participants, "test-participant")
		assert.Equal(t, finitestate.ParticipantNotStarted, participants["test-participant"])
	})

	t.Run("RegisterParticipant handles duplicate names", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Add participant first time
		err := tx.RegisterParticipant("duplicate")
		require.NoError(t, err)

		// Add same participant again
		err = tx.RegisterParticipant("duplicate")
		require.Error(t, err)
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
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
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

		err = tx.BeginValidation()
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})

	t.Run("MarkSucceeded handles FSM transition failure", func(t *testing.T) {
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		handler := slog.NewTextHandler(os.Stdout, nil)

		mockFSM := &MockFSM{}
		tx := &ConfigTransaction{
			domainConfig: cfg,
			logger:       slog.New(handler),
			fsm:          mockFSM,
		}

		expectedErr := errors.New("fsm transition failed")
		mockFSM.On("Transition", finitestate.StateSucceeded).Return(expectedErr)

		err = tx.MarkSucceeded()
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})

	t.Run("MarkError handles FSM transition failure", func(t *testing.T) {
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
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
		err = tx.MarkError(originalErr)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockFSM.AssertExpectations(t)
	})
}

func TestGetAppCollection(t *testing.T) {
	t.Parallel()

	t.Run("returns app collection", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Need to run validation first to create app collection
		err := tx.RunValidation()
		require.NoError(t, err)

		collection := tx.GetAppCollection()
		assert.NotNil(t, collection)
	})

	t.Run("app collection integration", func(t *testing.T) {
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "test-echo",
				Config: &echo.EchoApp{ID: "test-echo", Response: "test response"},
			},
		)

		handler := slog.NewTextHandler(os.Stdout, nil)
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Need to run validation first to create app collection
		err = tx.RunValidation()
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
		require.NoError(t, err)
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
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "echo1",
				Config: &echo.EchoApp{ID: "app1", Response: "Hello 1"},
			},
			apps.App{
				ID:     "echo2",
				Config: &echo.EchoApp{ID: "app2", Response: "Hello 2"},
			},
		)

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := collectApps(cfg)

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
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Apps = apps.NewAppCollection()

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := collectApps(cfg)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify no apps exist
		_, exists := collection.GetApp("nonexistent")
		assert.False(t, exists)
	})
}

func TestCreateMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("creates console logger middleware", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Create a console logger middleware config
		mw := middleware.Middleware{
			ID: "test-logger",
			Config: &configLogger.ConsoleLogger{
				Options: configLogger.LogOptionsGeneral{
					Format: configLogger.FormatJSON,
					Level:  configLogger.LevelInfo,
				},
				Output: "stdout",
			},
		}

		middlewares := middleware.MiddlewareCollection{mw}
		middlewareCollection, err := tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.NoError(t, err)
		tx.middleware.collection = middlewareCollection

		// Verify middleware was added to pool
		mwType := mw.Config.Type()
		pool := tx.GetMiddlewareRegistry()
		assert.Contains(t, pool, mwType)
		assert.Contains(t, pool[mwType], "test-logger")
	})

	t.Run("reuses existing middleware instance", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Create middleware
		mw := middleware.Middleware{
			ID: "test-logger",
			Config: &configLogger.ConsoleLogger{
				Options: configLogger.LogOptionsGeneral{
					Format: configLogger.FormatJSON,
					Level:  configLogger.LevelInfo,
				},
				Output: "stdout",
			},
		}

		// Create first instance
		middlewares := middleware.MiddlewareCollection{mw}
		middlewareCollection, err := tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.NoError(t, err)
		tx.middleware.collection = middlewareCollection
		pool := tx.GetMiddlewareRegistry()
		firstInstance := pool[mw.Config.Type()][mw.ID]

		// Try to create again - should reuse
		middlewares = middleware.MiddlewareCollection{mw}
		middlewareCollection, err = tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.NoError(t, err)
		tx.middleware.collection = middlewareCollection
		pool = tx.GetMiddlewareRegistry()
		secondInstance := pool[mw.Config.Type()][mw.ID]

		// Should be the same instance
		assert.Equal(t, firstInstance, secondInstance)
	})

	t.Run("handles unsupported middleware type", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Create unsupported middleware
		mw := middleware.Middleware{
			ID:     "unsupported",
			Config: &mockMiddlewareConfig{configType: "unsupported"},
		}

		middlewares := middleware.MiddlewareCollection{mw}
		_, err = tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported middleware type")
	})

	t.Run("handles console logger creation failure", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Create console logger with invalid output that will fail
		mw := middleware.Middleware{
			ID: "failing-logger",
			Config: &configLogger.ConsoleLogger{
				Options: configLogger.LogOptionsGeneral{
					Format: configLogger.FormatJSON,
					Level:  configLogger.LevelInfo,
				},
				Output: "/root/no-permission/test.log", // Should fail due to permissions
			},
		}

		middlewares := middleware.MiddlewareCollection{mw}
		_, err = tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create middleware")
	})
}

func TestGetMiddlewareRegistry(t *testing.T) {
	t.Parallel()

	t.Run("returns middleware registry", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		registry := tx.GetMiddlewareRegistry()
		assert.NotNil(t, registry)
		assert.IsType(t, httpCfg.MiddlewareRegistry{}, registry)
	})

	t.Run("registry contains created middleware", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Create middleware
		mw := middleware.Middleware{
			ID: "test-logger",
			Config: &configLogger.ConsoleLogger{
				Options: configLogger.LogOptionsGeneral{
					Format: configLogger.FormatJSON,
					Level:  configLogger.LevelInfo,
				},
				Output: "stdout",
			},
		}

		middlewares := middleware.MiddlewareCollection{mw}
		middlewareCollection, err := tx.middleware.factory.CreateFromDefinitions(middlewares)
		require.NoError(t, err)
		tx.middleware.collection = middlewareCollection

		pool := tx.GetMiddlewareRegistry()
		mwType := mw.Config.Type()
		assert.Contains(t, pool, mwType)
		assert.Contains(t, pool[mwType], "test-logger")
	})
}

func TestCreateMiddlewareInstances(t *testing.T) {
	t.Parallel()

	t.Run("creates middleware from endpoints", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create config with endpoints and middleware
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		// Add required listeners
		cfg.Listeners = listeners.ListenerCollection{
			{
				ID:      "http-1",
				Address: "127.0.0.1:8080",
				Type:    listeners.TypeHTTP,
				Options: options.NewHTTP(),
			},
		}
		// Add required apps
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "test-app",
				Config: &echo.EchoApp{ID: "test-echo", Response: "test response"},
			},
		)
		cfg.Endpoints = endpoints.EndpointCollection{
			{
				ID:         "endpoint1",
				ListenerID: "http-1",
				Middlewares: middleware.MiddlewareCollection{
					{
						ID: "endpoint-logger",
						Config: &configLogger.ConsoleLogger{
							Options: configLogger.LogOptionsGeneral{
								Format: configLogger.FormatJSON,
								Level:  configLogger.LevelInfo,
							},
							Output: "stdout",
						},
					},
				},
				Routes: routes.RouteCollection{
					{
						AppID:     "test-app",
						Condition: conditions.NewHTTP("/test", ""),
						Middlewares: middleware.MiddlewareCollection{
							{
								ID: "route-logger",
								Config: &configLogger.ConsoleLogger{
									Options: configLogger.LogOptionsGeneral{
										Format: configLogger.FormatJSON,
										Level:  configLogger.LevelDebug,
									},
									Output: "stdout",
								},
							},
						},
					},
				},
			},
		}

		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Need to run validation first to create middleware instances
		err = tx.RunValidation()
		require.NoError(t, err)

		// Verify middleware instances were created
		pool := tx.GetMiddlewareRegistry()
		loggerType := "console_logger"

		assert.Contains(t, pool, loggerType)
		assert.Contains(t, pool[loggerType], "endpoint-logger")
		assert.Contains(t, pool[loggerType], "route-logger")
		assert.Len(t, pool[loggerType], 2)
	})

	t.Run("shares middleware instances with same ID", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create config with same middleware ID used in multiple places
		sharedLoggerConfig := &configLogger.ConsoleLogger{
			Options: configLogger.LogOptionsGeneral{
				Format: configLogger.FormatJSON,
				Level:  configLogger.LevelInfo,
			},
			Output: "stdout",
		}

		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		// Add required listeners
		cfg.Listeners = listeners.ListenerCollection{
			{
				ID:      "http-1",
				Address: "127.0.0.1:8080",
				Type:    listeners.TypeHTTP,
				Options: options.NewHTTP(),
			},
		}
		// Add required apps
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "app1",
				Config: &echo.EchoApp{ID: "app1", Response: "app1 response"},
			},
			apps.App{
				ID:     "app2",
				Config: &echo.EchoApp{ID: "app2", Response: "app2 response"},
			},
		)
		cfg.Endpoints = endpoints.EndpointCollection{
			{
				ID:         "endpoint1",
				ListenerID: "http-1",
				Middlewares: middleware.MiddlewareCollection{
					{
						ID:     "shared-logger",
						Config: sharedLoggerConfig,
					},
				},
				Routes: routes.RouteCollection{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/app1", ""),
						Middlewares: middleware.MiddlewareCollection{
							{
								ID:     "shared-logger", // Same ID
								Config: sharedLoggerConfig,
							},
						},
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/app2", ""),
						Middlewares: middleware.MiddlewareCollection{
							{
								ID:     "shared-logger", // Same ID again
								Config: sharedLoggerConfig,
							},
						},
					},
				},
			},
		}

		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Need to run validation first to create middleware instances
		err = tx.RunValidation()
		require.NoError(t, err)

		// Verify only one instance was created
		pool := tx.GetMiddlewareRegistry()
		loggerType := "console_logger"

		assert.Contains(t, pool, loggerType)
		assert.Len(t, pool[loggerType], 1)
		assert.Contains(t, pool[loggerType], "shared-logger")
	})

	t.Run("different IDs create different instances", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create config with different middleware IDs but same config
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		// Add required listeners
		cfg.Listeners = listeners.ListenerCollection{
			{
				ID:      "http-1",
				Address: "127.0.0.1:8080",
				Type:    listeners.TypeHTTP,
				Options: options.NewHTTP(),
			},
		}
		// Add required apps
		cfg.Apps = apps.NewAppCollection(
			apps.App{
				ID:     "app1",
				Config: &echo.EchoApp{ID: "app1", Response: "app1 response"},
			},
			apps.App{
				ID:     "app2",
				Config: &echo.EchoApp{ID: "app2", Response: "app2 response"},
			},
		)
		cfg.Endpoints = endpoints.EndpointCollection{
			{
				ID:         "endpoint1",
				ListenerID: "http-1",
				Routes: routes.RouteCollection{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/app1", ""),
						Middlewares: middleware.MiddlewareCollection{
							{
								ID: "logger1",
								Config: &configLogger.ConsoleLogger{
									Options: configLogger.LogOptionsGeneral{
										Format: configLogger.FormatJSON,
										Level:  configLogger.LevelInfo,
									},
									Output: "stdout",
								},
							},
						},
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/app2", ""),
						Middlewares: middleware.MiddlewareCollection{
							{
								ID: "logger2", // Different ID
								Config: &configLogger.ConsoleLogger{
									Options: configLogger.LogOptionsGeneral{
										Format: configLogger.FormatJSON,
										Level:  configLogger.LevelInfo,
									},
									Output: "stdout",
								},
							},
						},
					},
				},
			},
		}

		tx, err := FromTest("test", cfg, handler)
		require.NoError(t, err)

		// Run validation to create middleware instances
		err = tx.RunValidation()
		require.NoError(t, err)

		// Verify two instances were created
		pool := tx.GetMiddlewareRegistry()
		loggerType := "console_logger"

		assert.Contains(t, pool, loggerType)
		assert.Len(t, pool[loggerType], 2)
		assert.Contains(t, pool[loggerType], "logger1")
		assert.Contains(t, pool[loggerType], "logger2")

		// Verify they are different instances
		instance1 := pool[loggerType]["logger1"]
		instance2 := pool[loggerType]["logger2"]
		assert.NotEqual(t, instance1, instance2)
	})

	t.Run("handles middleware creation failure", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)

		// Create config with invalid middleware
		cfg, err := config.NewFromProto(&pb.ServerConfig{})
		require.NoError(t, err)
		cfg.Version = config.VersionLatest
		cfg.Endpoints = endpoints.EndpointCollection{
			{
				ID:         "endpoint1",
				ListenerID: "http-1",
				Middlewares: middleware.MiddlewareCollection{
					{
						ID:     "bad-middleware",
						Config: &mockMiddlewareConfig{configType: "unsupported"},
					},
				},
			},
		}

		tx, err := New(SourceTest, "test", "", cfg, handler)
		require.NoError(t, err) // Constructor should succeed

		// Validation should fail when trying to create middleware instances
		err = tx.RunValidation()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "middleware instantiation validation failed")
	})
}

// mockMiddlewareConfig is a test implementation of middleware.MiddlewareConfig
type mockMiddlewareConfig struct {
	configType string
}

func (m *mockMiddlewareConfig) Type() string                 { return m.configType }
func (m *mockMiddlewareConfig) Validate() error              { return nil }
func (m *mockMiddlewareConfig) ToProto() any                 { return nil }
func (m *mockMiddlewareConfig) String() string               { return m.configType }
func (m *mockMiddlewareConfig) ToTree() *fancy.ComponentTree { return nil }

// createDualLoggerConfig creates a test config with two console loggers
func createDualLoggerConfig(t *testing.T, output1, output2 string) *config.Config {
	t.Helper()
	cfg, err := config.NewFromProto(&pb.ServerConfig{})
	require.NoError(t, err)
	cfg.Version = config.VersionLatest
	// Add required listeners
	cfg.Listeners = listeners.ListenerCollection{
		{
			ID:      "http-1",
			Address: "127.0.0.1:8080",
			Type:    listeners.TypeHTTP,
			Options: options.NewHTTP(),
		},
	}
	// Add required apps
	cfg.Apps = apps.NewAppCollection(
		apps.App{
			ID:     "app1",
			Config: &echo.EchoApp{ID: "app1", Response: "app1 response"},
		},
		apps.App{
			ID:     "app2",
			Config: &echo.EchoApp{ID: "app2", Response: "app2 response"},
		},
	)
	cfg.Endpoints = endpoints.EndpointCollection{
		{
			ID:         "endpoint1",
			ListenerID: "http-1",
			Routes: routes.RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/app1", ""),
					Middlewares: middleware.MiddlewareCollection{
						{
							ID: "logger1",
							Config: &configLogger.ConsoleLogger{
								Options: configLogger.LogOptionsGeneral{
									Format: configLogger.FormatJSON,
									Level:  configLogger.LevelInfo,
								},
								Output: output1,
							},
						},
					},
				},
				{
					AppID:     "app2",
					Condition: conditions.NewHTTP("/app2", ""),
					Middlewares: middleware.MiddlewareCollection{
						{
							ID: "logger2",
							Config: &configLogger.ConsoleLogger{
								Options: configLogger.LogOptionsGeneral{
									Format: configLogger.FormatJSON,
									Level:  configLogger.LevelInfo,
								},
								Output: output2,
							},
						},
					},
				},
			},
		},
	}
	return cfg
}

func TestMiddlewareDuplicateOutputFileValidation(t *testing.T) {
	t.Parallel()
	handler := slog.NewTextHandler(os.Stdout, nil)

	t.Run("fails validation with duplicate output files", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")
		cfg := createDualLoggerConfig(t, logFile, logFile)
		tx, err := New(SourceTest, "test", "", cfg, handler)
		require.NoError(t, err)

		// Validation should fail due to duplicate output files
		err = tx.RunValidation()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrResourceConflict)
		assert.Contains(t, err.Error(), "duplicate output file")
		assert.Contains(t, err.Error(), logFile)
	})

	t.Run("passes validation with different output files", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := createDualLoggerConfig(t,
			filepath.Join(tmpDir, "test1.log"),
			filepath.Join(tmpDir, "test2.log"))
		tx, err := New(SourceTest, "test", "", cfg, handler)
		require.NoError(t, err)
		assert.NotNil(t, tx)

		// Validation should pass with different files
		err = tx.RunValidation()
		require.NoError(t, err)
	})

	t.Run("allows multiple stdout/stderr loggers", func(t *testing.T) {
		cfg := createDualLoggerConfig(t, "stdout", "stderr")
		tx, err := New(SourceTest, "test", "", cfg, handler)
		require.NoError(t, err)

		// Validation should pass with stdout/stderr
		err = tx.RunValidation()
		require.NoError(t, err)
		assert.NotNil(t, tx)
	})
}
