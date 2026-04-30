//go:build integration

package mcp

import (
	_ "embed"
	"encoding/json"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/gateway_script.toml.tmpl
var gatewayScriptTemplate string

// GatewayScriptSuite exercises the gateway path where an MCP server exposes
// a Risor script app as a raw MCP tool with a user-supplied input_schema.
type GatewayScriptSuite struct {
	MCPIntegrationTestSuite
}

func (s *GatewayScriptSuite) SetupSuite() {
	s.SetupSuiteWithTemplate(gatewayScriptTemplate)
}

func (s *GatewayScriptSuite) TestListToolsUsesIDOverride() {
	result, err := s.GetMCPSession().ListTools(s.GetContext(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	s.Contains(names, "add", "expected the user-supplied tool ID, not the app_id")
	s.NotContains(names, "calc-script", "app_id should not leak as tool name when ID override is set")
}

func (s *GatewayScriptSuite) TestCallAddToolRunsScript() {
	result, err := s.GetMCPSession().CallTool(s.GetContext(), &mcpsdk.CallToolParams{
		Name:      "add",
		Arguments: map[string]any{"a": 2, "b": 3},
	})
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.False(result.IsError, "tool call should not error: content=%+v", result.Content)

	s.Require().NotEmpty(result.Content)
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	s.Require().True(ok, "first content should be text")

	var got map[string]any
	s.Require().NoError(json.Unmarshal([]byte(text.Text), &got))
	s.InDelta(5, got["sum"], 0.0001, "raw tool should return script's result JSON: %s", text.Text)
}

func TestGatewayScriptSuite(t *testing.T) {
	suite.Run(t, new(GatewayScriptSuite))
}
