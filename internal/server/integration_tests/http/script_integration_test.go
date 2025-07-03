//go:build integration

package http_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/script_risor_basic.toml.tmpl
var scriptRisorBasicTemplate string

// ScriptResponse represents the expected response structure from our scripts
type ScriptResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// ScriptIntegrationTestSuite tests end-to-end script execution via HTTP
type ScriptIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
}

func (s *ScriptIntegrationTestSuite) SetupSuite() {
	// Setup debug logging for better test debugging
	logging.SetupLogger("debug")

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
	tmpl, err := template.New("script_risor_basic").Parse(scriptRisorBasicTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered config:\n%s", configData)

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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/hello", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *ScriptIntegrationTestSuite) TearDownSuite() {
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

func (s *ScriptIntegrationTestSuite) TestRisorScriptBasicExecution() {
	// Make a GET request to the script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/hello", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer resp.Body.Close()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "Script should return 200 OK")

	// Verify content type
	s.Equal(
		"application/json",
		resp.Header.Get("Content-Type"),
		"Script should return JSON content type",
	)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var scriptResp ScriptResponse
	err = json.Unmarshal(body, &scriptResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	s.Equal("Hello from Risor!", scriptResp.Message, "Script should return expected message")
	s.NotEmpty(scriptResp.Timestamp, "Script should return timestamp")

	s.T().Logf("Script response: %+v", scriptResp)
}

func (s *ScriptIntegrationTestSuite) TestRisorScriptPostExecution() {
	// Make a POST request to the script endpoint
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/hello", s.port),
		"application/json",
		nil,
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer resp.Body.Close()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "Script should return 200 OK")

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var scriptResp ScriptResponse
	err = json.Unmarshal(body, &scriptResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content for POST
	s.Equal("Hello from Risor!", scriptResp.Message, "Script should return expected message")
	s.NotEmpty(scriptResp.Timestamp, "Script should return timestamp")

	s.T().Logf("Script POST response: %+v", scriptResp)
}

func TestScriptIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ScriptIntegrationTestSuite))
}
