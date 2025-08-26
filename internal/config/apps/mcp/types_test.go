package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp("test-app", "Test Server", "1.0.0")

	assert.NotNil(t, app)
	assert.Equal(t, "test-app", app.ID)
	assert.Equal(t, "Test Server", app.ServerName)
	assert.Equal(t, "1.0.0", app.ServerVersion)
	assert.NotNil(t, app.Transport)
	assert.NotNil(t, app.Tools)
	assert.NotNil(t, app.Resources)
	assert.NotNil(t, app.Prompts)
	assert.NotNil(t, app.Middlewares)
	assert.Empty(t, app.Tools)
	assert.Empty(t, app.Resources)
	assert.Empty(t, app.Prompts)
	assert.Empty(t, app.Middlewares)
	assert.Nil(t, app.compiledServer)
}

func TestApp_Type(t *testing.T) {
	app := &App{}
	assert.Equal(t, "mcp", app.Type())
}

func TestApp_GetCompiledServer(t *testing.T) {
	app := &App{}

	// Initially nil
	assert.Nil(t, app.GetCompiledServer())

	// After setting (this would normally be done during validation)
	// We can't easily create a real mcp.Server without dependencies,
	// so we test the nil case here and the successful case in validate_test.go
}

func TestScriptToolHandler_Type(t *testing.T) {
	handler := &ScriptToolHandler{}
	assert.Equal(t, "script", handler.Type())
}

func TestBuiltinToolHandler_Type(t *testing.T) {
	handler := &BuiltinToolHandler{}
	assert.Equal(t, "builtin", handler.Type())
}

func TestBuiltinType_String(t *testing.T) {
	tests := []struct {
		name     string
		builtin  BuiltinType
		expected string
	}{
		{"echo", BuiltinEcho, "ECHO"},
		{"calculation", BuiltinCalculation, "CALCULATION"},
		{"file_read", BuiltinFileRead, "FILE_READ"},
		{"unknown", BuiltinType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.builtin.String())
		})
	}
}

func TestMiddlewareType_String(t *testing.T) {
	tests := []struct {
		name       string
		middleware MiddlewareType
		expected   string
	}{
		{"rate_limiting", MiddlewareRateLimiting, "RATE_LIMITING"},
		{"logging", MiddlewareLogging, "MCP_LOGGING"},
		{"authentication", MiddlewareAuthentication, "MCP_AUTHENTICATION"},
		{"unknown", MiddlewareType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.middleware.String())
		})
	}
}

func TestBuiltinTypeConstants(t *testing.T) {
	// Test that the constants have expected values for enum consistency
	assert.Equal(t, BuiltinEcho, BuiltinType(0))
	assert.Equal(t, BuiltinCalculation, BuiltinType(1))
	assert.Equal(t, BuiltinFileRead, BuiltinType(2))
}

func TestMiddlewareTypeConstants(t *testing.T) {
	// Test that the constants have expected values for enum consistency
	assert.Equal(t, MiddlewareRateLimiting, MiddlewareType(0))
	assert.Equal(t, MiddlewareLogging, MiddlewareType(1))
	assert.Equal(t, MiddlewareAuthentication, MiddlewareType(2))
}
