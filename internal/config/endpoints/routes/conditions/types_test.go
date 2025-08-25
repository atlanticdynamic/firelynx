package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeConstants(t *testing.T) {
	// Test type constants
	assert.Equal(t, Unknown, Type(""))
	assert.Equal(t, TypeHTTP, Type("http_path"))
	assert.Equal(t, TypeMCP, Type("mcp_resource"))

	// Test string representation
	assert.Equal(t, "http_path", string(TypeHTTP))
}

func TestErrors(t *testing.T) {
	// Test error definitions
	require.Error(t, ErrInvalidHTTPCondition)
	require.Error(t, ErrEmptyValue)
	require.Error(t, ErrInvalidConditionType)

	// Ensure errors have meaningful messages
	assert.ErrorContains(t, ErrInvalidHTTPCondition, "HTTP")
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		condType Type
		expected string
	}{
		{TypeHTTP, "HTTP Path"},
		{TypeMCP, "MCP Resource"},
		{Unknown, "Unknown"},
		{Type("custom"), "Custom(custom)"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.condType), func(t *testing.T) {
			result := TypeString(tc.condType)
			assert.Equal(t, tc.expected, result)
		})
	}
}
