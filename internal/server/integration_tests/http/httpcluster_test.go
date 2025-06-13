//go:build integration
// +build integration

package http_test

import (
	"bytes"
	"context"
	_ "embed"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Embedded TOML test configs
var (
	//go:embed testdata/empty_config.toml
	emptyConfigTOML string

	//go:embed testdata/two_listeners.toml.tmpl
	twoListenersTOML string

	//go:embed testdata/one_listener.toml.tmpl
	oneListenerTOML string

	//go:embed testdata/echo_app.toml.tmpl
	echoAppTOML string

	//go:embed testdata/route_v1.toml.tmpl
	routeV1TOML string

	//go:embed testdata/route_v1_v2.toml.tmpl
	routeV1V2TOML string

	//go:embed testdata/invalid_address.toml
	invalidAddressTOML string

	//go:embed testdata/duplicate_ports.toml.tmpl
	duplicatePortsTOML string

	//go:embed testdata/listener_with_route.toml.tmpl
	listenerWithRouteTOML string
)

// renderListenerWithRouteTOML renders the listener with route template with the given port
func renderListenerWithRouteTOML(t *testing.T, port string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(listenerWithRouteTOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port string }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// renderEchoAppTOMLForHTTP renders the echo app template with the given port for HTTP tests
func renderEchoAppTOMLForHTTP(t *testing.T, port string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(echoAppTOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port string }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// renderRouteV1TOML renders the route v1 template with the given port
func renderRouteV1TOML(t *testing.T, port string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(routeV1TOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port string }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// renderRouteV1V2TOML renders the route v1 v2 template with the given port
func renderRouteV1V2TOML(t *testing.T, port string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(routeV1V2TOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port string }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// renderOneListenerTOML renders the one listener template with the given port1
func renderOneListenerTOML(t *testing.T, port1 string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(oneListenerTOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port1 string }{Port1: port1})
	require.NoError(t, err)

	return buf.String()
}

// renderDuplicatePortsTOML renders the duplicate ports template with the given port
func renderDuplicatePortsTOML(t *testing.T, port string) string {
	t.Helper()

	tmpl, err := template.New("config").Parse(duplicatePortsTOML)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Port string }{Port: port})
	require.NoError(t, err)

	return buf.String()
}

// waitForHTTPServer waits for an HTTP server to be accessible
func waitForHTTPServer(t *testing.T, url string, expectedStatus int) {
	t.Helper()
	assert.Eventually(t, func() bool {
		resp, err := http.Get(url)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == expectedStatus
	}, 2*time.Second, 50*time.Millisecond, "HTTP server should be accessible at %s", url)
}

// waitForHTTPServerDown waits for an HTTP server to be inaccessible
func waitForHTTPServerDown(t *testing.T, url string) {
	t.Helper()
	assert.Eventually(t, func() bool {
		_, err := http.Get(url)
		return err != nil
	}, 2*time.Second, 50*time.Millisecond, "HTTP server should be down at %s", url)
}

// TestHTTPClusterDynamicListeners tests adding and removing HTTP listeners dynamically
func TestHTTPClusterDynamicListeners(t *testing.T) {
	// Enable debug logging for this test
	logging.SetupLogger("debug")

	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Test 1: Start with no listeners
	config1, err := config.NewConfigFromBytes([]byte(emptyConfigTOML))
	require.NoError(t, err)
	require.NoError(t, config1.Validate(), "Should validate config")

	tx1, err := transaction.FromTest("no-listeners", config1, nil)
	require.NoError(t, err)
	err = tx1.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx1)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx1.GetState())

	// Test 2: Add a listener with a route (so it actually starts)
	port := fmt.Sprintf("%d", testutil.GetRandomPort(t))

	config2Data := renderListenerWithRouteTOML(t, port)
	config2, err := config.NewConfigFromBytes([]byte(config2Data))
	require.NoError(t, err)
	require.NoError(t, config2.Validate(), "Should validate config")

	tx2, err := transaction.FromTest("add-listener", config2, nil)
	require.NoError(t, err)
	err = tx2.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx2)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx2.GetState())

	// Wait for listener to be accessible and test the route
	assert.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", port))
		if err != nil {
			t.Logf("Request error: %v", err)
			return false
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Logf("Read body error: %v", err)
			return false
		}

		t.Logf("Response status: %d, body: %s", resp.StatusCode, string(body))

		if resp.StatusCode != http.StatusOK {
			return false
		}

		return strings.Contains(string(body), "Test response")
	}, 2*time.Second, 50*time.Millisecond, "Listener with route should be accessible")

	// Test 3: Remove the listener
	config3, err := config.NewConfigFromBytes([]byte(emptyConfigTOML))
	require.NoError(t, err)
	require.NoError(t, config3.Validate(), "Should validate config")

	tx3, err := transaction.FromTest("remove-listener", config3, nil)
	require.NoError(t, err)
	err = tx3.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx3)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx3.GetState())

	// Verify listener is gone
	waitForHTTPServerDown(t, fmt.Sprintf("http://127.0.0.1:%s/", port))

	// Stop the HTTP runner
	httpRunner.Stop()
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}

// TestHTTPClusterWithRoutesAndApps tests full end-to-end with apps and routes
func TestHTTPClusterWithRoutesAndApps(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Create configuration with listener, endpoint, route and app
	port := fmt.Sprintf("%d", testutil.GetRandomPort(t))

	configData := renderEchoAppTOMLForHTTP(t, port)
	testConfig, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err)
	require.NoError(t, testConfig.Validate(), "Should validate config")

	// Create transaction
	tx, err := transaction.FromTest("echo-app-test", testConfig, nil)
	require.NoError(t, err)
	err = tx.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx.GetState())

	// Wait for echo endpoint to be accessible and verify response
	assert.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/echo", port))
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		return strings.Contains(string(body), "Echo says: Hello!")
	}, 2*time.Second, 50*time.Millisecond, "Echo endpoint should return expected response")

	// Test non-existent path
	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/notfound", port))
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)

	// Stop the HTTP runner
	httpRunner.Stop()
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}

// TestHTTPClusterRouteUpdates tests updating routes on existing listeners
func TestHTTPClusterRouteUpdates(t *testing.T) {
	// Enable debug logging for this test
	logging.SetupLogger("debug")

	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	port := fmt.Sprintf("%d", testutil.GetRandomPort(t))

	// Step 1: Create listener with one route
	config1Data := renderRouteV1TOML(t, port)
	config1, err := config.NewConfigFromBytes([]byte(config1Data))
	require.NoError(t, err)
	require.NoError(t, config1.Validate(), "Should validate config")

	tx1, err := transaction.FromTest("initial-route", config1, nil)
	require.NoError(t, err)
	err = tx1.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx1)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx1.GetState())

	// Wait for initial route to work
	assert.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/v1", port))
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		return strings.Contains(string(body), "V1: Response")
	}, 2*time.Second, 50*time.Millisecond, "V1 route should work")

	// Step 2: Add a second route
	config2Data := renderRouteV1V2TOML(t, port)
	config2, err := config.NewConfigFromBytes([]byte(config2Data))
	require.NoError(t, err)
	require.NoError(t, config2.Validate(), "Should validate config")

	tx2, err := transaction.FromTest("add-route", config2, nil)
	require.NoError(t, err)
	err = tx2.RunValidation()
	require.NoError(t, err)
	err = saga.ProcessTransaction(ctx, tx2)
	require.NoError(t, err)
	assert.Equal(t, "completed", tx2.GetState())

	// Wait for both routes to work
	assert.Eventually(t, func() bool {
		// Check V1 route
		resp1, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/v1", port))
		if err != nil {
			t.Logf("V1 request error: %v", err)
			return false
		}
		defer resp1.Body.Close()

		if resp1.StatusCode != http.StatusOK {
			t.Logf("V1 response status: %d", resp1.StatusCode)
			return false
		}

		body1, err := io.ReadAll(resp1.Body)
		if err != nil {
			t.Logf("V1 read body error: %v", err)
			return false
		}

		if !strings.Contains(string(body1), "V1: Response") {
			t.Logf("V1 unexpected body: %s", string(body1))
			return false
		}

		// Check V2 route
		resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/v2", port))
		if err != nil {
			t.Logf("V2 request error: %v", err)
			return false
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Logf("V2 response status: %d", resp2.StatusCode)
			return false
		}

		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			t.Logf("V2 read body error: %v", err)
			return false
		}

		if !strings.Contains(string(body2), "V2: Response") {
			t.Logf("V2 unexpected body: %s", string(body2))
			return false
		}

		return true
	}, 5*time.Second, 50*time.Millisecond, "Both V1 and V2 routes should work")

	// Stop the HTTP runner
	httpRunner.Stop()
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}

func TestHTTPClusterErrorHandling(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner with short siphon timeout for testing
	httpRunner, err := httplistener.NewRunner(
		httplistener.WithSiphonTimeout(100 * time.Millisecond),
	)
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Get a random port for testing
	port := fmt.Sprintf("%d", testutil.GetRandomPort(t))

	t.Run("invalid-address", func(t *testing.T) {
		// Test 1: Invalid address format
		config1, err := config.NewConfigFromBytes([]byte(invalidAddressTOML))
		require.NoError(t, err)
		require.NoError(t, config1.Validate(), "Should validate config")

		tx1, err := transaction.FromTest("bad-address", config1, nil)
		require.NoError(t, err)
		err = tx1.RunValidation()
		require.NoError(t, err)
		err = saga.ProcessTransaction(ctx, tx1)
		require.NoError(t, err)
		// Transaction should complete, but server won't start successfully
		assert.Equal(t, "completed", tx1.GetState())
	})

	t.Run("port-in-use", func(t *testing.T) {
		config2Data := renderOneListenerTOML(t, port)
		config2, err := config.NewConfigFromBytes([]byte(config2Data))
		require.NoError(t, err)
		require.NoError(t, config2.Validate(), "Should validate config")

		tx2, err := transaction.FromTest("first-listener", config2, nil)
		require.NoError(t, err)
		err = tx2.RunValidation()
		require.NoError(t, err)
		err = saga.ProcessTransaction(ctx, tx2)
		require.NoError(t, err)
		assert.Equal(t, "completed", tx2.GetState())

		// Server won't start (no routes), but transaction completes
		waitForHTTPServerDown(t, fmt.Sprintf("http://127.0.0.1:%s/", port))
	})

	t.Run("duplicate-ports", func(t *testing.T) {
		config3Data := renderDuplicatePortsTOML(t, port)
		config3, err := config.NewConfigFromBytes([]byte(config3Data))
		require.NoError(t, err, "Config creation should succeed")
		require.NotNil(t, config3)

		// Validation should fail due to duplicate listener addresses
		err = config3.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate ID: listener address")
	})

	t.Cleanup(func() {
		// Stop the HTTP runner
		httpRunner.Stop()
		assert.Eventually(t, func() bool {
			return !httpRunner.IsRunning()
		}, time.Second, 10*time.Millisecond)
	})
}

// TestHTTPClusterSagaCompensation tests rollback when participant fails
func TestHTTPClusterSagaCompensation(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Create a mock participant that will fail
	failingParticipant := &MockFailingParticipant{
		name: "failing-participant",
	}

	// Register participants
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)
	err = saga.RegisterParticipant(failingParticipant)
	require.NoError(t, err)

	// Start the HTTP runner
	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	// Wait for runner to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	// Create configuration
	port := fmt.Sprintf("%d", testutil.GetRandomPort(t))
	configData := renderOneListenerTOML(t, port)
	testConfig, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err)
	require.NoError(t, testConfig.Validate(), "Should validate config")

	// Create transaction
	tx, err := transaction.FromTest("compensation-test", testConfig, nil)
	require.NoError(t, err)
	err = tx.RunValidation()
	require.NoError(t, err)

	// Process transaction - should fail due to failing participant
	err = saga.ProcessTransaction(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "intentional failure")

	// Verify transaction is in compensated state (since compensation was successful)
	assert.Equal(t, "compensated", tx.GetState())

	// Verify HTTP runner compensated (no listeners should be running)
	waitForHTTPServerDown(t, fmt.Sprintf("http://127.0.0.1:%s/", port))

	// Stop the HTTP runner
	httpRunner.Stop()
	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}

// MockFailingParticipant is a saga participant that always fails
type MockFailingParticipant struct {
	name string
}

func (m *MockFailingParticipant) String() string {
	return m.name
}

func (m *MockFailingParticipant) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (m *MockFailingParticipant) Stop() {}

func (m *MockFailingParticipant) GetState() string {
	return "running"
}

func (m *MockFailingParticipant) IsRunning() bool {
	return true
}

func (m *MockFailingParticipant) GetStateChan(ctx context.Context) <-chan string {
	ch := make(chan string)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch
}

func (m *MockFailingParticipant) StageConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return fmt.Errorf("intentional failure for testing")
}

func (m *MockFailingParticipant) CompensateConfig(
	ctx context.Context,
	tx *transaction.ConfigTransaction,
) error {
	return nil
}

func (m *MockFailingParticipant) CommitConfig(ctx context.Context) error {
	return nil
}
