//go:build e2e
// +build e2e

// Package server provides end-to-end tests for the firelynx server commands.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		url := fmt.Sprintf("http://127.0.0.1:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 15*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		// Read and verify the plain text response
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseText := string(body[:n])

		assert.Contains(
			t,
			responseText,
			"This is a test echo response",
			"Response should contain configured echo text",
		)
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

	// First test: apply single route config (this is the failing case)
	singleRouteLoader := loadConfigFromTemplate(t, "testdata/config.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Send configuration to the server
	err = testClient.ApplyConfig(ctx, singleRouteLoader)
	require.NoError(t, err, "Failed to apply single route configuration")

	// Extract HTTP port from address for URL construction
	httpPort := httpAddr[1:] // Remove leading colon

	// Create an HTTP client
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Test the single route
	t.Run("Single route responds after gRPC config update", func(t *testing.T) {
		url := fmt.Sprintf("http://127.0.0.1:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 15*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status")

		// Read and verify the plain text response
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseText := string(body[:n])

		assert.Contains(
			t,
			responseText,
			"This is a test echo response",
			"Response should contain configured echo text",
		)
	})

	// Wait a moment for the first configuration to fully complete in httpcluster
	time.Sleep(1 * time.Second)

	// Now apply the same single route configuration again to test if it's a "first config" issue
	t.Log("Re-applying the single route configuration...")
	err = testClient.ApplyConfig(ctx, singleRouteLoader)
	require.NoError(t, err, "Failed to re-apply single route configuration")

	// Test the single route again after re-application
	t.Run("Single route responds after re-application", func(t *testing.T) {
		url := fmt.Sprintf("http://127.0.0.1:%s/echo", httpPort)

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

		// Read and verify the plain text response
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseText := string(body[:n])

		assert.Contains(
			t,
			responseText,
			"This is a test echo response",
			"Response should contain configured echo text",
		)
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
		url := fmt.Sprintf("http://127.0.0.1:%s/echo", httpPort)

		// Wait for the endpoint to become available
		waitForHTTPEndpoint(t, url, 15*time.Second, 500*time.Millisecond)

		// Make a successful request once we know it's available
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err, "Failed to execute HTTP request")
		defer resp.Body.Close()

		// Read and verify the plain text response
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseText := string(body[:n])

		assert.Contains(
			t,
			responseText,
			"This is a test echo response",
			"Response should contain configured echo text",
		)
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

	// Debug: Check the original file content before replacing
	originalContent, err := os.ReadFile(configPath)
	require.NoError(t, err, "Failed to read original config file")
	t.Logf(
		"Original config file contains routes for path prefixes: %s",
		findPathPrefixes(string(originalContent)),
	)

	// Now replace the original file by moving the new one
	// This ensures a complete file replacement which is more reliably detected
	err = os.Rename(newConfigPath, configPath)
	require.NoError(t, err, "Failed to replace config file")

	// Debug: Verify the file was updated correctly
	updatedFileContent, err := os.ReadFile(configPath)
	require.NoError(t, err, "Failed to read updated config file")
	t.Logf(
		"Updated config file contains routes for path prefixes: %s",
		findPathPrefixes(string(updatedFileContent)),
	)

	// Ensure the replaced file is synced to disk
	syncFileToStorage(t, configPath)

	// Give a brief moment for filesystem to settle
	time.Sleep(100 * time.Millisecond)

	sendHUPSignalToProcess(t)

	// Test the new route after sending SIGHUP to trigger reload
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
				t.Logf("Received non-OK status: %d", resp.StatusCode)
				return false
			}

			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			responseText := string(body[:n])

			t.Logf("Response from new-path: %s", responseText)

			return strings.Contains(responseText, "This is a test echo response")
		}, 10*time.Second, 500*time.Millisecond, "New route never became available after config reload")
	})
}

// findPathPrefixes searches for path_prefix entries in a TOML config string
func findPathPrefixes(content string) string {
	var paths []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "path_prefix") {
			paths = append(paths, strings.TrimSpace(line))
		}
	}
	return strings.Join(paths, ", ")
}

// syncFileToStorage ensures file changes are flushed to stable storage
func syncFileToStorage(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Logf("Failed to open file for sync: %v", err)
		return
	}

	if err = file.Sync(); err != nil {
		t.Logf("Failed to sync file: %v", err)
	}

	if err = file.Close(); err != nil {
		t.Logf("Failed to close file for sync: %v", err)
	}

	// Also sync the parent directory to ensure file metadata is updated
	dirPath := filepath.Dir(path)
	dir, err := os.Open(dirPath)
	if err != nil {
		t.Logf("Failed to open directory for sync: %v", err)
		return
	}

	if err = dir.Sync(); err != nil {
		t.Logf("Failed to sync directory: %v", err)
	}

	if err = dir.Close(); err != nil {
		t.Logf("Failed to close directory for sync: %v", err)
	}
}
