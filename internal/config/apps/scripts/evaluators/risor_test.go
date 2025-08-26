//nolint:dupl
package evaluators

import (
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
		require.ErrorIs(t, err, ErrMissingCodeAndURI)
	})

	t.Run("negative timeout", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "print('hello')",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNegativeTimeout)
	})

	t.Run("multiple errors", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "",
			Timeout: -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingCodeAndURI)
		require.ErrorIs(t, err, ErrNegativeTimeout)
	})

	t.Run("both code and uri", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "print('hello')",
			URI:     "file://script.risor",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBothCodeAndURI)
	})

	t.Run("uri only valid", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			URI:     "file://script.risor",
			Timeout: 5 * time.Second,
		}
		// This will fail at build stage due to invalid URI, but basic validation should pass
		err := evaluator.Validate()
		// build() will be called and will fail because the file doesn't exist
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
	})

	t.Run("compilation failure with invalid code", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "invalid risor syntax <<<",
			Timeout: 5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
	})
}

func TestRisorEvaluator_GetCompiledEvaluator(t *testing.T) {
	t.Run("build error propagated", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "invalid risor syntax <<<",
			Timeout: 5 * time.Second,
		}
		result, err := evaluator.GetCompiledEvaluator()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
		assert.Nil(t, result)
	})

	t.Run("successful build returns evaluator", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Code:    "1 + 1",
			Timeout: 5 * time.Second,
		}
		result, err := evaluator.GetCompiledEvaluator()
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestRisorEvaluator_GetTimeout(t *testing.T) {
	t.Run("returns set timeout", func(t *testing.T) {
		timeout := 10 * time.Second
		evaluator := &RisorEvaluator{
			Timeout: timeout,
		}
		assert.Equal(t, timeout, evaluator.GetTimeout())
	})

	t.Run("returns default when zero", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Timeout: 0,
		}
		assert.Equal(t, DefaultEvalTimeout, evaluator.GetTimeout())
	})

	t.Run("returns default when negative", func(t *testing.T) {
		evaluator := &RisorEvaluator{
			Timeout: -5 * time.Second,
		}
		assert.Equal(t, DefaultEvalTimeout, evaluator.GetTimeout())
	})
}
