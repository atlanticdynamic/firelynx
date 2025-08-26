package scripts

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/require"
)

func TestAppScript_Validate(t *testing.T) {
	// Create mock static data with validation errors
	invalidStaticData := &staticdata.StaticData{
		MergeMode: 999, // Invalid merge mode to trigger validation error
	}

	// Create mock evaluator with validation errors
	invalidEvaluator := &evaluators.RisorEvaluator{
		// Empty code will trigger validation error
	}

	// Create valid static data and evaluator
	validStaticData := &staticdata.StaticData{
		Data: map[string]any{
			"key": "value",
		},
	}
	validEvaluator := &evaluators.RisorEvaluator{
		Code:    "print('hello')",
		Timeout: 5 * time.Second,
	}

	t.Run("valid script with all fields", func(t *testing.T) {
		script := &AppScript{
			ID:         "valid-script",
			StaticData: validStaticData,
			Evaluator:  validEvaluator,
		}
		err := script.Validate()
		require.NoError(t, err)
	})

	t.Run("valid script without static data", func(t *testing.T) {
		script := &AppScript{
			ID:        "simple-script",
			Evaluator: validEvaluator,
		}
		err := script.Validate()
		require.NoError(t, err)
	})

	t.Run("missing evaluator", func(t *testing.T) {
		script := &AppScript{
			StaticData: validStaticData,
		}
		err := script.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingEvaluator)
	})

	t.Run("invalid evaluator", func(t *testing.T) {
		script := &AppScript{
			StaticData: validStaticData,
			Evaluator:  invalidEvaluator,
		}
		err := script.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidEvaluator)
	})

	t.Run("invalid static data", func(t *testing.T) {
		script := &AppScript{
			StaticData: invalidStaticData,
			Evaluator:  validEvaluator,
		}
		err := script.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidStaticData)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		script := &AppScript{
			StaticData: invalidStaticData,
			Evaluator:  invalidEvaluator,
		}
		err := script.Validate()
		require.Error(t, err)
		// Should contain both error types when both validations fail
		require.ErrorIs(t, err, ErrInvalidEvaluator, "Should contain evaluator error")
		require.ErrorIs(t, err, ErrInvalidStaticData, "Should contain static data error")
	})
}
