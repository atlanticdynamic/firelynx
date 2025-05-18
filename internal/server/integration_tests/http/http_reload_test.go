//go:build integration
// +build integration

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	httpRunner "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/robbyt/go-supervisor/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPReloadWithDirectUpdate tests reloading the HTTP configuration directly
// via the reload mechanism, without using file-based reloading
func TestHTTPReloadWithDirectUpdate(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a logger that captures logs
	var logBuf testutil.ThreadSafeBuffer
	testHandler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(testHandler)

	// Get a free port for HTTP
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Create a context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a minimal configuration
	initialConfig := &config.Config{
		Version: config.VersionLatest,
		Listeners: listeners.ListenerCollection{
			&listeners.Listener{
				ID:      "http_listener",
				Type:    listeners.TypeHTTP,
				Address: httpAddr,
				Options: options.HTTP{
					ReadTimeout:  time.Second,
					WriteTimeout: time.Second,
					IdleTimeout:  time.Second,
					DrainTimeout: time.Second,
				},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			&endpoints.Endpoint{
				ID:         "echo_endpoint",
				ListenerID: "http_listener",
				Routes: routes.RouteCollection{
					&routes.Route{
						AppID: "echo_app",
						Rule: &conditions.HTTPRule{
							PathPrefix: "/echo",
						},
					},
				},
			},
		},
		Apps: apps.AppCollection{
			&apps.AppDefinition{
				ID:   "echo_app",
				Type: apps.TypeEcho,
				App: &echo.EchoApp{
					Response: "initial echo response",
				},
			},
		},
	}

	// Create a config callback function that returns our configuration
	var currentConfig *config.Config
	currentConfig = initialConfig

	configCallback := func() config.Config {
		if currentConfig == nil {
			logger.Error("Configuration is nil in callback")
			return config.Config{Version: config.VersionLatest}
		}
		return *currentConfig
	}

	// Create the application registry
	appRegistry, err := serverApps.NewAppCollection([]serverApps.App{})
	require.NoError(t, err, "Failed to create app registry")

	// Create the route registry
	routeRegistry := routing.NewRouteRegistry()

	// Create the transaction manager
	coreRunner, err := txmgr.NewRunner(
		configCallback,
		txmgr.WithLogHandler(logger.Handler()),
		txmgr.WithAppRegistry(appRegistry),
		txmgr.WithRouteRegistry(routeRegistry),
	)
	require.NoError(t, err, "Failed to create server core")

	// Create HTTP runner
	httpCallback := coreRunner.GetHTTPConfigCallback()
	runner, err := httpRunner.NewRunner(
		httpCallback,
		httpRunner.WithManagerLogger(logger.WithGroup("http.Runner")),
	)
	require.NoError(t, err, "Failed to create HTTP runner")

	// Create a supervisor to manage the runnables
	super, err := supervisor.New(
		supervisor.WithLogHandler(logger.Handler()),
		supervisor.WithRunnables(coreRunner, runner),
		supervisor.WithContext(ctx),
	)
	require.NoError(t, err, "Failed to create supervisor")

	// Start the supervisor in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- super.Run()
	}()

	// Wait briefly to ensure server starts up
	time.Sleep(500 * time.Millisecond)

	// Check for early errors
	select {
	case err := <-errCh:
		require.NoError(t, err, "Server failed to start")
	default:
		// No errors, continue with test
	}

	// Create an HTTP client for testing
	httpClient := &http.Client{Timeout: 2 * time.Second}

	// Test the initial echo endpoint
	t.Run("Initial echo endpoint responds", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d/echo", httpPort)

		// Try a few times to connect to the endpoint as it might take a moment to start
		var resp *http.Response
		var err error
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		require.NoError(t, err, "Failed to connect to echo endpoint")
		require.NotNil(t, resp, "Response should not be nil")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/echo", echoResp["path"], "Wrong path")
	})

	// Update the configuration with a new route
	updatedConfig := *initialConfig
	updatedEndpoints := make(endpoints.EndpointCollection, len(initialConfig.Endpoints))
	copy(updatedEndpoints, initialConfig.Endpoints)

	// Create a copy of the first endpoint
	endpoint := *updatedEndpoints[0]

	// Create a new routes collection with the existing routes
	routes := make(routes.RouteCollection, len(endpoint.Routes))
	copy(routes, endpoint.Routes)

	// Add the new route
	routes = append(routes, &routes.Route{
		AppID: "echo_app",
		Rule: &conditions.HTTPRule{
			PathPrefix: "/new-path",
		},
	})

	// Update the endpoint with the new routes
	endpoint.Routes = routes
	updatedEndpoints[0] = &endpoint

	// Update the config with the new endpoints
	updatedConfig.Endpoints = updatedEndpoints

	// Update the current configuration
	currentConfig = &updatedConfig

	// Manually trigger a reload on the core runner
	t.Log("Manually triggering reload...")
	coreRunner.Reload()

	// Monitor for HTTP runner to enter reloading state
	reloaded := false
	stateCh := runner.GetStateChan(ctx)

	// Add a timeout for waiting for reload to complete
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

waitLoop:
	for {
		select {
		case state := <-stateCh:
			t.Logf("HTTP runner state changed: %s", state)
			if state == "Reloading" || state == "Running" {
				reloaded = true
				break waitLoop
			}
		case <-timer.C:
			t.Log("Timeout waiting for reload")
			break waitLoop
		}
	}

	// Sleep a bit to allow the reload to finish
	time.Sleep(500 * time.Millisecond)

	// Test the new route
	t.Run("New route responds after direct reload", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d/new-path", httpPort)

		// Try a few times as the reload might take a moment
		var resp *http.Response
		var err error
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if !reloaded {
			t.Log("Warning: Reload didn't properly trigger state change")
		}

		require.NoError(t, err, "Failed to connect to new endpoint")
		require.NotNil(t, resp, "Response should not be nil")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/new-path", echoResp["path"], "Wrong path")
	})

	// Shutdown the server
	cancel()

	// Wait for server to shut down
	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("Server shutdown with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("Server shutdown timed out")
	}

	// Log buffer contents for debugging
	t.Logf("Logs:\n%s", logBuf.String())
}

// TestHTTPReloadWithSignal tests reloading the HTTP configuration via SIGHUP signal
func TestHTTPReloadWithSignal(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "http_reload_test.toml")

	// Get a free port for HTTP
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Create the initial config file
	initialConfigContent := fmt.Sprintf(`
version = "v1"

[logging]
level = "debug"
format = "text"

[[listeners]]
id = "http_listener"
address = "%s"
type = "http"

[listeners.http]
read_timeout = "1s"
write_timeout = "1s"
idle_timeout = "1s"
drain_timeout = "1s"

[[endpoints]]
id = "echo_endpoint"
listener_id = "http_listener"

[[endpoints.routes]]
app_id = "echo_app"
[endpoints.routes.http]
path_prefix = "/echo"

[[apps]]
id = "echo_app"
type = "echo"
[apps.echo]
response = "This is a test echo response"
`, httpAddr)

	err := os.WriteFile(configPath, []byte(initialConfigContent), 0o644)
	require.NoError(t, err, "Failed to write initial config file")

	// Create a logger that captures logs
	var logBuf testutil.ThreadSafeBuffer
	testHandler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(testHandler)

	// Create a context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the configuration from file
	cfg, err := config.NewConfig(configPath)
	require.NoError(t, err, "Failed to load config from file")

	// Start a mini-server with just the parts we need for this test
	// We'll use helper functions to simulate the firelynx server command
	var configProvider *testConfigProvider
	configProvider = newTestConfigProvider(cfg)

	// Create the application registry
	appRegistry, err := serverApps.NewAppCollection([]serverApps.App{})
	require.NoError(t, err, "Failed to create app registry")

	// Create the route registry
	routeRegistry := routing.NewRouteRegistry()

	// Create the transaction manager
	coreRunner, err := txmgr.NewRunner(
		configProvider.getConfig,
		txmgr.WithLogHandler(logger.Handler()),
		txmgr.WithAppRegistry(appRegistry),
		txmgr.WithRouteRegistry(routeRegistry),
	)
	require.NoError(t, err, "Failed to create server core")

	// Create HTTP runner
	httpCallback := coreRunner.GetHTTPConfigCallback()
	runner, err := httpRunner.NewRunner(
		httpCallback,
		httpRunner.WithManagerLogger(logger.WithGroup("http.Runner")),
	)
	require.NoError(t, err, "Failed to create HTTP runner")

	// Create a supervisor to manage the runnables
	super, err := supervisor.New(
		supervisor.WithLogHandler(logger.Handler()),
		supervisor.WithRunnables(coreRunner, runner),
		supervisor.WithContext(ctx),
		supervisor.WithReloadProvider(configProvider),
	)
	require.NoError(t, err, "Failed to create supervisor")

	// Start the supervisor in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- super.Run()
	}()

	// Wait briefly to ensure server starts up
	time.Sleep(500 * time.Millisecond)

	// Check for early errors
	select {
	case err := <-errCh:
		require.NoError(t, err, "Server failed to start")
	default:
		// No errors, continue with test
	}

	// Create an HTTP client for testing
	httpClient := &http.Client{Timeout: 2 * time.Second}

	// Test the initial echo endpoint
	t.Run("Initial echo endpoint responds", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d/echo", httpPort)

		// Try a few times to connect to the endpoint as it might take a moment to start
		var resp *http.Response
		var err error
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		require.NoError(t, err, "Failed to connect to echo endpoint")
		require.NotNil(t, resp, "Response should not be nil")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/echo", echoResp["path"], "Wrong path")
	})

	// Update the config file with a new route
	updatedConfigContent := fmt.Sprintf(`
version = "v1"

[logging]
level = "debug"
format = "text"

[[listeners]]
id = "http_listener"
address = "%s"
type = "http"

[listeners.http]
read_timeout = "1s"
write_timeout = "1s"
idle_timeout = "1s"
drain_timeout = "1s"

[[endpoints]]
id = "echo_endpoint"
listener_id = "http_listener"

[[endpoints.routes]]
app_id = "echo_app"
[endpoints.routes.http]
path_prefix = "/echo"

[[endpoints.routes]]
app_id = "echo_app"
[endpoints.routes.http]
path_prefix = "/new-path"

[[apps]]
id = "echo_app"
type = "echo"
[apps.echo]
response = "This is a test echo response"
`, httpAddr)

	err = os.WriteFile(configPath, []byte(updatedConfigContent), 0o644)
	require.NoError(t, err, "Failed to write updated config file")

	// Load the updated config
	updatedCfg, err := config.NewConfig(configPath)
	require.NoError(t, err, "Failed to load updated config from file")

	// Update the config provider
	configProvider.updateConfig(updatedCfg)

	// Simulate a SIGHUP by directly triggering the reload
	t.Log("Triggering reload...")
	super.ReloadAll()

	// Sleep to give time for reload to take effect
	time.Sleep(500 * time.Millisecond)

	// Test the new route
	t.Run("New route responds after reload", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d/new-path", httpPort)

		// Try a few times as the reload might take a moment
		var resp *http.Response
		var err error
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		require.NoError(t, err, "Failed to connect to new endpoint")
		require.NotNil(t, resp, "Response should not be nil")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/new-path", echoResp["path"], "Wrong path")
	})

	// Shutdown the server
	cancel()

	// Wait for server to shut down
	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("Server shutdown with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("Server shutdown timed out")
	}

	// Log buffer contents for debugging
	t.Logf("Logs:\n%s", logBuf.String())
}

// testConfigProvider is a helper for testing that provides config and implements ReloadProvider
type testConfigProvider struct {
	config       *config.Config
	reloadCh     chan struct{}
	autoReloadCh bool
}

func newTestConfigProvider(initialConfig *config.Config) *testConfigProvider {
	return &testConfigProvider{
		config:       initialConfig,
		reloadCh:     make(chan struct{}, 1),
		autoReloadCh: true,
	}
}

func (p *testConfigProvider) getConfig() config.Config {
	if p.config == nil {
		return config.Config{Version: config.VersionLatest}
	}
	return *p.config
}

func (p *testConfigProvider) updateConfig(newConfig *config.Config) {
	p.config = newConfig
	if p.autoReloadCh {
		// Send reload notification
		select {
		case p.reloadCh <- struct{}{}:
			// Notification sent
		default:
			// Channel full, skip
		}
	}
}

func (p *testConfigProvider) GetReloadTrigger() <-chan struct{} {
	return p.reloadCh
}
