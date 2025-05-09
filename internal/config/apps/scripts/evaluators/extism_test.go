package evaluators

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtismEvaluator_Type(t *testing.T) {
	extism := &ExtismEvaluator{}
	assert.Equal(t, EvaluatorTypeExtism, extism.Type())
}

func TestExtismEvaluator_String(t *testing.T) {
	tests := []struct {
		name      string
		evaluator *ExtismEvaluator
		want      string
	}{
		{
			name:      "nil",
			evaluator: nil,
			want:      "Extism(nil)",
		},
		{
			name:      "empty",
			evaluator: &ExtismEvaluator{},
			want:      "Extism(code=0 chars, entrypoint=)",
		},
		{
			name: "with code and entrypoint",
			evaluator: &ExtismEvaluator{
				Code:       "base64content",
				Entrypoint: "handle_request",
			},
			want: "Extism(code=13 chars, entrypoint=handle_request)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.evaluator.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtismEvaluator_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "base64content",
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.NoError(t, err)
	})

	t.Run("empty code", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "",
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyCode))
	})

	t.Run("empty entrypoint", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "base64content",
			Entrypoint: "",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyEntrypoint))
	})

	t.Run("multiple errors", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "",
			Entrypoint: "",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyCode))
		assert.True(t, errors.Is(err, ErrEmptyEntrypoint))
	})
}
