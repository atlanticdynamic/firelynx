//go:build e2e

package server

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Embed test configuration files
var (
	//go:embed testdata/basic_config.toml
	basicConfigTOML string

	//go:embed testdata/grpc_config.toml
	grpcConfigTOML string

	//go:embed testdata/initial_config.toml
	initialConfigTOML string
)

// TestServerWithConfigFile tests loading a basic TOML config file and starting the server
func TestServerWithConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Create a temp directory for the config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.toml")

	// Get a free port for HTTP
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Replace the port in the basic config
	configContent := strings.Replace(basicConfigTOML, ":8080", httpAddr, 1)

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err, "Failed to write config file")

	// Create a logger
	logging.SetupLogger("debug")
	logger := slog.Default()

	// Start the server
	serverCtx, serverCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer serverCancel()
	errCh := make(chan error, 1)
	go func() {
		err := Run(serverCtx, logger, configPath, "")
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Test that the echo endpoint responds
	httpClient := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://localhost%s/test", httpAddr)

	// Wait for endpoint to become available
	assert.Eventually(t, func() bool {
		resp, err := httpClient.Get(url)
		if err != nil {
			return false
		}
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "Echo endpoint should become available")

	// Make a successful request and verify response
	resp, err := httpClient.Get(url)
	require.NoError(t, err, "Should get response from echo endpoint")
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")
	responseText := string(body)
	assert.Contains(
		t,
		responseText,
		"Hello from test",
		"Response should contain configured echo text",
	)

	// Cancel the server
	serverCancel()

	// Wait for server to shut down
	assert.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			require.NoError(t, err, "Server should shut down cleanly")
			return true
		default:
			return false
		}
	}, 1*time.Minute, 100*time.Millisecond, "Server shutdown timed out")
}

// TestServerWithGRPCConfig tests starting server with gRPC API and sending config via client
func TestServerWithGRPCConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	// Get free ports for gRPC and HTTP
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Start the server with only gRPC
	serverCtx, serverCancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)
	go func() {
		err := Run(serverCtx, logger, "", grpcAddr)
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for gRPC server to be ready
	assert.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", grpcAddr)
		if err != nil {
			return false
		}
		assert.NoError(t, conn.Close())
		return true
	}, 5*time.Second, 100*time.Millisecond, "gRPC server should be listening")

	// Create a client
	c := client.New(client.Config{
		ServerAddr: grpcAddr,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	})

	// Wait for gRPC service to be ready
	assert.Eventually(t, func() bool {
		_, err := c.GetConfig(ctx)
		return err == nil
	}, 5*time.Second, 200*time.Millisecond, "gRPC service should become ready")

	// Replace the port in the config
	configContent := strings.Replace(grpcConfigTOML, ":8081", httpAddr, 1)

	// Create a temporary file to send the config
	tempFile, err := os.CreateTemp("", "grpc_config_*.toml")
	require.NoError(t, err, "Failed to create temp file")
	defer func() { assert.NoError(t, os.Remove(tempFile.Name())) }()

	_, err = tempFile.Write([]byte(configContent))
	require.NoError(t, err, "Failed to write temp file")
	assert.NoError(t, tempFile.Close())

	// Apply the config
	err = c.ApplyConfigFromPath(ctx, tempFile.Name())
	require.NoError(t, err, "Should send config successfully")

	// Test that the echo endpoint responds
	httpClient := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://localhost%s/grpc-test", httpAddr)

	// Wait for endpoint to become available
	assert.Eventually(t, func() bool {
		resp, err := httpClient.Get(url)
		if err != nil {
			return false
		}
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 8*time.Second, 200*time.Millisecond, "Echo endpoint should become available")

	// Verify response content
	resp, err := httpClient.Get(url)
	require.NoError(t, err, "Should get response from echo endpoint")
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")
	responseText := string(body)
	assert.Contains(
		t,
		responseText,
		"Hello from gRPC config",
		"Response should contain configured echo text",
	)

	// Shutdown the server
	serverCancel()

	// Wait for clean shutdown
	assert.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			require.NoError(t, err, "Server should shut down cleanly")
			return true
		default:
			return false
		}
	}, 1*time.Minute, 100*time.Millisecond, "Server shutdown timed out")
}

// TestServerWithFileAndGRPC tests loading initial config from file then updating via gRPC
func TestServerWithFileAndGRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	// Create a temp directory for the config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "initial_config.toml")

	// Get free ports
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Replace the port in the initial config
	configContent := strings.Replace(initialConfigTOML, ":8082", httpAddr, 1)

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err, "Failed to write initial config file")

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Start the server with both file and gRPC
	serverCtx, serverCancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)
	go func() {
		err := Run(serverCtx, logger, configPath, grpcAddr)
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for gRPC server to be ready
	assert.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", grpcAddr)
		if err != nil {
			return false
		}
		assert.NoError(t, conn.Close())
		return true
	}, 5*time.Second, 100*time.Millisecond, "gRPC server should be listening")

	// Create a client
	c := client.New(client.Config{
		ServerAddr: grpcAddr,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	})

	// Wait for gRPC service to be ready and get initial config
	var currentCfg *config.Config
	assert.Eventually(t, func() bool {
		protoCfg, err := c.GetConfig(ctx)
		if err != nil {
			return false
		}
		currentCfg, err = config.NewFromProto(protoCfg)
		return err == nil && len(currentCfg.Endpoints) > 0
	}, 5*time.Second, 200*time.Millisecond, "Should get initial config with endpoints")

	require.NotNil(t, currentCfg, "Config should not be nil")
	require.NotEmpty(t, currentCfg.Endpoints, "Should have at least one endpoint")

	// Test initial endpoint
	httpClient := &http.Client{Timeout: 2 * time.Second}
	initialURL := fmt.Sprintf("http://localhost%s/initial", httpAddr)

	assert.Eventually(t, func() bool {
		resp, err := httpClient.Get(initialURL)
		if err != nil {
			return false
		}
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 200*time.Millisecond, "Initial endpoint should be available")

	// Test that we can also use gRPC to get the current config
	retrievedProto, err := c.GetConfig(ctx)
	require.NoError(t, err, "Should be able to get config via gRPC")

	retrievedCfg, err := config.NewFromProto(retrievedProto)
	require.NoError(t, err, "Should be able to convert retrieved config")

	// Verify the retrieved config matches what we expect
	assert.Equal(
		t,
		currentCfg.Apps.Len(),
		retrievedCfg.Apps.Len(),
		"Retrieved config should have same number of apps",
	)
	assert.Len(
		t,
		retrievedCfg.Endpoints, len(currentCfg.Endpoints),
		"Retrieved config should have same number of endpoints",
	)

	// Shutdown the server
	serverCancel()

	// Wait for clean shutdown
	assert.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			require.NoError(t, err, "Server should shut down cleanly")
			return true
		default:
			return false
		}
	}, 1*time.Minute, 100*time.Millisecond, "Server shutdown timed out")
}

// TestConfigFileReload tests reloading configuration by sending SIGHUP
func TestConfigFileReload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	// Create a temp directory for the config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "reload_config.toml")

	// Get a free port for HTTP
	httpPort := testutil.GetRandomPort(t)
	httpAddr := fmt.Sprintf(":%d", httpPort)

	// Write initial config
	initialConfig := strings.Replace(initialConfigTOML, ":8082", httpAddr, 1)
	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	require.NoError(t, err, "Failed to write initial config file")

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Start the server
	serverCtx, serverCancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)
	go func() {
		err := Run(serverCtx, logger, configPath, "")
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Test initial endpoint
	httpClient := &http.Client{Timeout: 2 * time.Second}
	initialURL := fmt.Sprintf("http://localhost%s/initial", httpAddr)

	assert.Eventually(t, func() bool {
		resp, err := httpClient.Get(initialURL)
		if err != nil {
			return false
		}
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 200*time.Millisecond, "Initial endpoint should be available")

	// Update config file to add new route (simulate file update)
	// Work directly with TOML since file reloads are atomic
	updatedConfig := initialConfig + `

[[endpoints.routes]]
app_id = "new_app"
[endpoints.routes.http]
path_prefix = "/new-path"

[[apps]]
id = "new_app"
type = "echo"
[apps.echo]
response = "New path response"`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
	require.NoError(t, err, "Failed to update config file")

	// Send SIGHUP to trigger reload
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err, "Failed to find process")
	err = proc.Signal(syscall.SIGHUP)
	require.NoError(t, err, "Failed to send SIGHUP signal")

	// Test new endpoint becomes available
	newURL := fmt.Sprintf("http://localhost%s/new-path", httpAddr)

	assert.Eventually(t, func() bool {
		resp, err := httpClient.Get(newURL)
		if err != nil {
			return false
		}
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		return resp.StatusCode == http.StatusOK
	}, 8*time.Second, 200*time.Millisecond, "New endpoint should become available after reload")

	// Verify new endpoint response
	resp, err := httpClient.Get(newURL)
	require.NoError(t, err, "Should get response from new endpoint")
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")
	responseText := string(body)
	assert.Contains(
		t,
		responseText,
		"New path response",
		"Response should contain new path echo text",
	)

	// Shutdown the server
	serverCancel()

	// Wait for clean shutdown
	assert.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			require.NoError(t, err, "Server should shut down cleanly")
			return true
		default:
			return false
		}
	}, 1*time.Minute, 100*time.Millisecond, "Server shutdown timed out")
}

// TestServerRequiresConfigSource verifies that the server returns an error
// when neither config file nor gRPC address is provided
func TestServerRequiresConfigSource(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	err := Run(ctx, logger, "", "")
	require.Error(t, err, "Server should require at least one config source")
	assert.ErrorContains(t, err, "no configuration source specified")
}

// TestNewConfigFromBytes validates that we can create configs from embedded bytes
func TestNewConfigFromBytes(t *testing.T) {
	// Test with the basic config
	cfg, err := config.NewConfigFromBytes([]byte(basicConfigTOML))
	require.NoError(t, err, "Should create config from bytes")
	require.NoError(t, cfg.Validate(), "Should validate config")
	assert.Equal(t, "v1", cfg.Version)
	assert.Len(t, cfg.Listeners, 1)
	assert.Len(t, cfg.Endpoints, 1)
	assert.Equal(t, 1, cfg.Apps.Len())

	// Test conversion to proto and back
	proto := cfg.ToProto()
	assert.NotNil(t, proto)

	// Convert back to domain
	cfg2, err := config.NewFromProto(proto)
	require.NoError(t, err, "Should convert from proto")
	assert.True(t, cfg.Equals(cfg2), "Configs should be equal after round-trip")
}
