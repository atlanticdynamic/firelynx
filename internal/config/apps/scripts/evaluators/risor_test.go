//nolint:dupl
package evaluators

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRisorEvaluator_Type(t *testing.T) {
	risor := &RisorEvaluator{}
	assert.Equal(t, EvaluatorTypeRisor, risor.Type())
}

func TestRisorEvaluator_String(t *testing.T) {
	tests := []struct {
		name      string
		evaluator *RisorEvaluator
		want      string
	}{
		{
			name:      "nil",
			evaluator: nil,
			want:      "Risor(nil)",
		},
		{
			name:      "empty",
			evaluator: &RisorEvaluator{},
			want:      "Risor(code=0 chars, timeout=0s)",
		},
		{
			name: "with code and timeout",
			evaluator: &RisorEvaluator{
				Code:    "print('hello')",
				Timeout: 5 * time.Second,
			},
			want: "Risor(code=14 chars, timeout=5s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.evaluator.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRisorEvaluator_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "print('hello')",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.NoError(t, err)
	})

	t.Run("empty code", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingCodeAndURI)
	})

	t.Run("negative timeout", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "print('hello')",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNegativeTimeout))
	})

	t.Run("multiple errors", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingCodeAndURI)
		assert.True(t, errors.Is(err, ErrNegativeTimeout))
	})
}

func TestRisorEvaluator_GetCompiledEvaluator(t *testing.T) {
	t.Run("nil evaluator", func(t *testing.T) {
		evaluator := &RisorEvaluator{}
		result := evaluator.GetCompiledEvaluator()
		assert.Nil(t, result)
	})

	t.Run("non-nil evaluator", func(t *testing.T) {
		evaluator := &RisorEvaluator{}
		result := evaluator.GetCompiledEvaluator()
		assert.Nil(t, result)

		// TODO: Add test for compiled evaluator when Phase 2.1 is implemented
		// Test should verify:
		// 1. After calling enhanced Validate() with valid Risor code, GetCompiledEvaluator() returns non-nil platform.Evaluator
		// 2. After calling enhanced Validate() with invalid Risor code, Validate() returns error and GetCompiledEvaluator() returns nil
		// 3. The returned evaluator can execute simple Risor scripts (integration test with go-polyscript)
	})
}
