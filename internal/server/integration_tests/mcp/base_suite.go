package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

// MCPIntegrationTestSuite is a base test suite for MCP integration tests
type MCPIntegrationTestSuite struct {
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

// SetupSuiteWithConfig sets up the test suite with a given configuration
func (s *MCPIntegrationTestSuite) SetupSuiteWithConfig(cfg *config.Config) {
	s.initializeTestEnvironment()
	s.startServerWithConfig(cfg)
	s.establishMCPConnection()
}

// initializeTestEnvironment sets up the basic test environment
func (s *MCPIntegrationTestSuite) initializeTestEnvironment() {
	logging.SetupLogger("trace")
	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)
}

// startServerWithConfig starts the firelynx server with the given configuration
func (s *MCPIntegrationTestSuite) startServerWithConfig(cfg *config.Config) {
	// Update port in configuration to avoid conflicts
	s.updateConfigPort(cfg)

	// Validate the configuration
	s.Require().NoError(cfg.Validate(), "Config validation failed")

	// Create transaction storage
	txStore := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	s.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	var err error
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
}

// establishMCPConnection creates and establishes the MCP client connection
func (s *MCPIntegrationTestSuite) establishMCPConnection() {
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
		s.NoError(session.Close())
		return true
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept MCP connections")

	// Create the MCP client for tests
	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", s.port)
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: mcpURL,
	}
	s.mcpClient = mcpsdk.NewClient(&mcpsdk.Implementation{Name: "integration-test-client", Version: "1.0.0"}, nil)

	// Establish the MCP session
	var err error
	s.mcpSession, err = s.mcpClient.Connect(s.ctx, transport, nil)
	s.Require().NoError(err, "Failed to establish MCP session")
}

// SetupSuiteWithTemplate sets up the test suite with a template configuration
func (s *MCPIntegrationTestSuite) SetupSuiteWithTemplate(templateContent string) {
	s.initializeTestEnvironment()

	// Template variables with the port we just got
	templateVars := struct {
		Port int
	}{
		Port: s.port,
	}

	// Render the configuration template
	tmpl, err := template.New("config").Parse(templateContent)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered MCP config:\n%s", configData)

	// Load configuration from rendered template
	cfg, err := config.NewConfigFromBytes([]byte(configData))
	s.Require().NoError(err, "Failed to load config")

	s.startServerWithConfig(cfg)
	s.establishMCPConnection()
}

// SetupSuiteWithFile sets up the test suite with a configuration file
func (s *MCPIntegrationTestSuite) SetupSuiteWithFile(configFile string) {
	s.initializeTestEnvironment()

	// Load the configuration file
	cfg, err := config.NewConfig(configFile)
	s.Require().NoError(err, "Failed to load config file: %s", configFile)

	s.startServerWithConfig(cfg)
	s.establishMCPConnection()
}

// TearDownSuite tears down the test suite
func (s *MCPIntegrationTestSuite) TearDownSuite() {
	// Close MCP session BEFORE stopping the server to avoid connection reset
	if s.mcpSession != nil {
		// Use NoError but don't fail the test if MCP session close fails
		// This handles cases where the server is already shut down
		if err := s.mcpSession.Close(); err != nil {
			s.T().Logf("MCP session close error (may be expected during shutdown): %v", err)
		}
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

// updateConfigPort modifies the loaded config to use the test port
func (s *MCPIntegrationTestSuite) updateConfigPort(cfg *config.Config) {
	for i := range cfg.Listeners {
		listener := &cfg.Listeners[i]
		// Check if this is an HTTP listener
		if listener.Type == listeners.TypeHTTP {
			listener.Address = fmt.Sprintf(":%d", s.port)
		}
	}
}

// GetMCPSession returns the MCP session for tests to use
func (s *MCPIntegrationTestSuite) GetMCPSession() *mcpsdk.ClientSession {
	return s.mcpSession
}

// GetContext returns the test context
func (s *MCPIntegrationTestSuite) GetContext() context.Context {
	return s.ctx
}

// GetPort returns the test port
func (s *MCPIntegrationTestSuite) GetPort() int {
	return s.port
}

// ValidateEmbeddedConfig validates that the embedded config can be loaded and passes validation
func (s *MCPIntegrationTestSuite) ValidateEmbeddedConfig(configBytes []byte) *config.Config {
	// Load configuration from embedded bytes
	cfg, err := config.NewConfigFromBytes(configBytes)
	s.Require().NoError(err, "Should load config from embedded bytes")
	s.Require().NotNil(cfg, "Config should not be nil")

	// Validate configuration
	err = cfg.Validate()
	s.Require().NoError(err, "Config should validate successfully")

	s.T().Logf("Embedded config loaded and validated successfully")
	return cfg
}
