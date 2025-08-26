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
	logging.SetupLogger("debug")

	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.tempDir = s.T().TempDir()
	s.port = testutil.GetRandomPort(s.T())
	s.runnerErrCh = make(chan error, 1)
	s.client = &http.Client{Timeout: 5 * time.Second}

	templateVars := struct {
		Port int
	}{
		Port: s.port,
	}

	tmpl, err := template.New("config").Parse(headersIntegrationTemplate)
	s.Require().NoError(err, "Failed to parse config template")

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	s.Require().NoError(err, "Failed to render config template")

	configData := configBuffer.String()
	s.T().Logf("Rendered config:\n%s", configData)

	cfg, err := config.NewConfigFromBytes([]byte(configData))
	s.Require().NoError(err, "Failed to load config")
	s.Require().NoError(cfg.Validate(), "Config validation failed")

	txStore := txstorage.NewMemoryStorage()
	s.saga = orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	s.httpRunner, err = httplistener.NewRunner()
	s.Require().NoError(err)

	err = s.saga.RegisterParticipant(s.httpRunner)
	s.Require().NoError(err)

	go func() {
		s.runnerErrCh <- s.httpRunner.Run(s.ctx)
	}()

	s.Require().Eventually(func() bool {
		select {
		case err := <-s.runnerErrCh:
			s.T().Fatalf("HTTP runner failed to start: %v", err)
			return false
		default:
			return s.httpRunner.IsRunning()
		}
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	tx, err := transaction.FromTest(s.T().Name(), cfg, nil)
	s.Require().NoError(err)

	err = tx.RunValidation()
	s.Require().NoError(err)

	err = s.saga.ProcessTransaction(s.ctx, tx)
	s.Require().NoError(err)

	s.Require().Equal("completed", tx.GetState())

	s.Require().Eventually(func() bool {
		resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/no-headers", s.port))
		if err != nil {
			return false
		}
		s.NoError(resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Server should be ready to accept requests")
}

func (s *HeadersIntegrationTestSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.httpRunner != nil {
		s.httpRunner.Stop()

		s.Require().Eventually(func() bool {
			return !s.httpRunner.IsRunning()
		}, time.Second, 10*time.Millisecond, "HTTP runner should stop")

		select {
		case err := <-s.runnerErrCh:
			if err != nil {
				s.Require().ErrorIs(err, context.Canceled, "HTTP runner should only exit due to context cancellation")
			}
		case <-time.After(2 * time.Second):
			s.T().Log("Timeout waiting for HTTP runner goroutine to complete")
		}
	}
}

func (s *HeadersIntegrationTestSuite) TestSetHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/set-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.Equal("v2.1", resp.Header.Get("X-API-Version"), "X-API-Version should be set by middleware")
	s.Equal(
		"no-cache",
		resp.Header.Get("Cache-Control"),
		"Cache-Control should be set by middleware",
	)

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
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.Equal("session=abc123; Path=/", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")
	s.Equal("custom-value", resp.Header.Get("X-Custom-Header"), "X-Custom-Header should be added")

	s.T().Logf("Add headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestRemoveHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/remove-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")
	s.Empty(resp.Header.Get("X-AspNet-Version"), "X-AspNet-Version header should be removed")

	s.T().Logf("Remove headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestCombinedOperations() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/combined-headers", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")

	s.Equal(
		"nosniff",
		resp.Header.Get("X-Content-Type-Options"),
		"X-Content-Type-Options should be set",
	)
	s.Equal("DENY", resp.Header.Get("X-Frame-Options"), "X-Frame-Options should be set")

	s.Equal("secure=true; HttpOnly", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")

	s.T().Logf("Combined operations response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestSecurityHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/security", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

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

	s.Empty(resp.Header.Get("Server"), "Server header should be removed")
	s.Empty(resp.Header.Get("X-Powered-By"), "X-Powered-By header should be removed")

	s.T().Logf("Security headers response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestCORSHeaders() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/cors", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

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
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.Equal("session=abc123; Path=/", resp.Header.Get("Set-Cookie"), "Set-Cookie should be added")

	s.T().Logf("Multiple cookies response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestHeaderOverwrite() {
	resp, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/overwrite", s.port))
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

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
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK")

	s.T().Logf("No headers control group response headers: %+v", resp.Header)
}

func (s *HeadersIntegrationTestSuite) TestRequestHeaders() {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/set-headers", s.port), nil)
	s.Require().NoError(err, "Failed to create request")

	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("User-Agent", "HeadersIntegrationTest/1.0")

	resp, err := s.client.Do(req)
	s.Require().NoError(err, "Failed to make request")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Request should succeed with custom headers")

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
	defer func() { s.NoError(resp.Body.Close()) }()

	// Headers that don't conflict with app-set headers work fine
	s.Equal("v2.1", resp.Header.Get("X-API-Version"), "Middleware-only headers work correctly")
	s.Equal("no-cache", resp.Header.Get("Cache-Control"), "Middleware-only headers work correctly")

	// But Content-Type is overwritten by the echo app
	s.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"),
		"Echo app overwrites Content-Type after middleware sets it")

	// Verify that the echo app indeed sets this header by checking the no-headers endpoint
	respNoHeaders, err := s.client.Get(fmt.Sprintf("http://127.0.0.1:%d/no-headers", s.port))
	s.Require().NoError(err, "Failed to make request to no-headers endpoint")
	defer func() { s.NoError(respNoHeaders.Body.Close()) }()

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
