//go:build integration

package http_test

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/require"
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

//go:embed testdata/script_extism_basic.toml.tmpl
var scriptExtismBasicTemplate string

//go:embed testdata/script_risor_https.toml.tmpl
var scriptRisorHTTPSTemplate string

//go:embed testdata/script_starlark_https.toml.tmpl
var scriptStarlarkHTTPSTemplate string

//go:embed testdata/script_extism_https.toml.tmpl
var scriptExtismHTTPSTemplate string

// ScriptResponse represents the expected response structure from our scripts
type ScriptResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source,omitempty"`
	Evaluator string `json:"evaluator,omitempty"`
}

// StarlarkResponse represents the response structure from Starlark scripts
type StarlarkResponse struct {
	Message       string `json:"message"`
	RequestMethod string `json:"requestMethod"`
	RequestPath   string `json:"requestPath"`
	Language      string `json:"language"`
}

// scriptSuiteFields represents common fields across all script test suites
type scriptSuiteFields struct {
	ctx          context.Context
	cancel       context.CancelFunc
	port         int
	httpRunner   *httplistener.Runner
	saga         *orchestrator.SagaOrchestrator
	runnerErrCh  chan error
	scriptPath   string           // Optional: used by file:// URI test suites
	scriptServer *httptest.Server // Optional: used by HTTPS test suites
}

// setupScriptSuite is a helper function to reduce code duplication across script test suites
func setupScriptSuite(t *testing.T, templateName, templateContent string, fields *scriptSuiteFields) {
	t.Helper()
	setupScriptSuiteWithEndpoint(t, templateName, templateContent, fields, "/hello", http.StatusOK, "Server should be ready to accept requests")
}

// setupScriptSuiteWithEndpoint is a helper function that allows customizing the readiness endpoint and expected status
func setupScriptSuiteWithEndpoint(t *testing.T, templateName, templateContent string, fields *scriptSuiteFields, endpoint string, expectedStatus int, readyMessage string) {
	t.Helper()
	logging.SetupLogger("debug")

	fields.ctx, fields.cancel = context.WithCancel(t.Context())
	fields.port = testutil.GetRandomPort(t)
	fields.runnerErrCh = make(chan error, 1)

	// Template variables
	templateVars := struct {
		Port int
	}{
		Port: fields.port,
	}

	// Render the configuration template
	tmpl, err := template.New(templateName).Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	require.NoError(t, err, "Failed to render config template")

	configData := configBuffer.String()
	t.Logf("Rendered config:\n%s", configData)

	// Load and validate the configuration
	cfg, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err, "Failed to load config")
	require.NoError(t, cfg.Validate(), "Config validation failed")

	// Create transaction storage
	txStore := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	fields.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	fields.httpRunner, err = httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = fields.saga.RegisterParticipant(fields.httpRunner)
	require.NoError(t, err)

	// Start HTTP runner in background
	go func() {
		fields.runnerErrCh <- fields.httpRunner.Run(fields.ctx)
	}()

	// Wait for HTTP runner to start
	require.Eventually(t, func() bool {
		select {
		case err := <-fields.runnerErrCh:
			t.Fatalf("HTTP runner failed to start: %v", err)
			return false
		default:
			return fields.httpRunner.IsRunning()
		}
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	// Create a config transaction
	tx, err := transaction.FromTest(t.Name(), cfg, slog.Default().Handler())
	require.NoError(t, err)

	// Validate the transaction
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process the transaction through the orchestrator
	err = fields.saga.ProcessTransaction(fields.ctx, tx)
	require.NoError(t, err)

	// Verify the transaction completed successfully
	require.Equal(t, "completed", tx.GetState())

	// Wait for the server to be fully ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", fields.port, endpoint))
		if err != nil {
			return false
		}
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode == expectedStatus
	}, 10*time.Second, 100*time.Millisecond, readyMessage)
}

// setupScriptSuiteWithFile is a helper for file:// URI test suites
func setupScriptSuiteWithFile(t *testing.T, templateName, templateContent, scriptFilename, scriptContent string, fields *scriptSuiteFields) {
	t.Helper()
	logging.SetupLogger("debug")

	fields.ctx, fields.cancel = context.WithCancel(t.Context())
	fields.port = testutil.GetRandomPort(t)
	fields.runnerErrCh = make(chan error, 1)

	// Create script file in temp directory
	fields.scriptPath = createTempScript(t, scriptFilename, scriptContent)

	// Template variables for file URI templates
	templateVars := struct {
		Port       int
		ScriptPath string
	}{
		Port:       fields.port,
		ScriptPath: fields.scriptPath,
	}

	// Render the configuration template
	tmpl, err := template.New(templateName).Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	require.NoError(t, err, "Failed to render config template")

	configData := configBuffer.String()
	t.Logf("Rendered config:\n%s", configData)

	// Load and validate the configuration
	cfg, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err, "Failed to load config")
	require.NoError(t, cfg.Validate(), "Config validation failed")

	// Create transaction storage
	txStore := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	fields.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	fields.httpRunner, err = httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = fields.saga.RegisterParticipant(fields.httpRunner)
	require.NoError(t, err)

	// Start HTTP runner in background
	go func() {
		fields.runnerErrCh <- fields.httpRunner.Run(fields.ctx)
	}()

	// Wait for HTTP runner to start
	require.Eventually(t, func() bool {
		select {
		case err := <-fields.runnerErrCh:
			t.Fatalf("HTTP runner failed to start: %v", err)
			return false
		default:
			return fields.httpRunner.IsRunning()
		}
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	// Create a config transaction
	tx, err := transaction.FromTest(t.Name(), cfg, slog.Default().Handler())
	require.NoError(t, err)

	// Validate the transaction
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process the transaction through the orchestrator
	err = fields.saga.ProcessTransaction(fields.ctx, tx)
	require.NoError(t, err)

	// Verify the transaction completed successfully
	require.Equal(t, "completed", tx.GetState())

	// Wait for the server to be fully ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", fields.port))
		if err != nil {
			return false
		}
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

// setupScriptSuiteWithHTTPS is a helper for HTTPS test suites
func setupScriptSuiteWithHTTPS(t *testing.T, templateName, templateContent, scriptPath string, fields *scriptSuiteFields) {
	t.Helper()
	logging.SetupLogger("debug")

	fields.ctx, fields.cancel = context.WithCancel(t.Context())
	fields.port = testutil.GetRandomPort(t)
	fields.runnerErrCh = make(chan error, 1)

	// Set up HTTP test server for serving test scripts
	fields.scriptServer = setupHTTPTestServer()
	t.Logf("Test script server running at: %s", fields.scriptServer.URL)

	// Template variables with script server URL
	templateVars := struct {
		Port      int
		ScriptURL string
	}{
		Port:      fields.port,
		ScriptURL: fields.scriptServer.URL + scriptPath,
	}

	// Render the configuration template
	tmpl, err := template.New(templateName).Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	require.NoError(t, err, "Failed to render config template")

	configData := configBuffer.String()
	t.Logf("Rendered HTTPS config:\n%s", configData)

	// Load and validate the configuration
	cfg, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err, "Failed to load config")
	require.NoError(t, cfg.Validate(), "Config validation failed")

	// Create transaction storage
	txStore := txstorage.NewMemoryStorage()

	// Create saga orchestrator
	fields.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	fields.httpRunner, err = httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = fields.saga.RegisterParticipant(fields.httpRunner)
	require.NoError(t, err)

	// Start HTTP runner in background
	go func() {
		fields.runnerErrCh <- fields.httpRunner.Run(fields.ctx)
	}()

	// Wait for HTTP runner to start
	require.Eventually(t, func() bool {
		select {
		case err := <-fields.runnerErrCh:
			t.Fatalf("HTTP runner failed to start: %v", err)
			return false
		default:
			return fields.httpRunner.IsRunning()
		}
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	// Create a config transaction
	tx, err := transaction.FromTest(t.Name(), cfg, slog.Default().Handler())
	require.NoError(t, err)

	// Validate the transaction
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process the transaction through the orchestrator
	err = fields.saga.ProcessTransaction(fields.ctx, tx)
	require.NoError(t, err)

	// Verify the transaction completed successfully
	require.Equal(t, "completed", tx.GetState())

	// Wait for the server to be fully ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/execute", fields.port))
		if err != nil {
			return false
		}
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

// teardownScriptSuite is a helper function for common teardown logic
func teardownScriptSuite(t *testing.T, fields *scriptSuiteFields) {
	t.Helper()
	// Close script server if it exists
	if fields.scriptServer != nil {
		fields.scriptServer.Close()
	}

	// Cancel context to signal shutdown
	if fields.cancel != nil {
		fields.cancel()
	}

	// Stop HTTP runner if it exists
	if fields.httpRunner != nil {
		fields.httpRunner.Stop()

		// Wait for runner to stop
		require.Eventually(t, func() bool {
			return !fields.httpRunner.IsRunning()
		}, time.Second, 10*time.Millisecond, "HTTP runner should stop")
	}
}

// ScriptIntegrationTestSuite tests end-to-end script execution via HTTP
type ScriptIntegrationTestSuite struct {
	suite.Suite
	scriptSuiteFields
}

func (s *ScriptIntegrationTestSuite) SetupSuite() {
	setupScriptSuite(s.T(), "script_risor_basic", scriptRisorBasicTemplate, &s.scriptSuiteFields)
}

func (s *ScriptIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *ScriptIntegrationTestSuite) TestRisorScriptBasicExecution() {
	// Make a GET request to the script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/hello", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer func() { s.NoError(resp.Body.Close()) }()

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
	defer func() { s.NoError(resp.Body.Close()) }()

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
	scriptSuiteFields
}

func (s *StarlarkIntegrationTestSuite) SetupSuite() {
	setupScriptSuite(s.T(), "script_starlark_basic", scriptStarlarkBasicTemplate, &s.scriptSuiteFields)
}

func (s *StarlarkIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *StarlarkIntegrationTestSuite) TestStarlarkScriptBasicExecution() {
	// Make a GET request to the script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/hello", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer func() { s.NoError(resp.Body.Close()) }()

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
	defer func() { s.NoError(resp.Body.Close()) }()

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
	scriptSuiteFields
}

//go:embed testdata/script_timeout_test.toml.tmpl
var scriptTimeoutTemplate string

func (s *ScriptErrorIntegrationTestSuite) SetupSuite() {
	setupScriptSuiteWithEndpoint(s.T(), "script_timeout_test", scriptTimeoutTemplate, &s.scriptSuiteFields, "/timeout", http.StatusGatewayTimeout, "Server should be ready and timeout endpoint should timeout")
}

func (s *ScriptErrorIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *ScriptErrorIntegrationTestSuite) TestScriptTimeout() {
	// Make a GET request to the timeout endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/timeout", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer func() { s.NoError(resp.Body.Close()) }()

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
	require.NoError(t, err)
	return scriptPath
}

// FileURIResponse represents the response structure from file URI scripts
type FileURIResponse struct {
	Message  string `json:"message"`
	Source   string `json:"source"`
	Language string `json:"language,omitempty"`
}

// ExtismGreetResponse represents the response from Extism greet entrypoint
type ExtismGreetResponse struct {
	Greeting string `json:"greeting"`
}

// ExtismCountResponse represents the response from Extism count_vowels entrypoint
type ExtismCountResponse struct {
	Count  int    `json:"count"`
	Vowels string `json:"vowels"`
	Input  string `json:"input"`
}

// ExtismReverseResponse represents the response from Extism reverse_string entrypoint
type ExtismReverseResponse struct {
	Reversed string `json:"reversed"`
}

// RisorFileURIIntegrationTestSuite tests Risor script execution from file:// URIs
type RisorFileURIIntegrationTestSuite struct {
	suite.Suite
	scriptSuiteFields
}

func (s *RisorFileURIIntegrationTestSuite) SetupSuite() {
	scriptContent := `// Example Risor script for URI loading test
{
    "message": "Hello from Risor file URI!",
    "source": "file://",
    "timestamp": time.now().format("2006-01-02T15:04:05Z07:00")
}`
	setupScriptSuiteWithFile(s.T(), "script_risor_file_uri", scriptRisorFileURITemplate, "example_risor_script.risor", scriptContent, &s.scriptSuiteFields)
}

func (s *RisorFileURIIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *RisorFileURIIntegrationTestSuite) TestRisorFileURIExecution() {
	// Make a GET request to the file script endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/file-script", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer func() { s.NoError(resp.Body.Close()) }()

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
		s.NoError(resp.Body.Close())
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
	defer func() { s.NoError(resp.Body.Close()) }()

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

// ExtismIntegrationTestSuite tests Extism WASM script execution via HTTP
type ExtismIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
}

func (s *ExtismIntegrationTestSuite) SetupSuite() {
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)

	// Template variables with base64-encoded WASM
	templateVars := struct {
		Port       int
		WasmBase64 string
	}{
		Port:       s.port,
		WasmBase64: base64.StdEncoding.EncodeToString(wasmdata.TestModule),
	}

	// Render the Extism configuration template
	tmpl, err := template.New("script_extism_basic").Parse(scriptExtismBasicTemplate)
	s.Require().NoError(err, "Failed to parse template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	// s.T().Logf("Rendered Extism config:\n%s", configData)

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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/greet", s.port))
		if err != nil {
			return false
		}
		s.NoError(resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *ExtismIntegrationTestSuite) TearDownSuite() {
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

func (s *ExtismIntegrationTestSuite) TestExtismGreetExecution() {
	// Make a POST request with JSON input to the greet endpoint
	reqBody := `{"input": "integration test"}`
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/greet", s.port),
		"application/json",
		strings.NewReader(reqBody),
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer func() { s.NoError(resp.Body.Close()) }()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "WASM script should return 200 OK")

	// Verify content type
	s.Equal(
		"application/json",
		resp.Header.Get("Content-Type"),
		"WASM script should return JSON content type",
	)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var greetResp ExtismGreetResponse
	err = json.Unmarshal(body, &greetResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify WASM response content
	s.Equal("Hello, integration test!", greetResp.Greeting, "WASM should return expected greeting")

	s.T().Logf("Extism greet response: %+v", greetResp)
}

func (s *ExtismIntegrationTestSuite) TestExtismCountVowelsExecution() {
	// Make a POST request with JSON input to the count endpoint
	reqBody := `{"input": "hello world"}`
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/count", s.port),
		"application/json",
		strings.NewReader(reqBody),
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer func() { s.NoError(resp.Body.Close()) }()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "WASM script should return 200 OK")

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var countResp ExtismCountResponse
	err = json.Unmarshal(body, &countResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify WASM response content - "integration test" has 6 vowels (i, e, a, i, o, e)
	s.Equal(6, countResp.Count, "WASM should count vowels correctly")
	s.Equal("integration test", countResp.Input, "WASM should echo input")

	s.T().Logf("Extism count vowels response: %+v", countResp)
}

func (s *ExtismIntegrationTestSuite) TestExtismReverseStringExecution() {
	// Make a POST request with JSON input to the reverse endpoint
	reqBody := `{"input": "extism"}`
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/reverse", s.port),
		"application/json",
		strings.NewReader(reqBody),
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer func() { s.NoError(resp.Body.Close()) }()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "WASM script should return 200 OK")

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var reverseResp ExtismReverseResponse
	err = json.Unmarshal(body, &reverseResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify WASM response content - "integration test" reversed is "tset noitargetni"
	s.Equal("tset noitargetni", reverseResp.Reversed, "WASM should reverse string correctly")

	s.T().Logf("Extism reverse string response: %+v", reverseResp)
}

func TestExtismIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ExtismIntegrationTestSuite))
}

// setupHTTPTestServer creates an HTTP server for serving test scripts
func setupHTTPTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Serve Risor test script
	mux.HandleFunc("/scripts/test.risor", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		script := `// HTTPS-loaded Risor script
{
  "message": "Hello from HTTPS Risor!",
  "source": "https",
  "evaluator": "risor",
  "timestamp": time.now().format("2006-01-02T15:04:05Z07:00")
}`
		if _, err := w.Write([]byte(script)); err != nil {
			panic(err) // Test server write should not fail
		}
	})

	// Serve Starlark test script
	mux.HandleFunc("/scripts/test.star", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		script := `# HTTPS-loaded Starlark script
result = {
    "message": "Hello from HTTPS Starlark!",
    "source": "https", 
    "evaluator": "starlark",
    "timestamp": "2025-07-03T21:52:02-04:00"
}
# The underscore variable is returned to Go
_ = result`
		if _, err := w.Write([]byte(script)); err != nil {
			panic(err) // Test server write should not fail
		}
	})

	// Serve WASM test module (base64 encoded)
	mux.HandleFunc("/scripts/test.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		if _, err := w.Write(wasmdata.TestModule); err != nil {
			panic(err) // Test server write should not fail
		}
	})

	return httptest.NewServer(mux)
}

// RisorHTTPSIntegrationTestSuite tests Risor script execution from HTTPS URIs
type RisorHTTPSIntegrationTestSuite struct {
	suite.Suite
	scriptSuiteFields
}

func (s *RisorHTTPSIntegrationTestSuite) SetupSuite() {
	setupScriptSuiteWithHTTPS(s.T(), "script_risor_https", scriptRisorHTTPSTemplate, "/scripts/test.risor", &s.scriptSuiteFields)
}

func (s *RisorHTTPSIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

// testHTTPSScriptExecution is a helper function to test HTTPS script execution
// It reduces code duplication between different script type test suites
func testHTTPSScriptExecution(suite interface {
	T() *testing.T
	Require() *require.Assertions
	Equal(interface{}, interface{}, ...interface{}) bool
	NotEmpty(interface{}, ...interface{}) bool
	NoError(err error, msgAndArgs ...interface{}) bool
}, port int, expectedMessage, expectedEvaluator string,
) {
	// Make a POST request with JSON input to test HTTPS script loading
	reqBody := `{"name": "HTTPS Test"}`
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/execute", port),
		"application/json",
		strings.NewReader(reqBody),
	)
	suite.Require().NoError(err, "Failed to make POST request")
	defer func() { suite.NoError(resp.Body.Close()) }()

	// Verify status code
	suite.Equal(http.StatusOK, resp.StatusCode, "HTTPS script should return 200 OK")

	// Verify content type
	suite.Equal(
		"application/json",
		resp.Header.Get("Content-Type"),
		"HTTPS script should return JSON content type",
	)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err, "Failed to read response body")

	var scriptResp ScriptResponse
	err = json.Unmarshal(body, &scriptResp)
	suite.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	suite.Equal(
		expectedMessage,
		scriptResp.Message,
		"HTTPS script should return expected message",
	)
	suite.Equal("https", scriptResp.Source, "Script should indicate HTTPS source")
	suite.Equal(expectedEvaluator, scriptResp.Evaluator, fmt.Sprintf("Script should indicate %s evaluator", expectedEvaluator))
	suite.NotEmpty(scriptResp.Timestamp, "Script should include timestamp")

	suite.T().Logf("HTTPS %s response: %+v", expectedEvaluator, scriptResp)
}

func (s *RisorHTTPSIntegrationTestSuite) TestRisorHTTPSExecution() {
	testHTTPSScriptExecution(s, s.port, "Hello from HTTPS Risor!", "risor")
}

func TestRisorHTTPSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RisorHTTPSIntegrationTestSuite))
}

// StarlarkHTTPSIntegrationTestSuite tests Starlark script execution from HTTPS URIs
type StarlarkHTTPSIntegrationTestSuite struct {
	suite.Suite
	scriptSuiteFields
}

func (s *StarlarkHTTPSIntegrationTestSuite) SetupSuite() {
	setupScriptSuiteWithHTTPS(s.T(), "script_starlark_https", scriptStarlarkHTTPSTemplate, "/scripts/test.star", &s.scriptSuiteFields)
}

func (s *StarlarkHTTPSIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *StarlarkHTTPSIntegrationTestSuite) TestStarlarkHTTPSExecution() {
	testHTTPSScriptExecution(s, s.port, "Hello from HTTPS Starlark!", "starlark")
}

func TestStarlarkHTTPSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StarlarkHTTPSIntegrationTestSuite))
}

// ExtismHTTPSIntegrationTestSuite tests Extism/WASM script execution from HTTPS URIs
type ExtismHTTPSIntegrationTestSuite struct {
	suite.Suite
	scriptSuiteFields
}

func (s *ExtismHTTPSIntegrationTestSuite) SetupSuite() {
	setupScriptSuiteWithHTTPS(s.T(), "script_extism_https", scriptExtismHTTPSTemplate, "/scripts/test.wasm", &s.scriptSuiteFields)
}

func (s *ExtismHTTPSIntegrationTestSuite) TearDownSuite() {
	teardownScriptSuite(s.T(), &s.scriptSuiteFields)
}

func (s *ExtismHTTPSIntegrationTestSuite) TestExtismHTTPSExecution() {
	// Make a POST request with JSON input to test HTTPS WASM script loading
	reqBody := `{"input": "HTTPS Test"}`
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/execute", s.port),
		"application/json",
		strings.NewReader(reqBody),
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer func() { s.NoError(resp.Body.Close()) }()

	// Verify status code
	s.Equal(http.StatusOK, resp.StatusCode, "HTTPS WASM script should return 200 OK")

	// Verify content type
	s.Equal(
		"application/json",
		resp.Header.Get("Content-Type"),
		"HTTPS WASM script should return JSON content type",
	)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var scriptResp map[string]interface{}
	err = json.Unmarshal(body, &scriptResp)
	s.Require().NoError(err, "Failed to parse JSON response")

	// Verify script response content
	s.Equal(
		"Hello, HTTPS integration test!",
		scriptResp["greeting"],
		"HTTPS WASM script should return expected greeting",
	)

	s.T().Logf("HTTPS Extism response: %+v", scriptResp)
}

func TestExtismHTTPSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ExtismHTTPSIntegrationTestSuite))
}
