package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeConstants(t *testing.T) {
	// Test type constants
	assert.Equal(t, Type(""), Unknown)
	assert.Equal(t, Type("http_path"), TypeHTTP)
	assert.Equal(t, Type("grpc_service"), TypeGRPC)
	assert.Equal(t, Type("mcp_resource"), TypeMCP)

	// Test string representation
	assert.Equal(t, "http_path", string(TypeHTTP))
	assert.Equal(t, "grpc_service", string(TypeGRPC))
}

func TestErrors(t *testing.T) {
	// Test error definitions
	assert.NotNil(t, ErrInvalidHTTPCondition)
	assert.NotNil(t, ErrInvalidGRPCCondition)
	assert.NotNil(t, ErrEmptyValue)
	assert.NotNil(t, ErrInvalidConditionType)

	// Ensure errors have meaningful messages
	assert.Contains(t, ErrInvalidHTTPCondition.Error(), "HTTP")
	assert.Contains(t, ErrInvalidGRPCCondition.Error(), "gRPC")
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		condType Type
		expected string
	}{
		{TypeHTTP, "HTTP Path"},
		{TypeGRPC, "gRPC Service"},
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
