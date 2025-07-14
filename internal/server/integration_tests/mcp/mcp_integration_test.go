//go:build integration

package mcp

import (
	_ "embed"
	"strings"
	"testing"
	"text/template"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/mcp_builtin_tools.toml.tmpl
var mcpBuiltinToolsTemplate string

// MCPBuiltinToolsIntegrationTestSuite tests that MCP builtin tools are properly rejected
// since builtin handlers are not yet implemented
type MCPBuiltinToolsIntegrationTestSuite struct {
	suite.Suite
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TestBuiltinToolsRejected() {
	// Test that configurations with builtin tools are properly rejected
	// since builtin handlers are not yet implemented

	// Template variables
	templateVars := struct {
		Port int
	}{
		Port: 51127, // Use a fixed port for this test since we're not starting a server
	}

	// Render the configuration template
	tmpl, err := template.New("config").Parse(mcpBuiltinToolsTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered MCP config:\n%s", configData)

	// Attempt to load configuration - this should fail
	_, err = config.NewConfigFromBytes([]byte(configData))
	s.Require().Error(err, "Config loading should fail for builtin tools")

	// Verify the error mentions builtin handlers
	s.Contains(err.Error(), "builtin handlers are not yet implemented",
		"Error should mention that builtin handlers are not implemented")

	s.T().Logf("Expected error received: %v", err)
}

func TestMCPBuiltinToolsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MCPBuiltinToolsIntegrationTestSuite))
}
