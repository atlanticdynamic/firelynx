//go:build integration

package mcp

import (
	_ "embed"
	"testing"

	mcp_client "github.com/atlanticdynamic/firelynx/internal/client/mcp"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/mcp_builtin_tools.toml.tmpl
var mcpBuiltinToolsTemplate string

// MCPBuiltinToolsIntegrationTestSuite tests MCP builtin tools via HTTP
type MCPBuiltinToolsIntegrationTestSuite struct {
	MCPIntegrationTestSuite
}

func (s *MCPBuiltinToolsIntegrationTestSuite) SetupSuite() {
	// Use the base suite's template setup
	s.SetupSuiteWithTemplate(mcpBuiltinToolsTemplate)
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TestEchoTool() {
	// Test that we can call the echo tool using the official MCP client
	// Call echo tool
	result, err := s.GetMCPSession().CallTool(s.GetContext(), &mcp_client.CallToolParams{
		Name: "echo",
		Arguments: map[string]any{
			"message": "Hello, MCP!",
		},
	})
	s.Require().NoError(err, "Echo tool call should succeed")
	s.Require().NotNil(result, "Echo tool should return result")
	s.Require().False(result.IsError, "Echo tool should not return error")
	s.Require().NotEmpty(result.Content, "Echo tool should return content")

	// Verify the echo content
	s.Require().Len(result.Content, 1, "Echo tool should return exactly one content item")

	// Check that it's text content with our message
	textContent, ok := result.Content[0].(*mcp_client.TextContent)
	s.Require().True(ok, "Echo tool should return text content")
	s.Contains(textContent.Text, "Hello, MCP!", "Echo tool should echo our message")

	s.T().Logf("Echo tool response: %s", textContent.Text)
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TestListTools() {
	// Test that we can list available tools
	result, err := s.GetMCPSession().ListTools(s.GetContext(), &mcp_client.ListToolsParams{})
	s.Require().NoError(err, "ListTools should succeed")
	s.Require().NotNil(result, "ListTools should return result")
	s.Require().NotEmpty(result.Tools, "Should have tools available")

	// Verify we have our expected tools
	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}

	s.Contains(toolNames, "echo", "Should have echo tool")
	s.Contains(toolNames, "read_file", "Should have read_file tool")

	s.T().Logf("Available tools: %v", toolNames)
}

func TestMCPBuiltinToolsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MCPBuiltinToolsIntegrationTestSuite))
}
