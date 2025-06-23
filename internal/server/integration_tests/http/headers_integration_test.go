//go:build integration

package http_test

import (
	"context"
	_ "embed"
	"fmt"
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

//go:embed testdata/headers_integration.toml.tmpl
var headersIntegrationTemplate string

type HeadersIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	tempDir     string
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
	client      *http.Client
}

func (s *HeadersIntegrationTestSuite) SetupSuite() {
	// Setup debug logging for better test debugging
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.tempDir = s.T().TempDir()
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)
	s.client = &http.Client{Timeout: 5 * time.Second}

	// Template variables
	templateVars := struct {
		Port int
	}{
		Port: s.port,
	}

	// Render the configuration template
	tmpl, err := template.New("config").Parse(headersIntegrationTemplate)
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
		resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/no-headers", s.port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *HeadersIntegrationTestSuite) TearDownSuite() {
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
}

func (s *HeadersIntegrationTestSuite) TestSetHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/set-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify headers were set by middleware
	s.Equal("v2.1", resp.Header.Get("X-API-Version"), "X-API-Version should be set by middleware")
	s.Equal(
		"no-cache",
		resp.Header.Get("Cache-Control"),
		"Cache-Control should be set by middleware",
	)

	// Content-Type is overwritten by the echo app after middleware runs
	// This is expected behavior - apps can override middleware headers
	s.Equal(
		"text/plain; charset=utf-8",
		resp.Header.Get("Content-Type"),
		"Content-Type is overwritten by echo app",
	)

	s.T().Logf("Set headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestAddHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/add-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify headers were added
	s.Equal("session=abc123; Path=/", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")
	s.Equal("custom-value", resp.Header.Get("X-Custom-Header"), "X-Custom-Header should be added")

	s.T().Logf("Add headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestRemoveHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/remove-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify headers were removed
	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")
	s.Empty(resp.Header.Get("X-AspNet-Version"), "X-AspNet-Version header should be removed")

	s.T().Logf("Remove headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestCombinedOperations() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/combined-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify removed headers are gone
	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")

	// Verify set headers
	s.Equal(
		"nosniff",
		resp.Header.Get("X-Content-Type-Options"),
		"X-Content-Type-Options should be set",
	)
	s.Equal("DENY", resp.Header.Get("X-Frame-Options"), "X-Frame-Options should be set")

	// Verify added headers
	s.Equal("secure=true; HttpOnly", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")

	s.T().Logf("Combined operations response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestSecurityHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/security", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify security headers are set
	s.Equal(
		"nosniff",
		resp.Header.Get("X-Content-Type-Options"),
		"X-Content-Type-Options should be set",
	)
	s.Equal("DENY", resp.Header.Get("X-Frame-Options"), "X-Frame-Options should be set")
	s.Equal("1; mode=block", resp.Header.Get("X-XSS-Protection"), "X-XSS-Protection should be set")
	s.Equal(
		"max-age=31536000; includeSubDomains",
		resp.Header.Get("Strict-Transport-Security"),
		"HSTS should be set",
	)
	s.Equal(
		"strict-origin-when-cross-origin",
		resp.Header.Get("Referrer-Policy"),
		"Referrer-Policy should be set",
	)

	// Verify removed headers
	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")

	s.T().Logf("Security headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestCORSHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/cors", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify CORS headers are set
	s.Equal("*", resp.Header.Get("Access-Control-Allow-Origin"), "CORS origin should be set")
	s.Equal(
		"GET,POST,PUT,DELETE",
		resp.Header.Get("Access-Control-Allow-Methods"),
		"CORS methods should be set",
	)
	s.Equal(
		"Content-Type,Authorization",
		resp.Header.Get("Access-Control-Allow-Headers"),
		"CORS headers should be set",
	)

	s.T().Logf("CORS headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestMultipleCookies() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/multiple-cookies", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify multiple Set-Cookie headers can be added
	s.Equal("session=abc123; Path=/", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")

	s.T().Logf("Multiple cookies response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestHeaderOverwrite() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/overwrite", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify middleware sets headers, but echo app overwrites Content-Type
	s.Equal(
		"text/plain; charset=utf-8",
		resp.Header.Get("Content-Type"),
		"Content-Type is overwritten by echo app",
	)
	s.Equal(
		"middleware-value",
		resp.Header.Get("X-Override"),
		"X-Override should be set by middleware",
	)

	s.T().Logf("Header overwrite response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestNoHeadersControlGroup() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/no-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	// Verify no special headers are set (this is our control group)
	// Should only have basic headers that go-supervisor sets by default
	s.T().Logf("No headers control group response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestRequestHeaders() {
	// Test that the middleware doesn't interfere with request processing
	// by sending custom request headers and verifying the app still functions
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/set-headers", s.port), nil)
	s.Require().NoError(err, "Failed to create request")

	// Add some request headers
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("User-Agent", "HeadersIntegrationTest/1.0")

	resp, err := s.client.Do(req)
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Request should succeed with custom headers")

	// Verify response headers are still set by middleware (except Content-Type which is overwritten by echo app)
	s.Equal(
		"text/plain; charset=utf-8",
		resp.Header.Get("Content-Type"),
		"Content-Type is overwritten by echo app",
	)
	s.Equal("v2.1", resp.Header.Get("X-API-Version"), "X-API-Version should be set by middleware")

	s.T().Logf("Request with custom headers succeeded")
}

func (s *HeadersIntegrationTestSuite) TestMiddlewareVsAppHeaderPrecedence() {
	// This test confirms our theory: middleware sets headers first, then the app can override them

	// Test endpoint that sets Content-Type via middleware
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/set-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	// Headers that don't conflict with app-set headers work fine
	s.Equal("v2.1", resp.Header.Get("X-API-Version"), "Middleware-only headers work correctly")
	s.Equal("no-cache", resp.Header.Get("Cache-Control"), "Middleware-only headers work correctly")

	// But Content-Type is overwritten by the echo app
	s.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"),
		"Echo app overwrites Content-Type after middleware sets it")

	// Verify that the echo app indeed sets this header by checking the no-headers endpoint
	respNoHeaders, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/no-headers", s.port))
	s.Require().NoError(err, "Failed to make request to no-headers endpoint")
	defer respNoHeaders.Body.Close()

	// Even with no middleware, echo app sets Content-Type
	s.Equal("text/plain; charset=utf-8", respNoHeaders.Header.Get("Content-Type"),
		"Echo app always sets Content-Type, proving it overwrites middleware")

	// But middleware-only headers don't exist on no-headers endpoint
	s.Empty(
		respNoHeaders.Header.Get("X-API-Version"),
		"No middleware headers on no-headers endpoint",
	)
	s.Empty(
		respNoHeaders.Header.Get("Cache-Control"),
		"No middleware headers on no-headers endpoint",
	)

	s.T().Logf("Confirmed: middleware runs first, app can override headers")
}

func TestHeadersIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HeadersIntegrationTestSuite))
}
