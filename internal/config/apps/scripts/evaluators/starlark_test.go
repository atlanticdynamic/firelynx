//nolint:dupl
package evaluators

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStarlarkEvaluator_Type(t *testing.T) {
	starlark := &StarlarkEvaluator{}
	assert.Equal(t, EvaluatorTypeStarlark, starlark.Type())
}

func TestStarlarkEvaluator_String(t *testing.T) {
	tests := []struct {
		name      string
		evaluator *StarlarkEvaluator
		want      string
	}{
		{
			name:      "nil",
			evaluator: nil,
			want:      "Starlark(nil)",
		},
		{
			name:      "empty",
			evaluator: &StarlarkEvaluator{},
			want:      "Starlark(code=0 chars, timeout=0s)",
		},
		{
			name: "with code and timeout",
			evaluator: &StarlarkEvaluator{
				Code:    "print('hello')",
				Timeout: 5 * time.Second,
			},
			want: "Starlark(code=14 chars, timeout=5s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.evaluator.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStarlarkEvaluator_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{
			Code:    "print('hello')",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.NoError(t, err)
	})

	t.Run("empty code", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{
			Code:    "",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingCodeAndURI)
	})

	t.Run("negative timeout", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{
			Code:    "print('hello')",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNegativeTimeout))
	})

	t.Run("multiple errors", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{
			Code:    "",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingCodeAndURI)
		assert.True(t, errors.Is(err, ErrNegativeTimeout))
	})
}

func TestStarlarkEvaluator_GetCompiledEvaluator(t *testing.T) {
	t.Run("nil evaluator", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{}
		result, err := evaluator.GetCompiledEvaluator()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-nil evaluator", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{}
		result, err := evaluator.GetCompiledEvaluator()
		assert.Error(t, err)
		assert.Nil(t, result)

		// TODO: Add test for compiled evaluator when Phase 2.1 is implemented
		// Test should verify:
		// 1. After calling enhanced Validate() with valid Starlark code, GetCompiledEvaluator() returns non-nil platform.Evaluator
		// 2. After calling enhanced Validate() with invalid Starlark code, Validate() returns error and GetCompiledEvaluator() returns nil
		// 3. The returned evaluator can execute simple Starlark scripts (integration test with go-polyscript)
	})
}
