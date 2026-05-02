package mcpserver

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/calculation"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/fileread"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/script"
	"github.com/stretchr/testify/assert"
)

// TestApps_SatisfyMCPTypedToolProvider verifies that first-class typed apps
// satisfy the provider contract structurally. Apps live in sibling packages
// and do not import this one, so this test guards the consumer-defined contract.
func TestApps_SatisfyMCPTypedToolProvider(t *testing.T) {
	cases := []struct {
		name string
		app  any
	}{
		{name: "echo", app: &echo.App{}},
		{name: "calculation", app: &calculation.App{}},
		{name: "fileread", app: &fileread.App{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := tc.app.(MCPTypedToolProvider)
			assert.True(t, ok, "%T must implement MCPTypedToolProvider", tc.app)
		})
	}
}

// TestScriptApp_SatisfiesMCPRawToolProvider guards the contract that allows
// Risor/Starlark/Extism script apps to back MCP tools through the raw path.
func TestScriptApp_SatisfiesMCPRawToolProvider(t *testing.T) {
	var _ MCPRawToolProvider = (*script.ScriptApp)(nil)
}
