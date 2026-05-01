//go:build integration

package mcp

import (
	_ "embed"
	"encoding/json"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/gateway_typed.toml.tmpl
var gatewayTypedTemplate string

// GatewayTypedSuite exercises the gateway path where an MCP server exposes
// built-in firelynx apps as typed MCP tools.
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
	s.Contains(names, "calculate", "expected calculation tool to be registered via app's MCPToolName")
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

func (s *GatewayTypedSuite) TestCallCalculationToolReturnsResult() {
	result, err := s.GetMCPSession().CallTool(s.GetContext(), &mcpsdk.CallToolParams{
		Name: "calculate",
		Arguments: map[string]any{
			"left":     6,
			"right":    2,
			"operator": "/",
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.False(result.IsError, "tool call should not error")

	s.Require().NotEmpty(result.Content)
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	s.Require().True(ok, "first content should be text")

	var got map[string]any
	s.Require().NoError(json.Unmarshal([]byte(text.Text), &got))
	s.InDelta(3, got["result"], 0.0001, "calculation tool should return result=3: %s", text.Text)
}

func TestGatewayTypedSuite(t *testing.T) {
	suite.Run(t, new(GatewayTypedSuite))
}
