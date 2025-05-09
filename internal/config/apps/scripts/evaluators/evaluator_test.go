package evaluators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluatorTypeConstants(t *testing.T) {
	// Verify the evaluator type constants
	assert.Equal(t, EvaluatorType(0), EvaluatorTypeUnspecified)
	assert.Equal(t, EvaluatorType(1), EvaluatorTypeRisor)
	assert.Equal(t, EvaluatorType(2), EvaluatorTypeStarlark)
	assert.Equal(t, EvaluatorType(3), EvaluatorTypeExtism)
}

func TestEvaluatorType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  EvaluatorType
		want string
	}{
		{
			name: "unspecified",
			typ:  EvaluatorTypeUnspecified,
			want: "Unspecified",
		},
		{
			name: "risor",
			typ:  EvaluatorTypeRisor,
			want: "Risor",
		},
		{
			name: "starlark",
			typ:  EvaluatorTypeStarlark,
			want: "Starlark",
		},
		{
			name: "extism",
			typ:  EvaluatorTypeExtism,
			want: "Extism",
		},
		{
			name: "unknown",
			typ:  EvaluatorType(99),
			want: "Unknown(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
