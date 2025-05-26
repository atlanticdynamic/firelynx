//go:build e2e
// +build e2e

package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	serverCmd "github.com/atlanticdynamic/firelynx/cmd/firelynx/server"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/atlanticdynamic/firelynx/internal/config/loader/toml"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TemplateData contains values to substitute in server config templates
type TemplateData struct {
	HTTPAddr string
	GRPCAddr string
}

// runServerWithConfig starts a server with the given configuration file path and gRPC address
// and returns a cleanup function.
func runServerWithConfig(
	t *testing.T,
	ctx context.Context,
	configPath, grpcAddr string,
) (context.CancelFunc, error) {
	t.Helper()
	// Create a logger that writes to the test's log with DEBUG level
	var logBuf bytes.Buffer
	handlerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	testHandler := slog.NewTextHandler(&logBuf, handlerOptions)
	logger := slog.New(testHandler)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		err := serverCmd.Run(ctx, logger, configPath, grpcAddr)
		errCh <- err
	}()

	// Wait briefly to ensure server starts up
	select {
	case err := <-errCh:
		cancel() // Clean up context if server failed to start
		return nil, fmt.Errorf("server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server appears to be starting successfully
	}

	// Return a cleanup function that will shut down the server
	cleanup := func() {
		t.Log("Shutting down server...")
		cancel()

		// Wait for server to shut down with timeout
		select {
		case err := <-errCh:
			if err != nil {
				t.Logf("Server shutdown with error: %v", err)
			} else {
				t.Log("Server shutdown successfully")
			}
		case <-time.After(2 * time.Second):
			t.Log("Server shutdown timed out")
		}

		// Log the server output
		t.Logf("Server logs:\n%s", logBuf.String())
	}

	return cleanup, nil
}

// writeConfigFile writes a config file from a template with the given data
func writeConfigFile(t *testing.T, templatePath, outputPath string, data TemplateData) {
	t.Helper()
	configContent, err := ProcessTemplate(templatePath, data)
	require.NoError(t, err, "Failed to process template")

	// Debug print the processed template
	// t.Logf("Processed template content for %s:\n%s", templatePath, string(configContent))

	err = os.WriteFile(outputPath, configContent, 0o644)
	require.NoError(t, err, "Failed to write config file")

	// Debug: Log the config content
	t.Logf("Written config file %s with content:\n%s", outputPath, string(configContent))
}

// waitForHTTPEndpoint checks if an HTTP endpoint is responding and retries until it succeeds or times out
func waitForHTTPEndpoint(t *testing.T, url string, timeout, retryInterval time.Duration) bool {
	t.Helper()
	httpClient := &http.Client{Timeout: 2 * time.Second}

	return assert.Eventually(t, func() bool {
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

		return resp.StatusCode == http.StatusOK
	}, timeout, retryInterval, "Endpoint never became available: %s", url)
}

// createTempConfig creates a temporary directory and configures it with the given template
func createTempConfig(t *testing.T, templatePath string, data TemplateData) (string, string) {
	t.Helper()
	// Create a temporary directory for the config file
	tempDir := t.TempDir()

	// Use test name in config file for easier identification in logs
	configPath := filepath.Join(tempDir, fmt.Sprintf("%s.toml", t.Name()))

	// Write the configuration file
	writeConfigFile(t, templatePath, configPath, data)

	return tempDir, configPath
}

// getTestHTTPAndGRPCAddresses returns test HTTP and gRPC addresses using random ports
func getTestHTTPAndGRPCAddresses(t *testing.T) (string, string) {
	t.Helper()
	httpPort := testutil.GetRandomPort(t)
	grpcPort := testutil.GetRandomPort(t)

	httpAddr := fmt.Sprintf(":%d", httpPort)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	return httpAddr, grpcAddr
}

// loadConfigFromTemplate loads a configuration from a template file with the given data
func loadConfigFromTemplate(t *testing.T, templatePath string, data TemplateData) loader.Loader {
	t.Helper()
	configContent, err := ProcessTemplate(templatePath, data)
	require.NoError(t, err, "Failed to process template")

	return toml.NewTomlLoader(configContent)
}

// sendHUPSignalToProcess sends a SIGHUP signal to trigger config reload
// The go-supervisor will handle the signal and call Reload() on all Reloadable components
func sendHUPSignalToProcess(t *testing.T) {
	t.Helper()
	// Get the current process PID (which includes the test and the server)
	pid := os.Getpid()

	// Send SIGHUP to the current process, which will be handled by go-supervisor
	proc, err := os.FindProcess(pid)
	require.NoError(t, err, "Failed to find process")

	t.Logf("Sending SIGHUP signal to process %d to trigger config reload", pid)
	err = proc.Signal(syscall.SIGHUP)
	require.NoError(t, err, "Failed to send SIGHUP signal")
}
