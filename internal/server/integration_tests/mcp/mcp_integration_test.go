//go:build integration

package mcp_test

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client/mcp"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/mcp_builtin_tools.toml.tmpl
var mcpBuiltinToolsTemplate string

// MCPBuiltinToolsIntegrationTestSuite tests MCP builtin tools via HTTP
type MCPBuiltinToolsIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
	mcpClient   mcp.Client
	mcpSession  mcp.Session
}

func (s *MCPBuiltinToolsIntegrationTestSuite) SetupSuite() {
	logging.SetupLogger("trace")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)

	// Template variables
	templateVars := struct {
		Port int
	}{
		Port: s.port,
	}

	// Render the configuration template
	tmpl, err := template.New("mcp_builtin_tools").Parse(mcpBuiltinToolsTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered MCP config:\n%s", configData)

	// Load and validate the configuration
	cfg, err := config.NewConfigFromBytes([]byte(configData))
	s.Require().NoError(err, "Failed to load config")
	s.Require().NoError(cfg.Validate(), "Config validation failed")

	// Create transaction storage
	txStore := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	s.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	s.httpRunner, err = httplistener.NewRunner()
	s.Require().NoError(err)

	// Register HTTP runner with orchestrator
	err = s.saga.RegisterParticipant(s.httpRunner)
	s.Require().NoError(err)

	// Start HTTP runner in background
	go func() {
		s.runnerErrCh <- s.httpRunner.Run(s.ctx)
	}()

	// Wait for HTTP runner to start
	s.Require().Eventually(func() bool {
		select {
		case err := <-s.runnerErrCh:
			s.T().Fatalf("HTTP runner failed to start: %v", err)
			return false
		default:
			return s.httpRunner.IsRunning()
		}
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	// Create a config transaction
	tx, err := transaction.FromTest(s.T().Name(), cfg, slog.Default().Handler())
	s.Require().NoError(err)

	// Validate the transaction
	err = tx.RunValidation()
	s.Require().NoError(err)

	// Process the transaction through the orchestrator
	err = s.saga.ProcessTransaction(s.ctx, tx)
	s.Require().NoError(err)

	// Verify the transaction completed successfully
	s.Require().Equal("completed", tx.GetState())

	// Wait for the server to be fully ready
	s.Require().Eventually(func() bool {
		// Try to connect with MCP client to verify server is ready
		mcpURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", s.port)
		transport := mcp.NewStreamableTransport(mcpURL, nil)

		// Create temporary client to test connectivity
		tempClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"})
		session, err := tempClient.Connect(s.ctx, transport)
		if err != nil {
			return false
		}
		session.Close()
		return true
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept MCP connections")

	// Create the MCP client for tests
	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", s.port)
	transport := mcp.NewStreamableTransport(mcpURL, nil)
	s.mcpClient = mcp.NewClient(&mcp.Implementation{Name: "integration-test-client", Version: "1.0.0"})

	// Establish the MCP session
	s.mcpSession, err = s.mcpClient.Connect(s.ctx, transport)
	s.Require().NoError(err, "Failed to establish MCP session")
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TearDownSuite() {
	// Close MCP session if it exists
	if s.mcpSession != nil {
		s.mcpSession.Close()
	}

	// Cancel context to signal shutdown
	if s.cancel != nil {
		s.cancel()
	}

	// Stop HTTP runner if it exists
	if s.httpRunner != nil {
		s.httpRunner.Stop()

		// Wait for runner to stop
		s.Require().Eventually(func() bool {
			return !s.httpRunner.IsRunning()
		}, time.Second, 10*time.Millisecond, "HTTP runner should stop")
	}
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TestEchoTool() {
	// Test that we can call the echo tool using the official MCP client
	// The session initialization is already handled by the MCP client in SetupSuite

	// Call echo tool
	result, err := s.mcpSession.CallTool(s.ctx, &mcp.CallToolParams{
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
	textContent, ok := result.Content[0].(*mcp.TextContent)
	s.Require().True(ok, "Echo tool should return text content")
	s.Contains(textContent.Text, "Hello, MCP!", "Echo tool should echo our message")

	s.T().Logf("Echo tool response: %s", textContent.Text)
}

func (s *MCPBuiltinToolsIntegrationTestSuite) TestListTools() {
	// Test that we can list available tools
	result, err := s.mcpSession.ListTools(s.ctx, &mcp.ListToolsParams{})
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
