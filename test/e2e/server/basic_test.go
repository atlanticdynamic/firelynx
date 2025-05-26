//go:build e2e
// +build e2e

package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicServerStartup is a minimal test that just verifies the server can start up
// with a known good configuration file from the examples package
func TestBasicServerStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping basic server startup test in short mode")
	}

	// Create a root context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get random ports for HTTP
	httpAddr, _ := getTestHTTPAndGRPCAddresses(t)

	// Create a temporary directory and write the config file using the template
	_, configPath := createTempConfig(t, "testdata/config.tmpl", TemplateData{
		HTTPAddr: httpAddr,
	})

	// Debug: Log the HTTP address we're using
	t.Logf("Test expects HTTP server on address: %s", httpAddr)

	// Start the server
	cleanup, err := runServerWithConfig(t, ctx, configPath, "")
	require.NoError(t, err, "Failed to start firelynx server")
	defer cleanup()

	// For basic testing, we need to extract the configured HTTP port from the minimal config
	// We'll make a simple request to verify the server is operational

	// Create an HTTP client
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Test that at least one endpoint is responding
	t.Run("Server responds to requests", func(t *testing.T) {
		// This is a simple check to ensure the server is running
		// The actual endpoint will depend on the minimal_config.toml content
		// Get the port from httpAddr which is in the format ":port"
		port := httpAddr[1:] // Remove leading colon
		url := fmt.Sprintf("http://localhost:%s/echo", port)

		// Wait for the endpoint to become available
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

			// If we get any response (even an error response),
			// the server is operational
			return true
		}, 5*time.Second, 500*time.Millisecond, "Server never became available")

		// Verify the server is still running
		t.Log("Server is running and responding to requests")
	})
}
