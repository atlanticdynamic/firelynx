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
	"os"
	"path/filepath"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/script_risor_basic.toml.tmpl
var scriptRisorBasicTemplate string

//go:embed testdata/script_starlark_basic.toml.tmpl
var scriptStarlarkBasicTemplate string

//go:embed testdata/script_risor_file_uri.toml.tmpl
var scriptRisorFileURITemplate string

//go:embed testdata/script_starlark_file_uri.toml.tmpl
var scriptStarlarkFileURITemplate string

// ScriptResponse represents the expected response structure from our scripts
type ScriptResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// StarlarkResponse represents the response structure from Starlark scripts
type StarlarkResponse struct {
	Message       string `json:"message"`
	RequestMethod string `json:"request_method"`
	RequestPath   string `json:"request_path"`
	Language      string `json:"language"`
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

// StarlarkIntegrationTestSuite tests Starlark script execution via HTTP
type StarlarkIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
}

func (s *StarlarkIntegrationTestSuite) SetupSuite() {
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

	// Render the Starlark configuration template
	tmpl, err := template.New("script_starlark_basic").Parse(scriptStarlarkBasicTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered Starlark config:\n%s", configData)

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

func (s *StarlarkIntegrationTestSuite) TearDownSuite() {
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

func (s *StarlarkIntegrationTestSuite) TestStarlarkScriptBasicExecution() {
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

	var starlarkResp StarlarkResponse
	err = json.Unmarshal(body, &starlarkResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	s.Equal("Hello from Starlark!", starlarkResp.Message, "Script should return expected message")
	s.Equal("GET", starlarkResp.RequestMethod, "Script should return request method")
	s.Equal("/hello", starlarkResp.RequestPath, "Script should return request path")
	s.Equal("python-like", starlarkResp.Language, "Script should return language")

	s.T().Logf("Starlark response: %+v", starlarkResp)
}

func (s *StarlarkIntegrationTestSuite) TestStarlarkScriptPostExecution() {
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

	var starlarkResp StarlarkResponse
	err = json.Unmarshal(body, &starlarkResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content for POST
	s.Equal("Hello from Starlark!", starlarkResp.Message, "Script should return expected message")
	s.Equal("POST", starlarkResp.RequestMethod, "Script should return POST method")
	s.Equal("/hello", starlarkResp.RequestPath, "Script should return request path")
	s.Equal("python-like", starlarkResp.Language, "Script should return language")

	s.T().Logf("Starlark POST response: %+v", starlarkResp)
}

func TestStarlarkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StarlarkIntegrationTestSuite))
}

// ScriptErrorIntegrationTestSuite tests error scenarios for script execution via HTTP
type ScriptErrorIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
}

//go:embed testdata/script_timeout_test.toml.tmpl
var scriptTimeoutTemplate string

//go:embed testdata/script_syntax_error.toml.tmpl
var scriptSyntaxErrorTemplate string

func (s *ScriptErrorIntegrationTestSuite) SetupSuite() {
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

	// Render the timeout test configuration template
	tmpl, err := template.New("script_timeout_test").Parse(scriptTimeoutTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered timeout config:\n%s", configData)

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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/timeout", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusGatewayTimeout // We expect timeout for this endpoint
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready and timeout endpoint should timeout")
}

func (s *ScriptErrorIntegrationTestSuite) TearDownSuite() {
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

func (s *ScriptErrorIntegrationTestSuite) TestScriptTimeout() {
	// Make a GET request to the timeout endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/timeout", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer resp.Body.Close()

	// Verify timeout status code
	s.Equal(http.StatusGatewayTimeout, resp.StatusCode, "Script should timeout and return 504")

	// Read response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	// Verify error message
	s.Contains(string(body), "Script Execution Timeout", "Response should contain timeout message")

	s.T().Logf("Timeout response: %s", string(body))
}

func TestScriptErrorIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ScriptErrorIntegrationTestSuite))
}

// createTempScript creates a temporary script file and returns its path
func createTempScript(t *testing.T, filename, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, filename)
	err := os.WriteFile(scriptPath, []byte(content), 0o644)
	assert.Nil(t, err)
	return scriptPath
}

// FileURIResponse represents the response structure from file URI scripts
type FileURIResponse struct {
	Message  string `json:"message"`
	Source   string `json:"source"`
	Language string `json:"language,omitempty"`
}

// RisorFileURIIntegrationTestSuite tests Risor script execution from file:// URIs
type RisorFileURIIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
	scriptPath  string
}

func (s *RisorFileURIIntegrationTestSuite) SetupSuite() {
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)

	// Create script file in temp directory
	scriptContent := `// Example Risor script for URI loading test
{
    "message": "Hello from Risor file URI!",
    "source": "file://",
    "timestamp": time.now().format("2006-01-02T15:04:05Z07:00")
}`
	s.scriptPath = createTempScript(s.T(), "example_risor_script.risor", scriptContent)

	// Template variables
	templateVars := struct {
		Port       int
		ScriptPath string
	}{
		Port:       s.port,
		ScriptPath: s.scriptPath,
	}

	// Render the file URI configuration template
	tmpl, err := template.New("script_risor_file_uri").Parse(scriptRisorFileURITemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered Risor file URI config:\n%s", configData)

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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *RisorFileURIIntegrationTestSuite) TearDownSuite() {
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

func (s *RisorFileURIIntegrationTestSuite) TestRisorFileURIExecution() {
	// Make a GET request to the file script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", s.port))
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

	var fileResp FileURIResponse
	err = json.Unmarshal(body, &fileResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	s.Equal("Hello from Risor file URI!", fileResp.Message, "Script should return expected message")
	s.Equal("file://", fileResp.Source, "Script should indicate file source")

	s.T().Logf("Risor file URI response: %+v", fileResp)
}

func TestRisorFileURIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RisorFileURIIntegrationTestSuite))
}

// StarlarkFileURIIntegrationTestSuite tests Starlark script execution from file:// URIs
type StarlarkFileURIIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
	scriptPath  string
}

func (s *StarlarkFileURIIntegrationTestSuite) SetupSuite() {
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)

	// Create script file in temp directory
	scriptContent := `# Example Starlark script for URI loading test
result = {
    "message": "Hello from Starlark file URI!",
    "source": "file://",
    "language": "python-like"
}
# The underscore variable is returned to Go
_ = result`
	s.scriptPath = createTempScript(s.T(), "example_starlark_script.star", scriptContent)

	// Template variables
	templateVars := struct {
		Port       int
		ScriptPath string
	}{
		Port:       s.port,
		ScriptPath: s.scriptPath,
	}

	// Render the file URI configuration template
	tmpl, err := template.New("script_starlark_file_uri").Parse(scriptStarlarkFileURITemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered Starlark file URI config:\n%s", configData)

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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *StarlarkFileURIIntegrationTestSuite) TearDownSuite() {
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

func (s *StarlarkFileURIIntegrationTestSuite) TestStarlarkFileURIExecution() {
	// Make a GET request to the file script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", s.port))
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

	var fileResp FileURIResponse
	err = json.Unmarshal(body, &fileResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	s.Equal(
		"Hello from Starlark file URI!",
		fileResp.Message,
		"Script should return expected message",
	)
	s.Equal("file://", fileResp.Source, "Script should indicate file source")
	s.Equal("python-like", fileResp.Language, "Script should return language")

	s.T().Logf("Starlark file URI response: %+v", fileResp)
}

func TestStarlarkFileURIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StarlarkFileURIIntegrationTestSuite))
}
