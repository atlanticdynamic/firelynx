package scripts

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	// Test error wrapping relationships
	assert.True(t, errors.Is(ErrMissingEvaluator, ErrAppScript))
	assert.True(t, errors.Is(ErrInvalidEvaluator, ErrAppScript))
	assert.True(t, errors.Is(ErrInvalidStaticData, ErrAppScript))
	assert.True(t, errors.Is(ErrProtoConversion, ErrAppScript))
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "Base error",
			err:  ErrAppScript,
			want: "app script error",
		},
		{
			name: "Missing evaluator",
			err:  ErrMissingEvaluator,
			want: "app script error: missing evaluator",
		},
		{
			name: "Invalid evaluator",
			err:  ErrInvalidEvaluator,
			want: "app script error: invalid evaluator",
		},
		{
			name: "Invalid static data",
			err:  ErrInvalidStaticData,
			want: "app script error: invalid static data",
		},
		{
			name: "Proto conversion",
			err:  ErrProtoConversion,
			want: "app script error: proto conversion error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
