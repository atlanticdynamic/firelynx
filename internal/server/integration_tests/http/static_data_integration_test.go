//go:build integration

package http_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
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

//go:embed testdata/static_data_risor.toml.tmpl
var staticDataRisorTemplate string

// StaticDataIntegrationSuite tests that route-level static data properly merges with app-level static data
type StaticDataIntegrationSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	port        int
	httpRunner  *httplistener.Runner
	saga        *orchestrator.SagaOrchestrator
	runnerErrCh chan error
}

func TestStaticDataIntegrationSuite(t *testing.T) {
	suite.Run(t, &StaticDataIntegrationSuite{})
}

func (s *StaticDataIntegrationSuite) SetupSuite() {
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
	tmpl, err := template.New("static_data_risor").Parse(staticDataRisorTemplate)
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
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/route1", s.port))
		if err != nil {
			return false
		}
		defer func() { s.NoError(resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "HTTP listener should be ready")
}

func (s *StaticDataIntegrationSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.httpRunner != nil {
		// Wait for runner to stop
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		select {
		case err := <-s.runnerErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				s.T().Logf("HTTP runner error: %v", err)
			}
		case <-ctx.Done():
			s.T().Log("HTTP runner shutdown timeout")
		}
	}
}

// TestRisorStaticDataMerging tests that route-level static data overrides app-level static data
func (s *StaticDataIntegrationSuite) TestRisorStaticDataMerging() {
	s.T().Log("Testing Risor static data merging")

	// Test Route 1: Should get route-level override
	s.Run("Route1_OverrideRouteValue", func() {
		result := s.makeGetRequestAndParse(fmt.Sprintf("http://127.0.0.1:%d/api/route1", s.port))

		s.Equal("from_app", result.AppValue, "App-level static data should be preserved")
		s.Equal(
			"from_route1",
			result.RouteValue,
			"Route-level static data should override app-level",
		)
	})

	// Test Route 2: Should get different route-level override
	s.Run("Route2_DifferentRouteValue", func() {
		result := s.makeGetRequestAndParse(fmt.Sprintf("http://127.0.0.1:%d/api/route2", s.port))

		s.Equal("from_app", result.AppValue, "App-level static data should be preserved")
		s.Equal(
			"from_route2",
			result.RouteValue,
			"Route-level static data should override app-level",
		)
	})

	// Test Route 3: Should get app-level default (no route override)
	s.Run("Route3_AppDefault", func() {
		result := s.makeGetRequestAndParse(fmt.Sprintf("http://127.0.0.1:%d/api/route3", s.port))

		s.Equal("from_app", result.AppValue, "App-level static data should be preserved")
		s.Equal(
			"default_route_value",
			result.RouteValue,
			"Should use app-level default when no route override",
		)
	})
}

// StaticDataResponse represents the JSON response from the Risor script
type StaticDataResponse struct {
	AppValue   string `json:"appValue"`
	RouteValue string `json:"routeValue"`
	Timestamp  string `json:"timestamp"`
}

// makeGetRequestAndParse makes a GET request and parses the JSON response
func (s *StaticDataIntegrationSuite) makeGetRequestAndParse(url string) StaticDataResponse {
	resp, err := http.Get(url)
	s.Require().NoError(err, "HTTP request should succeed")
	defer func() { s.NoError(resp.Body.Close()) }()

	s.Equal(http.StatusOK, resp.StatusCode, "Should get 200 OK response")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Should be able to read response body")

	var result StaticDataResponse
	err = json.Unmarshal(body, &result)
	s.Require().NoError(err, "Response should be valid JSON")

	s.T().Logf("Response: %+v", result)
	return result
}
