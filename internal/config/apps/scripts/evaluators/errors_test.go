package evaluators

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	// Test error wrapping relationships
	assert.True(t, errors.Is(ErrInvalidEvaluatorType, ErrEvaluator))
	assert.True(t, errors.Is(ErrEmptyCode, ErrEvaluator))
	assert.True(t, errors.Is(ErrNegativeTimeout, ErrEvaluator))
	assert.True(t, errors.Is(ErrEmptyEntrypoint, ErrEvaluator))
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
			assert.Error(t, err)
			assert.Equal(t, tt.want, err.Error())
			assert.True(t, errors.Is(err, ErrInvalidEvaluatorType))
		})
	}
}
