//go:build integration

package http_test

import (
	"bufio"
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
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/logger_integration.toml.tmpl
var loggerIntegrationTemplate string

// LogEntry represents the structure of JSON log entries
type LogEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
	HTTP  struct {
		Method      string                 `json:"method,omitempty"`
		Path        string                 `json:"path,omitempty"`
		StatusCode  int                    `json:"status,omitempty"`
		ClientIP    string                 `json:"client_ip,omitempty"`
		Duration    int                    `json:"duration,omitempty"`
		Query       string                 `json:"query,omitempty"`        // Query string as used by logger
		QueryParams map[string]interface{} `json:"query_params,omitempty"` // Keep for backward compatibility
		Protocol    string                 `json:"protocol,omitempty"`
		Host        string                 `json:"host,omitempty"`
		Scheme      string                 `json:"scheme,omitempty"`
		UserAgent   string                 `json:"user_agent,omitempty"`
		BodySize    *int                   `json:"body_size,omitempty"`
		Request     struct {
			Headers map[string][]string `json:"headers,omitempty"`
		} `json:"request,omitempty"`
		Response struct {
			Headers map[string][]string `json:"headers,omitempty"`
		} `json:"response,omitempty"`
	} `json:"http,omitempty"`
}

type LoggerIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	tempDir     string
	envLogDir   string
	port        int
	logFile     string
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	originalEnv map[string]string
	runnerErrCh chan error
}

func (s *LoggerIntegrationTestSuite) SetupSuite() {
	// Setup debug logging for better test debugging
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.tempDir = s.T().TempDir()
	s.port = testutil.GetRandomPort(s.T())
	s.logFile = filepath.Join(s.tempDir, "firelynx.log")
	s.runnerErrCh = make(chan error, 1)

	// Set environment variables for environment variable interpolation testing
	s.envLogDir = filepath.Join(s.tempDir, "env_logs")
	err := os.MkdirAll(s.envLogDir, 0o755)
	s.Require().NoError(err, "Failed to create env log directory")

	s.originalEnv = map[string]string{
		"FIRELYNX_LOG_DIR": os.Getenv("FIRELYNX_LOG_DIR"),
		"HOSTNAME":         os.Getenv("HOSTNAME"),
		"TEST_SESSION":     os.Getenv("TEST_SESSION"),
	}

	// Set test environment variables
	os.Setenv("FIRELYNX_LOG_DIR", s.envLogDir)
	os.Setenv("HOSTNAME", "test-host")
	os.Setenv("TEST_SESSION", "session-123")

	// Template variables
	templateVars := struct {
		Port    int
		LogFile string
	}{
		Port:    s.port,
		LogFile: s.logFile,
	}

	// Render the configuration template
	tmpl, err := template.New("config").Parse(loggerIntegrationTemplate)
	s.Require().NoError(err, "Failed to parse config template")

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

	// Start the HTTP runner
	go func() {
		s.runnerErrCh <- s.httpRunner.Run(s.ctx)
	}()

	// Wait for runner to start with error checking
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
	tx, err := transaction.FromTest(s.T().Name(), cfg, nil)
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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/test", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *LoggerIntegrationTestSuite) TearDownSuite() {
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

		// Wait for background goroutine to complete
		select {
		case err := <-s.runnerErrCh:
			if err != nil && err != context.Canceled {
				s.T().Logf("HTTP runner exited with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			s.T().Log("Timeout waiting for HTTP runner goroutine to complete")
		}
	}

	// Restore environment variables
	for key, value := range s.originalEnv {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

func (s *LoggerIntegrationTestSuite) TestStandardLogger() {
	// Make a request to generate logs
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/test", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")
	s.Contains(
		string(body),
		"Integration Test Response",
		"Response should contain expected text",
	)

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(s.logFile)
		return len(logEntries) > 0
	}, 5*time.Second, 100*time.Millisecond, "Log file should contain entries")

	// Find our test request log entry
	var testEntry *LogEntry
	for _, entry := range logEntries {
		if entry.HTTP.Path == "/test" && entry.HTTP.Method == "GET" {
			testEntry = &entry
			break
		}
	}
	s.Require().NotNil(testEntry, "Should find log entry for /test request")

	// Verify standard preset fields are present
	s.Equal("GET", testEntry.HTTP.Method, "Method should be logged")
	s.Equal("/test", testEntry.HTTP.Path, "Path should be logged")
	s.Equal(200, testEntry.HTTP.StatusCode, "Status code should be logged")
	s.NotEmpty(testEntry.HTTP.ClientIP, "Client IP should be logged")
	s.NotZero(testEntry.HTTP.Duration, "Duration should be logged")
	s.Equal("INFO", testEntry.Level, "Log level should be info")

	s.T().Logf("Standard log entry: %+v", testEntry)
}

func (s *LoggerIntegrationTestSuite) TestEnvironmentVariableLogger() {
	envLogFile := filepath.Join(s.envLogDir, "access-test-host.log")

	// Make a request with query parameters and headers to test detailed preset
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://127.0.0.1:%d/env-test?debug=true", s.port),
		nil,
	)
	s.Require().NoError(err, "Failed to create request")

	req.Header.Set("User-Agent", "FireLynx-EnvTest/1.0")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(envLogFile)
		for _, entry := range logEntries {
			if entry.HTTP.Path == "/env-test" && entry.HTTP.Method == "GET" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Environment variable log file should contain /env-test entry")

	// Find our test request log entry
	var testEntry *LogEntry
	for _, entry := range logEntries {
		if entry.HTTP.Path == "/env-test" && entry.HTTP.Method == "GET" {
			testEntry = &entry
			break
		}
	}
	s.Require().NotNil(testEntry, "Should find log entry for /env-test request")

	// Verify detailed preset fields are present (from env-logger config)
	s.Equal("GET", testEntry.HTTP.Method, "Method should be logged")
	s.Equal("/env-test", testEntry.HTTP.Path, "Path should be logged")
	s.NotEmpty(testEntry.HTTP.Query, "Query string should be logged with detailed preset")
	s.NotEmpty(testEntry.HTTP.Protocol, "Protocol should be logged with detailed preset")
	s.NotEmpty(testEntry.HTTP.Host, "Host should be logged with detailed preset")
	s.Equal(
		"INFO",
		testEntry.Level,
		"Log level should be info for successful requests (status 200)",
	)

	// Verify query parameters were captured
	s.Contains(testEntry.HTTP.Query, "debug=true", "Should contain debug parameter")

	s.T().Logf("Environment variable log entry: %+v", testEntry)
}

func (s *LoggerIntegrationTestSuite) TestMinimalLogger() {
	logFile := s.logFile + ".minimal"

	// Make a request to test minimal preset
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/minimal-test", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(logFile)
		for _, entry := range logEntries {
			if entry.HTTP.Path == "/minimal-test" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Minimal log file should contain /minimal-test entry")

	// Find our test request log entry
	var testEntry *LogEntry
	for _, entry := range logEntries {
		if entry.HTTP.Path == "/minimal-test" {
			testEntry = &entry
			break
		}
	}
	s.Require().NotNil(testEntry, "Should find log entry for /minimal-test request")

	// Verify minimal preset fields only (method, path, status_code)
	s.Equal("GET", testEntry.HTTP.Method, "Minimal preset should include method")
	s.Equal("/minimal-test", testEntry.HTTP.Path, "Minimal preset should include path")
	s.Equal(200, testEntry.HTTP.StatusCode, "Minimal preset should include status_code")

	// Verify minimal preset excludes detailed fields
	s.Empty(testEntry.HTTP.ClientIP, "Minimal preset should NOT include client_ip")
	s.Zero(testEntry.HTTP.Duration, "Minimal preset should NOT include duration")
	s.Nil(testEntry.HTTP.QueryParams, "Minimal preset should NOT include query_params")

	s.T().Logf("Minimal log entry: %+v", testEntry)
}

func (s *LoggerIntegrationTestSuite) TestManualLogger() {
	logFile := filepath.Join(s.envLogDir, "manual-session-123.log")

	// Make a request with various headers to test manual field configuration
	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://127.0.0.1:%d/manual-test?test=manual", s.port),
		nil,
	)
	s.Require().NoError(err, "Failed to create request")

	req.Header.Set("User-Agent", "FireLynx-ManualTest/1.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer should-be-excluded")
	req.Header.Set("Cookie", "session=should-be-excluded")

	resp, err := client.Do(req)
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(logFile)
		for _, entry := range logEntries {
			if entry.HTTP.Path == "/manual-test" && entry.HTTP.Method == "POST" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Manual log file should contain /manual-test entry")

	// Find our test request log entry
	var testEntry *LogEntry
	for _, entry := range logEntries {
		if entry.HTTP.Path == "/manual-test" && entry.HTTP.Method == "POST" {
			testEntry = &entry
			break
		}
	}
	s.Require().NotNil(testEntry, "Should find log entry for /manual-test request")

	// Verify manual configuration fields
	s.Equal("POST", testEntry.HTTP.Method, "Manual config should include method")
	s.Equal("/manual-test", testEntry.HTTP.Path, "Manual config should include path")
	s.Equal(200, testEntry.HTTP.StatusCode, "Manual config should include status_code")
	s.NotEmpty(testEntry.HTTP.ClientIP, "Manual config should include client_ip")
	s.NotZero(testEntry.HTTP.Duration, "Manual config should include duration")
	s.NotEmpty(testEntry.HTTP.Query, "Manual config should include query string")
	s.NotEmpty(testEntry.HTTP.Protocol, "Manual config should include protocol")
	s.NotEmpty(testEntry.HTTP.Host, "Manual config should include host")
	s.Equal(
		"INFO",
		testEntry.Level,
		"Log level should be info for successful requests (status 200)",
	)

	s.T().Logf("Manual configuration log entry: %+v", testEntry)
}

func (s *LoggerIntegrationTestSuite) TestPresetFunctionality() {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/preset-test", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(s.logFile)
		for _, entry := range logEntries {
			if entry.HTTP.Path == "/preset-test" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Log file should contain /preset-test entry")

	var testEntry *LogEntry
	for _, entry := range logEntries {
		if entry.HTTP.Path == "/preset-test" {
			testEntry = &entry
			break
		}
	}
	s.Require().NotNil(testEntry, "Should find log entry for /preset-test request")

	s.NotEmpty(testEntry.HTTP.Method, "Standard preset should include method")
	s.NotEmpty(testEntry.HTTP.Path, "Standard preset should include path")
	s.NotZero(testEntry.HTTP.StatusCode, "Standard preset should include status_code")
	s.NotEmpty(testEntry.HTTP.ClientIP, "Standard preset should include client_ip")
	s.NotZero(testEntry.HTTP.Duration, "Standard preset should include duration")
}

func (s *LoggerIntegrationTestSuite) TestPathFiltering() {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/normal", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp2.Body.Close()

	var logEntries []LogEntry
	s.Eventually(func() bool {
		logEntries = s.readLogEntries(s.logFile)
		for _, entry := range logEntries {
			if entry.HTTP.Path == "/normal" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Log file should contain /normal entry")

	healthLogged := false
	normalLogged := false

	for _, entry := range logEntries {
		if entry.HTTP.Path == "/health" {
			healthLogged = true
		}
		if entry.HTTP.Path == "/normal" {
			normalLogged = true
		}
	}

	s.False(healthLogged, "/health should be filtered out by exclude_paths")
	s.True(normalLogged, "/normal should be logged")
}

func (s *LoggerIntegrationTestSuite) TestEnvironmentVariableInterpolation() {
	// Test that environment variables were properly interpolated by checking file existence
	expectedFiles := []string{
		"access-test-host.log",   // From ${FIRELYNX_LOG_DIR}/access-${HOSTNAME}.log
		"manual-session-123.log", // From ${FIRELYNX_LOG_DIR}/manual-${TEST_SESSION}.log
	}

	// Make a request to ensure the loggers create the files
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/env-interpolation-test", s.port))
	s.Require().NoError(err, "Failed to make request")
	resp.Body.Close()

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(s.envLogDir, expectedFile)

		s.Eventually(func() bool {
			_, err := os.Stat(filePath)
			return err == nil
		}, 5*time.Second, 100*time.Millisecond, "Environment variable interpolated log file should exist: %s", expectedFile)

		s.T().Logf("Environment variable interpolated file exists: %s", filePath)
	}
}

func (s *LoggerIntegrationTestSuite) TestMultipleMiddlewarePerEndpoint() {
	// Test GET request - should only appear in get-only.log
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/multi-middleware", s.port))
	s.Require().NoError(err, "Failed to make GET request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK for GET")

	// Test POST request - should only appear in post-only.log
	postResp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/multi-middleware", s.port),
		"application/json",
		strings.NewReader(`{"test": "data"}`),
	)
	s.Require().NoError(err, "Failed to make POST request")
	defer postResp.Body.Close()

	s.Require().Equal(http.StatusOK, postResp.StatusCode, "Expected 200 OK for POST")

	// Wait for logs to be written
	getOnlyLogFile := s.logFile + ".get-only.log"
	postOnlyLogFile := s.logFile + ".post-only.log"

	// Verify GET request appears only in get-only.log
	s.Eventually(func() bool {
		entries := s.readLogEntries(getOnlyLogFile)
		for _, entry := range entries {
			if entry.HTTP.Path == "/multi-middleware" && entry.HTTP.Method == "GET" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "GET request should appear in get-only.log")

	// Verify POST request appears only in post-only.log
	s.Eventually(func() bool {
		entries := s.readLogEntries(postOnlyLogFile)
		for _, entry := range entries {
			if entry.HTTP.Path == "/multi-middleware" && entry.HTTP.Method == "POST" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "POST request should appear in post-only.log")

	// Verify GET request does NOT appear in post-only.log
	postEntries := s.readLogEntries(postOnlyLogFile)
	for _, entry := range postEntries {
		s.NotEqual("GET", entry.HTTP.Method, "GET requests should not appear in post-only.log")
	}

	// Verify POST request does NOT appear in get-only.log
	getEntries := s.readLogEntries(getOnlyLogFile)
	for _, entry := range getEntries {
		s.NotEqual("POST", entry.HTTP.Method, "POST requests should not appear in get-only.log")
	}

	// Verify different presets are applied
	// GET logger uses minimal preset (only method, path, status)
	getEntry := getEntries[0]
	s.Equal("GET", getEntry.HTTP.Method)
	s.Equal("/multi-middleware", getEntry.HTTP.Path)
	s.Equal(200, getEntry.HTTP.StatusCode)
	s.Empty(getEntry.HTTP.ClientIP, "Minimal preset should not include client IP")
	s.Zero(getEntry.HTTP.Duration, "Minimal preset should not include duration")

	// POST logger uses detailed preset (includes more fields)
	postEntry := postEntries[0]
	s.Equal("POST", postEntry.HTTP.Method)
	s.Equal("/multi-middleware", postEntry.HTTP.Path)
	s.Equal(200, postEntry.HTTP.StatusCode)
	s.NotEmpty(postEntry.HTTP.ClientIP, "Detailed preset should include client IP")
	s.NotZero(postEntry.HTTP.Duration, "Detailed preset should include duration")
	s.NotEmpty(postEntry.HTTP.Protocol, "Detailed preset should include protocol")
	s.NotEmpty(postEntry.HTTP.Host, "Detailed preset should include host")

	s.T().Logf("GET-only log entries: %d", len(getEntries))
	s.T().Logf("POST-only log entries: %d", len(postEntries))
}

func (s *LoggerIntegrationTestSuite) TestMethodExclusion() {
	// Test excluded methods (HEAD and OPTIONS should not be logged)
	client := &http.Client{}

	// Make a GET request - should be logged
	getReq, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://127.0.0.1:%d/exclude-methods", s.port),
		nil,
	)
	s.Require().NoError(err)
	getResp, err := client.Do(getReq)
	s.Require().NoError(err)
	defer getResp.Body.Close()

	// Make a HEAD request - should NOT be logged
	headReq, err := http.NewRequest(
		"HEAD",
		fmt.Sprintf("http://127.0.0.1:%d/exclude-methods", s.port),
		nil,
	)
	s.Require().NoError(err)
	headResp, err := client.Do(headReq)
	s.Require().NoError(err)
	defer headResp.Body.Close()

	// Make an OPTIONS request - should NOT be logged
	optionsReq, err := http.NewRequest(
		"OPTIONS",
		fmt.Sprintf("http://127.0.0.1:%d/exclude-methods", s.port),
		nil,
	)
	s.Require().NoError(err)
	optionsResp, err := client.Do(optionsReq)
	s.Require().NoError(err)
	defer optionsResp.Body.Close()

	// Wait for logs and verify
	excludeLogFile := s.logFile + ".exclude-methods.log"

	s.Eventually(func() bool {
		entries := s.readLogEntries(excludeLogFile)
		return len(entries) > 0
	}, 5*time.Second, 100*time.Millisecond, "Should have at least one log entry")

	entries := s.readLogEntries(excludeLogFile)

	// Verify only GET request is logged
	for _, entry := range entries {
		s.NotEqual("HEAD", entry.HTTP.Method, "HEAD requests should be excluded")
		s.NotEqual("OPTIONS", entry.HTTP.Method, "OPTIONS requests should be excluded")
	}

	// Should have exactly one entry (the GET request)
	s.Len(entries, 1, "Should have exactly one log entry (GET request only)")
	s.Equal("GET", entries[0].HTTP.Method, "The single entry should be the GET request")

	s.T().Logf("Excluded methods test passed with %d logged entries", len(entries))
}

func (s *LoggerIntegrationTestSuite) TestTextFormatLogger() {
	// Test standard preset with text format
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/text-standard", s.port))
	s.Require().NoError(err, "Failed to make request to /text-standard")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	standardLogFile := s.logFile + ".text-standard.log"
	s.Eventually(func() bool {
		content := s.readTextLogFile(standardLogFile)
		return strings.Contains(content, "/text-standard") && strings.Contains(content, "GET")
	}, 5*time.Second, 100*time.Millisecond, "Standard text log should contain request entry")

	// Test detailed preset with text format
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/text-detailed?test=value", s.port))
	s.Require().NoError(err, "Failed to make request to /text-detailed")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	detailedLogFile := s.logFile + ".text-detailed.log"
	s.Eventually(func() bool {
		content := s.readTextLogFile(detailedLogFile)
		return strings.Contains(content, "/text-detailed") &&
			strings.Contains(content, "test=value")
	}, 5*time.Second, 100*time.Millisecond, "Detailed text log should contain request entry with query params")

	// Verify text format characteristics
	standardContent := s.readTextLogFile(standardLogFile)
	s.NotEmpty(standardContent, "Standard text log should not be empty")
	s.NotContains(standardContent, "{", "Text format should not contain JSON braces")
	s.Contains(standardContent, "INFO", "Text format should contain INFO level")
	s.Contains(standardContent, "method=GET", "Text format should contain method=GET")
	s.Contains(standardContent, "path=/text-standard", "Text format should contain path")
	s.Contains(standardContent, "status=200", "Text format should contain status code")
	s.Contains(standardContent, "client_ip=127.0.0.1", "Text format should contain client IP")
	s.Contains(standardContent, "duration=", "Text format should contain duration")

	detailedContent := s.readTextLogFile(detailedLogFile)
	s.NotEmpty(detailedContent, "Detailed text log should not be empty")
	s.NotContains(detailedContent, "{", "Text format should not contain JSON braces")
	s.Contains(detailedContent, "INFO", "Text format should contain INFO level")
	s.Contains(detailedContent, "method=GET", "Text format should contain method=GET")
	s.Contains(detailedContent, "path=/text-detailed", "Text format should contain path")
	s.Contains(
		detailedContent,
		"query=\"test=value\"",
		"Detailed text format should contain query params",
	)
	s.Contains(detailedContent, "protocol=HTTP/1.1", "Detailed text format should contain protocol")
	s.Contains(detailedContent, "host=127.0.0.1:", "Detailed text format should contain host")
	s.Contains(detailedContent, "scheme=http", "Detailed text format should contain scheme")

	s.T().Logf("Standard text log content: %s", standardContent)
	s.T().Logf("Detailed text log content: %s", detailedContent)
}

// Helper function to read text log files (non-JSON)
func (s *LoggerIntegrationTestSuite) readTextLogFile(logFilePath string) string {
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		s.T().Logf("Text log file %s does not exist or cannot be read: %v", logFilePath, err)
		return ""
	}
	return string(content)
}

// Helper function to read and parse log entries from a file
func (s *LoggerIntegrationTestSuite) readLogEntries(logFilePath string) []LogEntry {
	file, err := os.Open(logFilePath)
	if err != nil {
		s.T().Logf("Log file %s does not exist or cannot be opened: %v", logFilePath, err)
		return nil
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		s.T().Logf("Raw JSON line: %s", line)

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			s.T().Logf("Failed to parse JSON log line: %s, error: %v", line, err)
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		s.T().Logf("Error reading log file: %v", err)
	}

	s.T().Logf("Read %d log entries from %s", len(entries), logFilePath)
	for i, entry := range entries {
		s.T().
			Logf("Entry %d: Method=%s, Path=%s, Msg=%s", i+1, entry.HTTP.Method, entry.HTTP.Path, entry.Msg)
	}
	return entries
}

func TestLoggerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(LoggerIntegrationTestSuite))
}
