//go:build integration

package mcp

import (
	_ "embed"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/gateway_typed.toml.tmpl
var gatewayTypedTemplate string

// GatewayTypedSuite exercises the gateway path where an MCP server exposes
// a built-in firelynx app (echo) as a typed MCP tool.
type GatewayTypedSuite struct {
	MCPIntegrationTestSuite
}

func (s *GatewayTypedSuite) SetupSuite() {
	s.SetupSuiteWithTemplate(gatewayTypedTemplate)
}

func (s *GatewayTypedSuite) TestListToolsExposesEchoApp() {
	result, err := s.GetMCPSession().ListTools(s.GetContext(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	s.Contains(names, "echo", "expected echo tool to be registered via app's MCPToolName")
}

func (s *GatewayTypedSuite) TestCallEchoToolReturnsConfiguredResponse() {
	result, err := s.GetMCPSession().CallTool(s.GetContext(), &mcpsdk.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"message": "hello"},
	})
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.False(result.IsError, "tool call should not error")

	// Echo tool returns EchoOutput{Result: "<response>: <message>"}.
	// The structured result is mirrored as JSON in the first text content.
	s.Require().NotEmpty(result.Content)
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	s.Require().True(ok, "first content should be text")
	s.Contains(text.Text, "ack: hello")
}

func TestGatewayTypedSuite(t *testing.T) {
	suite.Run(t, new(GatewayTypedSuite))
}
