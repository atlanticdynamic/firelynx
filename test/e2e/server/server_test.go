//go:build e2e
// +build e2e

// Package server provides end-to-end tests for the firelynx server commands.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigFileServer tests starting the server with a configuration file
func TestConfigFileServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a root context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get random ports for HTTP
	httpAddr, _ := getTestHTTPAndGRPCAddresses(t)

	// Create a temporary directory and write the config file
	_, configPath := createTempConfig(t, "testdata/config.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Start the server with the config file
	cleanup, err := runServerWithConfig(t, ctx, configPath, "")
	require.NoError(t, err, "Failed to start firelynx server")
	defer cleanup()

	// Extract HTTP port from address for URL construction
	httpPort := httpAddr[1:] // Remove leading colon

	// Create an HTTP client
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Test the echo endpoint
	t.Run("Echo endpoint responds", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 5*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		// Parse and verify the response
		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "GET", echoResp["method"], "Wrong HTTP method")
		assert.Equal(t, "/echo", echoResp["path"], "Wrong path")
	})
}

// TestGRPCServer tests starting the server with a gRPC listener and updating config
func TestGRPCServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a root context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get random ports for HTTP and gRPC
	httpAddr, grpcAddr := getTestHTTPAndGRPCAddresses(t)

	// Start the server with gRPC listener only
	cleanup, err := runServerWithConfig(t, ctx, "", grpcAddr)
	require.NoError(t, err, "Failed to start firelynx server")
	defer cleanup()

	// Create a client to use for checking server readiness
	testClient := client.New(client.Config{
		ServerAddr: grpcAddr,
	})

	// Wait for gRPC server to be ready by attempting to get config
	assert.Eventually(t, func() bool {
		_, err := testClient.GetConfig(ctx)
		if err != nil {
			t.Logf("Server not ready yet: %v", err)
			return false
		}
		return true
	}, 5*time.Second, 200*time.Millisecond, "gRPC server never became ready to accept requests")

	// Create the config
	configLoader := loadConfigFromTemplate(t, "testdata/config.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Send configuration to the server
	err = testClient.ApplyConfig(ctx, configLoader)
	require.NoError(t, err, "Failed to apply configuration")

	// Extract HTTP port from address for URL construction
	httpPort := httpAddr[1:] // Remove leading colon

	// Create an HTTP client
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Test the echo endpoint
	t.Run("Echo endpoint responds after gRPC config update", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 5*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		// Parse and verify the response
		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "GET", echoResp["method"], "Wrong HTTP method")
		assert.Equal(t, "/echo", echoResp["path"], "Wrong path")
	})

	// Now update the configuration with a new route
	t.Log("Updating configuration with new route...")
	updatedLoader := loadConfigFromTemplate(t, "testdata/config_with_new_route.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Send the updated configuration
	err = testClient.ApplyConfig(ctx, updatedLoader)
	require.NoError(t, err, "Failed to apply updated configuration")

	// Test the new route
	t.Run("New route responds after config update", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%s/new-path", httpPort)

		// Wait for the new route to become available
		waitForHTTPEndpoint(t, url, 5*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		// Parse and verify the response
		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/new-path", echoResp["path"], "Wrong path")
	})
}

// TestConfigFileReload tests reloading configuration by updating the config file
func TestConfigFileReload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a root context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get a free port for HTTP
	httpAddr, _ := getTestHTTPAndGRPCAddresses(t)

	// Create a temporary directory and write the config file
	tempDir, configPath := createTempConfig(t, "testdata/config.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Start the server with the config file
	cleanup, err := runServerWithConfig(t, ctx, configPath, "")
	require.NoError(t, err, "Failed to start firelynx server")
	defer cleanup()

	// Extract HTTP port from address for URL construction
	httpPort := httpAddr[1:] // Remove leading colon

	// Create an HTTP client
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Test the initial echo endpoint
	t.Run("Initial echo endpoint responds", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 5*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Parse and verify the response
		var echoResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&echoResp)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "echo_app", echoResp["app_id"], "Wrong app_id")
		assert.Equal(t, "/echo", echoResp["path"], "Wrong path")
	})

	// Update the config file with the new route
	t.Log("Updating config file...")
	updatedContent, err := ProcessTemplate("testdata/config_with_new_route.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})
	require.NoError(t, err, "Failed to process template")

	// Create a new file instead of updating the existing one
	// This is more likely to trigger filesystem notification on all platforms
	newConfigPath := filepath.Join(tempDir, "updated_config.toml")
	err = os.WriteFile(newConfigPath, updatedContent, 0o644)
	require.NoError(t, err, "Failed to write updated config file")

	// Ensure file is synced to disk
	syncFileToStorage(t, newConfigPath)

	// Now replace the original file by moving the new one
	// This ensures a complete file replacement which is more reliably detected
	err = os.Rename(newConfigPath, configPath)
	require.NoError(t, err, "Failed to replace config file")

	// Ensure the replaced file is synced to disk
	syncFileToStorage(t, configPath)

	// Test the new route (file watcher should detect changes)
	t.Run("New route responds after config reload", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%s/new-path", httpPort)

		// Wait for the new route to become available - use a longer timeout for file watching
		assert.Eventually(t, func() bool {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				return false
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				t.Logf("Request failed: %v", err)
				return false
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return false
			}

			var echoResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&echoResp); err != nil {
				return false
			}

			return echoResp["app_id"] == "echo_app" && echoResp["path"] == "/new-path"
		}, 10*time.Second, 500*time.Millisecond, "New route never became available after config reload")
	})
}

// syncFileToStorage ensures file changes are flushed to stable storage
func syncFileToStorage(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Logf("Failed to open file for sync: %v", err)
		return
	}
	err = file.Close()
	if err != nil {
		t.Logf("Failed to close file for sync: %v", err)
	}

	if err = file.Sync(); err != nil {
		t.Logf("Failed to sync file: %v", err)
	}

	// Also sync the parent directory to ensure file metadata is updated
	dirPath := filepath.Dir(path)
	dir, err := os.Open(dirPath)
	if err != nil {
		t.Logf("Failed to open directory for sync: %v", err)
		return
	}
	err = dir.Close()
	if err != nil {
		t.Logf("Failed to close directory for sync: %v", err)
	}

	if err = dir.Sync(); err != nil {
		t.Logf("Failed to sync directory: %v", err)
	}
}
