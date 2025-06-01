package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeConstants(t *testing.T) {
	// Test type constants
	assert.Equal(t, Type(""), Unknown)
	assert.Equal(t, Type("http_path"), TypeHTTP)
	assert.Equal(t, Type("mcp_resource"), TypeMCP)

	// Test string representation
	assert.Equal(t, "http_path", string(TypeHTTP))
}

func TestErrors(t *testing.T) {
	// Test error definitions
	assert.NotNil(t, ErrInvalidHTTPCondition)
	assert.NotNil(t, ErrEmptyValue)
	assert.NotNil(t, ErrInvalidConditionType)

	// Ensure errors have meaningful messages
	assert.Contains(t, ErrInvalidHTTPCondition.Error(), "HTTP")
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
