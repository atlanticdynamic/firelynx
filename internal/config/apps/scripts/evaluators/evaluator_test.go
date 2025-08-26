package evaluators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluatorTypeConstants(t *testing.T) {
	// Verify the evaluator type constants
	assert.Equal(t, EvaluatorTypeUnspecified, EvaluatorType(0))
	assert.Equal(t, EvaluatorTypeRisor, EvaluatorType(1))
	assert.Equal(t, EvaluatorTypeStarlark, EvaluatorType(2))
	assert.Equal(t, EvaluatorTypeExtism, EvaluatorType(3))
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
