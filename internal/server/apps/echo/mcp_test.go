package echo

import (
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEcho_MCPToolName(t *testing.T) {
	app := &App{}
	assert.Equal(t, "echo", app.MCPToolName())
}

func TestEcho_MCPToolDescription(t *testing.T) {
	app := &App{}
	assert.NotEmpty(t, app.MCPToolDescription())
}

func TestEcho_MCPToolOption_Registers(t *testing.T) {
	app := &App{id: "echo-app", response: "hi"}

	tests := []struct {
		name     string
		toolName string
	}{
		{name: "default name", toolName: app.MCPToolName()},
		{name: "user override", toolName: "renamed_echo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := app.MCPToolOption(tt.toolName)
			require.NotNil(t, opt, "MCPToolOption should not return nil")

			h, err := mcpio.NewHandler(opt, mcpio.WithName("test"))
			require.NoError(t, err, "handler should accept the tool option")
			require.NotNil(t, h)
		})
	}
}

func TestEcho_ToolFunc(t *testing.T) {
	app := &App{response: "hello"}
	out, err := app.echoToolFunc(t.Context(), nil, EchoInput{Message: "world"})
	require.NoError(t, err)
	assert.Equal(t, "hello: world", out.Result)
}
