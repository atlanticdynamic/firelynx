package transaction

import (
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
