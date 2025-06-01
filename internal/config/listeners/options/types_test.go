package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestType_Values(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  Type
		expected string
	}{
		{
			name:     "Unknown type",
			typeVal:  Unknown,
			expected: "",
		},
		{
			name:     "HTTP type",
			typeVal:  TypeHTTP,
			expected: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Type(tt.expected), tt.typeVal)
		})
	}
}

func TestOptions_Interface(t *testing.T) {
	// Test that HTTPOptions implements Options interface
	var _ Options = HTTP{}
}
