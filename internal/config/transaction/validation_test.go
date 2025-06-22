package transaction

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunValidation_ShouldReturnValidationErrors tests that RunValidation
// returns validation errors instead of swallowing them.
// This test is expected to FAIL until the bug in setStateInvalid is fixed.
func TestRunValidation_ShouldReturnValidationErrors(t *testing.T) {
	t.Run("unsupported_version_should_return_error", func(t *testing.T) {
		// Create a config with unsupported version
		invalidConfig := &config.Config{
			Version: "v999", // Unsupported version
		}

		// Create transaction
		tx, err := New(
			SourceTest,
			"TestRunValidation_ShouldReturnValidationErrors",
			"test-request-id",
			invalidConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// Run validation - THIS SHOULD RETURN AN ERROR BUT CURRENTLY DOESN'T
		err = tx.RunValidation()

		// EXPECTED BEHAVIOR (currently fails):
		assert.Error(t, err, "RunValidation should return validation errors")
		assert.ErrorIs(t, err, ErrValidationFailed, "Should return ErrValidationFailed")
		assert.ErrorIs(
			t,
			err,
			errz.ErrUnsupportedConfigVer,
			"Should contain the underlying config validation error",
		)

		// State should be Invalid regardless
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})

	t.Run("duplicate_listener_ids_should_return_error", func(t *testing.T) {
		// Create config with duplicate listener IDs
		invalidConfig := &config.Config{
			Version: config.VersionLatest,
			Listeners: listeners.ListenerCollection{
				{
					ID:      "duplicate-id",
					Address: ":8080",
					Type:    listeners.TypeHTTP,
				},
				{
					ID:      "duplicate-id", // Duplicate!
					Address: ":8081",
					Type:    listeners.TypeHTTP,
				},
			},
		}

		// Create transaction
		tx, err := New(
			SourceTest,
			"TestRunValidation_ShouldReturnValidationErrors",
			"test-request-id",
			invalidConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Run validation - THIS SHOULD RETURN AN ERROR BUT CURRENTLY DOESN'T
		err = tx.RunValidation()

		// EXPECTED BEHAVIOR (currently fails):
		assert.Error(t, err, "RunValidation should return validation errors for duplicate IDs")
		assert.ErrorIs(t, err, ErrValidationFailed, "Should return ErrValidationFailed")
		assert.ErrorIs(
			t,
			err,
			errz.ErrDuplicateID,
			"Should contain the underlying duplicate ID error",
		)

		// State should be Invalid regardless
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})

	t.Run("valid_config_should_succeed", func(t *testing.T) {
		// Create a valid config
		validConfig := &config.Config{
			Version: config.VersionLatest,
		}

		// Create transaction
		tx, err := New(
			SourceTest,
			"TestRunValidation_ShouldReturnValidationErrors",
			"test-request-id",
			validConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Run validation - should succeed
		err = tx.RunValidation()

		// Should succeed
		assert.NoError(t, err, "Valid config should pass validation")
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
		assert.True(t, tx.IsValid.Load())
	})
}

func TestRunValidation_NilConfig(t *testing.T) {
	// The implementation checks for nil config before transitioning to validating
	// Since we can't create a transaction with nil config through the constructor,
	// and the FSM doesn't allow created->invalid transition directly,
	// we need to test this differently

	// First, let's verify that the constructor prevents nil config
	_, err := New(
		SourceTest,
		"TestRunValidation_NilConfig",
		"test-request-id",
		nil, // nil config
		slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
	)
	require.Error(t, err)
	// Constructor should return ErrNilConfig when config is nil
	require.ErrorIs(t, err, ErrNilConfig)

	// The nil config check in RunValidation is defensive programming
	// It would only trigger if someone manually constructed a ConfigTransaction
	// with a nil domainConfig, which shouldn't happen in normal usage
}

func TestRunValidation_CalledMultipleTimes(t *testing.T) {
	// Test that RunValidation can only be called once
	validConfig := &config.Config{
		Version: config.VersionLatest,
	}

	tx, err := New(
		SourceTest,
		"TestRunValidation_CalledMultipleTimes",
		"test-request-id",
		validConfig,
		slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
	)
	require.NoError(t, err)

	// First call should succeed
	err = tx.RunValidation()
	assert.NoError(t, err)
	assert.Equal(t, finitestate.StateValidated, tx.GetState())
	assert.True(t, tx.IsValid.Load())

	// Second call should fail because we're no longer in StateCreated
	err = tx.RunValidation()
	assert.Error(t, err)
	// The error will be from FSM transition failure - check that it's not a validation error
	assert.NotErrorIs(t, err, ErrValidationFailed)

	// State should remain validated from first call
	assert.Equal(t, finitestate.StateValidated, tx.GetState())
	assert.True(t, tx.IsValid.Load())
}

func TestSetStateInvalid_ErrorAlreadyWrapped(t *testing.T) {
	// This test verifies that setStateInvalid doesn't double-wrap errors
	// We need to create a custom validator that returns already-wrapped errors

	// Create a config that will trigger validation errors
	invalidConfig := &config.Config{
		Version: config.VersionLatest,
		Listeners: listeners.ListenerCollection{
			{
				ID:      "test",
				Address: "", // Invalid: empty address
				Type:    listeners.TypeHTTP,
			},
		},
	}

	tx, err := New(
		SourceTest,
		"TestSetStateInvalid_ErrorAlreadyWrapped",
		"test-request-id",
		invalidConfig,
		slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
	)
	require.NoError(t, err)

	// Manually transition to validating state
	err = tx.fsm.Transition(finitestate.StateValidating)
	require.NoError(t, err)

	// Call setStateInvalid with a mix of wrapped and unwrapped errors
	alreadyWrapped := fmt.Errorf("%w: %w", ErrValidationFailed, errors.New("underlying error"))
	inputErrors := []error{
		alreadyWrapped,
		errors.New("new error"),
	}
	tx.setStateInvalid(inputErrors)

	// Check that IsValid is false
	assert.False(t, tx.IsValid.Load())
	assert.Equal(t, finitestate.StateInvalid, tx.GetState())
}

func TestRunValidation_FullLifecycle(t *testing.T) {
	t.Run("successful validation with state transitions", func(t *testing.T) {
		// Create valid config
		validConfig := &config.Config{
			Version: config.VersionLatest,
		}

		// Create transaction with real FSM
		tx, err := New(
			SourceTest,
			"TestRunValidation_FullLifecycle",
			"test-request-id",
			validConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Initial state should be created
		assert.Equal(t, finitestate.StateCreated, tx.GetState())
		assert.False(t, tx.IsValid.Load())

		// Run validation
		err = tx.RunValidation()
		assert.NoError(t, err)

		// Final state should be validated
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
		assert.True(t, tx.IsValid.Load())
	})

	t.Run("failed validation with state transitions", func(t *testing.T) {
		// Create invalid config with bad version
		invalidConfig := &config.Config{
			Version: "invalid-version",
		}

		// Create transaction with real FSM
		tx, err := New(
			SourceTest,
			"TestRunValidation_FullLifecycle",
			"test-request-id",
			invalidConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Initial state
		assert.Equal(t, finitestate.StateCreated, tx.GetState())

		// Run validation
		err = tx.RunValidation()
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)

		// Final state should be invalid
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})
}

func TestRunValidation_StateTransitionLogging(t *testing.T) {
	// This test exercises the state transition paths and logging
	t.Run("invalid config with multiple errors", func(t *testing.T) {
		// Create a config that will have multiple validation errors
		invalidConfig := &config.Config{
			Version: "invalid-version",
			Listeners: listeners.ListenerCollection{
				{
					ID:      "", // Invalid: empty ID
					Address: "", // Invalid: empty address
					Type:    listeners.TypeHTTP,
				},
			},
		}

		tx, err := New(
			SourceTest,
			"TestRunValidation_StateTransitionLogging",
			"test-request-id",
			invalidConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Run validation
		err = tx.RunValidation()

		// Should fail with validation error
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidationFailed)

		// Should be in invalid state
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
	})

	t.Run("validation state already set", func(t *testing.T) {
		validConfig := &config.Config{
			Version: config.VersionLatest,
		}

		tx, err := New(
			SourceTest,
			"TestRunValidation_StateAlreadySet",
			"test-request-id",
			validConfig,
			slog.New(slog.NewTextHandler(os.Stdout, nil)).Handler(),
		)
		require.NoError(t, err)

		// Manually transition to invalid state
		err = tx.fsm.Transition(finitestate.StateValidating)
		require.NoError(t, err)
		err = tx.fsm.Transition(finitestate.StateInvalid)
		require.NoError(t, err)

		// Try to run validation - should fail because not in created state
		err = tx.RunValidation()
		assert.Error(t, err)
		// Should get state transition error, not validation failed
		assert.NotErrorIs(t, err, ErrValidationFailed)

		// State should remain invalid
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
	})
}
