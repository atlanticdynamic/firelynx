//go:build integration

package mcp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	mcpapp "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/calculator.risor
var calculatorRisorScript []byte

//go:embed testdata/mcp_script_tools.toml.tmpl
var mcpScriptToolsTemplate string

// MCPScriptToolsIntegrationTestSuite tests MCP script-based tools via HTTP
type MCPScriptToolsIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
	mcpClient   *mcpsdk.Client
	mcpSession  *mcpsdk.ClientSession
}

func (s *MCPScriptToolsIntegrationTestSuite) SetupSuite() {
	logging.SetupLogger("trace")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)

	tempDir := s.T().TempDir()

	calculatorScriptPath := tempDir + "/calculator.risor"
	err := os.WriteFile(calculatorScriptPath, calculatorRisorScript, 0o644)
	s.Require().NoError(err, "Failed to write calculator script to temp directory")

	templateVars := struct {
		Port                 int
		CalculatorScriptPath string
	}{
		Port:                 s.port,
		CalculatorScriptPath: calculatorScriptPath,
	}

	// Render the script tools configuration template
	tmpl, err := template.New("mcp_script_tools").Parse(mcpScriptToolsTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered MCP script config:\n%s", configData)

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
		transport := &mcpsdk.StreamableClientTransport{
			Endpoint: mcpURL,
		}

		// Create temporary client to test connectivity
		tempClient := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
		session, err := tempClient.Connect(s.ctx, transport, nil)
		if err != nil {
			return false
		}
		session.Close() //nolint:errcheck // Session close errors are acceptable in readiness checks
		return true
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept MCP connections")

	// Create the MCP client for tests
	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", s.port)
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: mcpURL,
	}
	s.mcpClient = mcpsdk.NewClient(&mcpsdk.Implementation{Name: "script-test-client", Version: "1.0.0"}, nil)

	// Establish the MCP session
	s.mcpSession, err = s.mcpClient.Connect(s.ctx, transport, nil)
	s.Require().NoError(err, "Failed to establish MCP session")
}

func (s *MCPScriptToolsIntegrationTestSuite) TearDownSuite() {
	// Close MCP session if it exists
	if s.mcpSession != nil {
		s.mcpSession.Close() //nolint:errcheck // Session close errors are acceptable in teardown
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

func (s *MCPScriptToolsIntegrationTestSuite) TestCalculatorTool() {
	// Test script-based calculator tool
	result, err := s.mcpSession.CallTool(s.ctx, &mcpsdk.CallToolParams{
		Name: "calculator",
		Arguments: map[string]any{
			"expression": "2 + 2",
		},
	})
	s.Require().NoError(err, "Calculator tool call should succeed")
	s.Require().NotNil(result, "Calculator tool should return result")
	s.Require().False(result.IsError, "Calculator tool should not return error")
	s.Require().NotEmpty(result.Content, "Calculator tool should return content")

	// Verify the calculator content
	s.Require().Len(result.Content, 1, "Calculator tool should return exactly one content item")

	// Check that it's text content with actual calculation result
	textContent, ok := result.Content[0].(*mcpsdk.TextContent)
	s.Require().True(ok, "Calculator tool should return text content")
	s.Contains(textContent.Text, "Result: 4", "Calculator tool should return actual calculated result")

	s.T().Logf("Calculator tool response: %s", textContent.Text)
}

func (s *MCPScriptToolsIntegrationTestSuite) TestCalculatorToolError() {
	// Test script-based calculator tool with error case
	result, err := s.mcpSession.CallTool(s.ctx, &mcpsdk.CallToolParams{
		Name: "calculator",
		Arguments: map[string]any{
			"expression": "5 / 0",
		},
	})

	// Per MCP spec: tool errors should be returned as successful results with IsError=true
	// so the LLM can see the error and potentially self-correct
	s.Require().NoError(err, "Tool call should succeed at protocol level")
	s.Require().NotNil(result, "Result should not be nil")
	s.Require().True(result.IsError, "Result should indicate error")
	s.Require().NotEmpty(result.Content, "Error result should have content")

	// Verify error message is accessible to LLM
	textContent, ok := result.Content[0].(*mcpsdk.TextContent)
	s.Require().True(ok, "Error content should be text")
	s.Contains(textContent.Text, "Division by zero", "Error message should be visible to LLM")
}

func (s *MCPScriptToolsIntegrationTestSuite) TestListScriptTools() {
	// Test that we can list script tools
	result, err := s.mcpSession.ListTools(s.ctx, &mcpsdk.ListToolsParams{})
	s.Require().NoError(err, "ListTools should succeed")
	s.Require().NotNil(result, "ListTools should return result")
	s.Require().NotEmpty(result.Tools, "Should have tools available")

	// Verify we have our expected script tool
	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}

	s.Contains(toolNames, "calculator", "Should have calculator script tool")

	s.T().Logf("Available script tools: %v", toolNames)
}

func TestMCPScriptToolsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MCPScriptToolsIntegrationTestSuite))
}

func TestEnhancedCalculatorOperations(t *testing.T) {
	ctx := t.Context()

	// Create Risor evaluator using embedded script from calculator.risor
	risorEval := &evaluators.RisorEvaluator{
		Code:    string(calculatorRisorScript),
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	// Create script tool handler
	handler := &mcpapp.ScriptToolHandler{
		Evaluator:  risorEval,
		StaticData: &staticdata.StaticData{Data: map[string]any{}},
	}

	_, mcpHandler, err := handler.CreateMCPTool()
	require.NoError(t, err)

	testCases := []struct {
		name       string
		expression string
		wantError  bool
		wantResult string
	}{
		// Basic arithmetic operations
		{"Addition", "2 + 3", false, "Result: 5"},
		{"Subtraction", "10 - 4", false, "Result: 6"},
		{"Multiplication", "3 * 7", false, "Result: 21"},
		{"Division", "15 / 3", false, "Result: 5"},

		// Enhanced operations
		{"Power operation", "2 ^ 3", false, "Result: 8"},
		{"Modulo operation", "17 % 5", false, "Result: 2"},
		{"Square root", "sqrt(16)", false, "Result: 4"},

		// Floating point operations
		{"Float addition", "3.5 + 2.5", false, "Result: 6"},
		{"Float division", "7.5 / 2.5", false, "Result: 3"},

		// Error cases
		{"Division by zero", "10 / 0", true, "Division by zero"},
		{"Modulo by zero", "10 % 0", true, "Modulo by zero"},
		{"Negative square root", "sqrt(-4)", true, "Cannot calculate square root of a negative number"},
		{"Invalid expression", "invalid", true, "Unable to evaluate: invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(map[string]any{
				"expression": tc.expression,
			})
			require.NoError(t, err)
			params := &mcpsdk.CallToolParamsRaw{
				Arguments: json.RawMessage(args),
			}

			req := &mcpsdk.CallToolRequest{
				Params: params,
			}

			result, err := mcpHandler(ctx, req)
			require.NoError(t, err, "Tool call should succeed at protocol level")
			assert.NotNil(t, result, "Result should not be nil")

			if tc.wantError {
				assert.True(t, result.IsError, "Result should indicate error for: %s", tc.expression)
				assert.NotEmpty(t, result.Content, "Error result should have content")
				textContent, ok := result.Content[0].(*mcpsdk.TextContent)
				assert.True(t, ok, "Error content should be text")
				assert.Contains(t, textContent.Text, tc.wantResult, "Error message should match expected")
			} else {
				assert.False(t, result.IsError, "Result should not indicate error for: %s", tc.expression)
				assert.NotEmpty(t, result.Content, "Success result should have content")
				textContent, ok := result.Content[0].(*mcpsdk.TextContent)
				assert.True(t, ok, "Success content should be text")
				assert.Contains(t, textContent.Text, tc.wantResult, "Result should match expected for: %s", tc.expression)
			}
		})
	}
}
