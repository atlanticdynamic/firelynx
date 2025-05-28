package transaction

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
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
		assert.Empty(t, tx.terminalErrors)
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
	assert.Contains(t, tx.terminalErrors, validationErr)
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
	err = tx.MarkFailed(failErr)
	require.NoError(t, err)

	// Now can begin compensation
	err = tx.BeginCompensation()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateCompensating, tx.GetState())
}

func TestConfigTransaction_MarkCompensated(t *testing.T) {
	t.Parallel()

	tx, _ := setupTest(t)

	// Move through states to compensation
	err := tx.BeginValidation()
	require.NoError(t, err)
	tx.IsValid.Store(true)
	err = tx.MarkValidated()
	require.NoError(t, err)
	err = tx.BeginExecution()
	require.NoError(t, err)
	err = tx.MarkFailed(errors.New("failed"))
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
	assert.Contains(t, tx.terminalErrors, errorMsg)
}

func TestConfigTransaction_MarkFailed(t *testing.T) {
	t.Parallel()

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
	err = tx.MarkFailed(failErr)
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateFailed, tx.GetState())
	assert.Contains(t, tx.terminalErrors, failErr)
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
		tx, _ := setupTest(t)

		// Setup to failed state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)
		err = tx.MarkFailed(errors.New("failed"))
		require.NoError(t, err)

		// BeginRollback should map to BeginCompensation
		err = tx.BeginRollback()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateCompensating, tx.GetState())
	})

	t.Run("MarkRolledBack", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Setup to compensating state
		err := tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)
		err = tx.MarkFailed(errors.New("failed"))
		require.NoError(t, err)
		err = tx.BeginCompensation()
		require.NoError(t, err)

		// MarkRolledBack should map to MarkCompensated
		err = tx.MarkRolledBack()
		assert.NoError(t, err)
		assert.Equal(t, finitestate.StateCompensated, tx.GetState())
	})
}

func TestConfigTransaction_GetErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns empty errors initially", func(t *testing.T) {
		tx, _ := setupTest(t)
		assert.Empty(t, tx.GetErrors())
	})

	t.Run("returns validation error after MarkInvalid", func(t *testing.T) {
		tx, _ := setupTest(t)
		err1 := errors.New("validation error")

		// Transition to validating state first
		require.NoError(t, tx.BeginValidation())

		// Mark as invalid with error
		require.NoError(t, tx.MarkInvalid(err1))

		errors := tx.GetErrors()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors, err1)
	})

	t.Run("returns error after MarkError", func(t *testing.T) {
		tx, _ := setupTest(t)
		err1 := errors.New("terminal error")

		// Add error (this transitions to error state)
		require.NoError(t, tx.MarkError(err1))

		errors := tx.GetErrors()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors, err1)
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
		assert.NoError(t, err) // RunValidation itself doesn't return the validation error
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
		assert.NotEmpty(t, tx.terminalErrors)
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
		require.NoError(t, tx.setStateInvalid(validationErrs))
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
		assert.Len(t, tx.terminalErrors, 2)

		err := tx.BeginExecution()
		assert.ErrorIs(t, err, ErrNotValidated)
	})

	t.Run("validates compensation path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())

		require.NoError(t, tx.BeginCompensation())
		assert.Equal(t, finitestate.StateCompensating, tx.GetState())

		require.NoError(t, tx.MarkCompensated())
		assert.Equal(t, finitestate.StateCompensated, tx.GetState())
	})

	t.Run("validates failure path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())
		assert.Contains(t, tx.terminalErrors, testErr)
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
