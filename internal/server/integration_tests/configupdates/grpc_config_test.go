//go:build integration
// +build integration

package configupdates

import (
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/echo_app.toml
var echoAppTOML string

//go:embed testdata/route_v1_v2.toml
var multiRouteTOML string

//go:embed testdata/duplicate_endpoints.toml
var duplicateEndpointsTOML string

// createTempConfigFile creates a temporary config file from embedded TOML content
func createTempConfigFile(t *testing.T, baseContent string, httpPort int) string {
	t.Helper()

	// Replace the template port placeholder with the actual port
	content := strings.ReplaceAll(baseContent, "{{PORT}}", fmt.Sprintf("%d", httpPort))

	// Create a temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	return configPath
}

// TestGRPCConfigServiceHTTPIntegration tests gRPC config service integration using existing TOML configs and client
func TestGRPCConfigServiceHTTPIntegration(t *testing.T) {
	ctx := t.Context()

	// Create transaction storage and saga orchestrator
	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	// Create HTTP runner
	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	// Create transaction siphon
	txSiphon := make(chan *transaction.ConfigTransaction)

	// Create gRPC config service runner
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Register HTTP runner with orchestrator
	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	// Start both runners
	httpErrCh := make(chan error, 1)
	go func() {
		httpErrCh <- httpRunner.Run(ctx)
	}()

	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(ctx)
	}()

	// Wait for both runners to start
	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "HTTP runner should start")

	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should start")

	// Process transactions from siphon
	go func() {
		for tx := range txSiphon {
			if tx == nil {
				continue
			}
			// Process transaction through saga orchestrator
			if err := saga.ProcessTransaction(ctx, tx); err != nil {
				t.Logf("Failed to process transaction: %v", err)
			}
		}
	}()

	// Create client
	testClient := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Wait for gRPC server to be ready by checking if we can connect
	assert.Eventually(t, func() bool {
		_, err := testClient.GetConfig(ctx)
		// GetConfig may return an error if no config has been set yet, but connection should work
		// We're mainly testing that the gRPC service is listening and responding
		return err == nil || strings.Contains(err.Error(), "no configuration") ||
			strings.Contains(err.Error(), "not found")
	}, 5*time.Second, 200*time.Millisecond, "gRPC server should be ready")

	// Test using embedded TOML configs with routes and apps
	t.Run("Echo app configuration via client", func(t *testing.T) {
		httpPort := testutil.GetRandomPort(t)

		// Create temporary config file from embedded TOML
		configPath := createTempConfigFile(t, echoAppTOML, httpPort)

		// Use client to apply config from file path
		err := testClient.ApplyConfigFromPath(ctx, configPath)
		require.NoError(t, err, "Failed to apply config via client")

		// Test HTTP endpoint
		url := fmt.Sprintf("http://localhost:%d/echo", httpPort)
		httpClient := &http.Client{Timeout: 2 * time.Second}

		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(url)
			if err != nil {
				t.Logf("HTTP request failed: %v", err)
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				t.Logf("Received non-OK status: %d", httpResp.StatusCode)
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				t.Logf("Failed to read response body: %v", err)
				return false
			}

			responseText := string(body)
			t.Logf("Received response via client: %s", responseText)

			return strings.Contains(responseText, "Echo says: Hello!")
		}, 5*time.Second, 250*time.Millisecond, "HTTP endpoint should work via client config")
	})

	// Test using the client with loader interface
	t.Run("Config via loader interface", func(t *testing.T) {
		httpPort := testutil.GetRandomPort(t)

		// Create temporary config file from embedded multi-route TOML
		configPath := createTempConfigFile(t, multiRouteTOML, httpPort)

		// Create loader and use client.ApplyConfig
		configLoader, err := loader.NewLoaderFromFilePath(configPath)
		require.NoError(t, err)

		err = testClient.ApplyConfig(ctx, configLoader)
		require.NoError(t, err, "Failed to apply config via loader")

		// Test both routes
		httpClient := &http.Client{Timeout: 2 * time.Second}

		// Test /v1
		v1Url := fmt.Sprintf("http://localhost:%d/v1", httpPort)
		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(v1Url)
			if err != nil {
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				return false
			}

			return strings.Contains(string(body), "V1: Response")
		}, 5*time.Second, 250*time.Millisecond, "/v1 should work")

		// Test /v2
		v2Url := fmt.Sprintf("http://localhost:%d/v2", httpPort)
		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(v2Url)
			if err != nil {
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				return false
			}

			return strings.Contains(string(body), "V2: Response")
		}, 5*time.Second, 250*time.Millisecond, "/v2 should work")
	})

	// Test sequential reconfigurations of the same listener (mimicking e2e test pattern)
	t.Run("Sequential reconfigurations same listener", func(t *testing.T) {
		httpPort := testutil.GetRandomPort(t)

		// Step 1: Apply single route config
		configPath1 := createTempConfigFile(t, echoAppTOML, httpPort)
		configLoader1, err := loader.NewLoaderFromFilePath(configPath1)
		require.NoError(t, err)

		err = testClient.ApplyConfig(ctx, configLoader1)
		require.NoError(t, err, "Failed to apply single route config")

		// Test the single route works
		echoUrl := fmt.Sprintf("http://localhost:%d/echo", httpPort)
		httpClient := &http.Client{Timeout: 2 * time.Second}

		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(echoUrl)
			if err != nil {
				t.Logf("Single route request failed: %v", err)
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				return false
			}

			return strings.Contains(string(body), "Echo says: Hello!")
		}, 5*time.Second, 250*time.Millisecond, "Single route should work first")

		// Step 2: Apply multi-route config to SAME listener (same port)
		configPath2 := createTempConfigFile(t, multiRouteTOML, httpPort)
		configLoader2, err := loader.NewLoaderFromFilePath(configPath2)
		require.NoError(t, err)

		err = testClient.ApplyConfig(ctx, configLoader2)
		require.NoError(t, err, "Failed to apply multi-route config to same listener")

		// Test that both routes work on the same listener after reconfiguration
		v1Url := fmt.Sprintf("http://localhost:%d/v1", httpPort)
		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(v1Url)
			if err != nil {
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				return false
			}

			return strings.Contains(string(body), "V1: Response")
		}, 5*time.Second, 250*time.Millisecond, "/v1 should work after reconfiguration")

		v2Url := fmt.Sprintf("http://localhost:%d/v2", httpPort)
		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(v2Url)
			if err != nil {
				return false
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return false
			}

			body, err := io.ReadAll(httpResp.Body)
			if err != nil {
				return false
			}

			return strings.Contains(string(body), "V2: Response")
		}, 5*time.Second, 250*time.Millisecond, "/v2 should work after reconfiguration")

		// Verify old route is no longer available (replaced by new config)
		assert.Eventually(t, func() bool {
			httpResp, err := httpClient.Get(echoUrl)
			if err != nil {
				// Connection refused is expected since the route was replaced
				return true
			}
			defer httpResp.Body.Close()
			// Route should return 404 since it's not in the new config
			return httpResp.StatusCode == http.StatusNotFound
		}, 2*time.Second, 250*time.Millisecond, "Old /echo route should be removed after reconfiguration")
	})

	// Cleanup
	cfgServiceRunner.Stop()
	httpRunner.Stop()

	assert.Eventually(t, func() bool {
		return !cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should stop")

	assert.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "HTTP runner should stop")
}

// TestValidateConfigIntegration tests the ValidateConfig RPC endpoint
func TestValidateConfigIntegration(t *testing.T) {
	ctx := t.Context()

	// Create transaction siphon
	txSiphon := make(chan *transaction.ConfigTransaction)

	// Create gRPC config service runner
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Start config service
	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(ctx)
	}()

	// Wait for config service to start
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should start")

	// Create client
	testClient := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Wait for gRPC server to be ready
	assert.Eventually(t, func() bool {
		_, err := testClient.GetConfig(ctx)
		return err == nil || strings.Contains(err.Error(), "no configuration") ||
			strings.Contains(err.Error(), "not found")
	}, 5*time.Second, 200*time.Millisecond, "gRPC server should be ready")

	t.Run("valid_config_passes_validation", func(t *testing.T) {
		httpPort := testutil.GetRandomPort(t)

		configPath := createTempConfigFile(t, echoAppTOML, httpPort)

		configLoader, err := loader.NewLoaderFromFilePath(configPath)
		require.NoError(t, err)

		cfg, err := configLoader.LoadProto()
		require.NoError(t, err)

		isValid, validationErr := testClient.ValidateConfig(ctx, cfg)
		require.True(t, isValid)
		require.NoError(t, validationErr)

		select {
		case tx := <-txSiphon:
			t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("invalid_config_fails_validation", func(t *testing.T) {
		httpPort := testutil.GetRandomPort(t)

		configPath := createTempConfigFile(t, duplicateEndpointsTOML, httpPort)

		configLoader, err := loader.NewLoaderFromFilePath(configPath)
		require.NoError(t, err)

		cfg, err := configLoader.LoadProto()
		require.NoError(t, err)

		isValid, validationErr := testClient.ValidateConfig(ctx, cfg)
		require.False(t, isValid)
		require.Error(t, validationErr)
		assert.ErrorIs(t, validationErr, client.ErrConfigRejected)

		select {
		case tx := <-txSiphon:
			t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
		case <-time.After(100 * time.Millisecond):
		}
	})

	// Cleanup
	cfgServiceRunner.Stop()

	assert.Eventually(t, func() bool {
		return !cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should stop")
}
