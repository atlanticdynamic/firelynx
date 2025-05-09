package evaluators

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEvaluatorType(t *testing.T) {
	tests := []struct {
		name    string
		typ     EvaluatorType
		wantErr bool
	}{
		{
			name:    "risor",
			typ:     EvaluatorTypeRisor,
			wantErr: false,
		},
		{
			name:    "starlark",
			typ:     EvaluatorTypeStarlark,
			wantErr: false,
		},
		{
			name:    "extism",
			typ:     EvaluatorTypeExtism,
			wantErr: false,
		},
		{
			name:    "unspecified",
			typ:     EvaluatorTypeUnspecified,
			wantErr: true,
		},
		{
			name:    "invalid",
			typ:     EvaluatorType(99),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEvaluatorType(tt.typ)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidEvaluatorType))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
