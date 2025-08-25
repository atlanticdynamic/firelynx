package evaluators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	// Test error wrapping relationships
	require.ErrorIs(t, ErrInvalidEvaluatorType, ErrEvaluator)
	require.ErrorIs(t, ErrEmptyCode, ErrEvaluator)
	require.ErrorIs(t, ErrNegativeTimeout, ErrEvaluator)
	require.ErrorIs(t, ErrEmptyEntrypoint, ErrEvaluator)
}

func TestNewInvalidEvaluatorTypeError(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "integer value",
			value: 42,
			want:  "evaluator error: invalid evaluator type: 42",
		},
		{
			name:  "string value",
			value: "unknown",
			want:  "evaluator error: invalid evaluator type: unknown",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "evaluator error: invalid evaluator type: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewInvalidEvaluatorTypeError(tt.value)
			require.Error(t, err)
			assert.Equal(t, tt.want, err.Error())
			require.ErrorIs(t, err, ErrInvalidEvaluatorType)
		})
	}
}
