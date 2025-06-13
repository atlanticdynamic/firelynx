//go:build integration
// +build integration

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

//go:embed server/testdata/basic_config.toml
var validConfigContent string

//go:embed server/testdata/invalid_config.toml
var invalidConfigContent string

// createTempConfigFile creates a temporary config file with the given content
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	// Replace template port with a real port for valid configs
	if strings.Contains(content, "{{PORT}}") {
		port := testutil.GetRandomPort(t)
		content = strings.ReplaceAll(content, "{{PORT}}", fmt.Sprintf("%d", port))
	}

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	return configPath
}

func TestRenderConfigSummary(t *testing.T) {
	// Create a test config by loading the valid config content
	configPath := createTempConfigFile(t, validConfigContent)
	cfg, err := config.NewConfig(configPath)
	require.NoError(t, err)

	// Test the renderConfigSummary function
	summary := renderConfigSummary(configPath, cfg)

	// Verify the summary contains expected content
	assert.Contains(t, summary, "Config Summary:")
	assert.Contains(t, summary, "- Path: "+configPath)
	assert.Contains(t, summary, "- Version: v1")
	assert.Contains(t, summary, "- Listeners: 1")
	assert.Contains(t, summary, "- Endpoints: 1")
	assert.Contains(t, summary, "- Apps: 1")
	assert.Contains(t, summary, "Use --tree for a more detailed view of the config.")

	// Verify the format is correct (starts with newline, ends without newline)
	assert.True(t, strings.HasPrefix(summary, "\nConfig Summary:"))
	assert.True(t, strings.HasSuffix(summary, "Use --tree for a more detailed view of the config."))

	// Test with a config that has different counts
	t.Run("with_different_counts", func(t *testing.T) {
		// Create a config with specific known values using the collection types
		testCfg := &config.Config{
			Version: "v2",
			Listeners: listeners.ListenerCollection{
				{ID: "l1"}, {ID: "l2"}, {ID: "l3"}, // 3 listeners
			},
			Endpoints: endpoints.EndpointCollection{
				{ID: "e1"}, {ID: "e2"}, {ID: "e3"}, {ID: "e4"}, {ID: "e5"}, // 5 endpoints
			},
			Apps: apps.AppCollection{
				{ID: "a1"}, {ID: "a2"}, // 2 apps
			},
		}

		summary := renderConfigSummary("test-config.toml", testCfg)

		assert.Contains(t, summary, "- Path: test-config.toml")
		assert.Contains(t, summary, "- Version: v2")
		assert.Contains(t, summary, "- Listeners: 3")
		assert.Contains(t, summary, "- Endpoints: 5")
		assert.Contains(t, summary, "- Apps: 2")
	})

	t.Run("with_empty_config", func(t *testing.T) {
		// Test with empty collections
		emptyCfg := &config.Config{
			Version:   "v1",
			Listeners: listeners.ListenerCollection{},
			Endpoints: endpoints.EndpointCollection{},
			Apps:      apps.AppCollection{},
		}

		summary := renderConfigSummary("empty-config.toml", emptyCfg)

		assert.Contains(t, summary, "- Path: empty-config.toml")
		assert.Contains(t, summary, "- Version: v1")
		assert.Contains(t, summary, "- Listeners: 0")
		assert.Contains(t, summary, "- Endpoints: 0")
		assert.Contains(t, summary, "- Apps: 0")
	})
}

func TestValidateLocal(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		err := validateLocal(t.Context(), configPath, false)
		assert.NoError(t, err)
	})

	t.Run("invalid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, invalidConfigContent)

		err := validateLocal(t.Context(), configPath, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate ID")
	})

	t.Run("with_tree_view", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		err := validateLocal(t.Context(), configPath, true)
		assert.NoError(t, err)
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		err := validateLocal(t.Context(), "/path/that/does/not/exist.toml", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config")
	})
}

func TestValidateRemote(t *testing.T) {
	// Start a test gRPC server
	txSiphon := make(chan *transaction.ConfigTransaction)
	grpcPort := testutil.GetRandomPort(t)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
	require.NoError(t, err)

	// Start the server
	cfgServiceErrCh := make(chan error, 1)
	go func() {
		cfgServiceErrCh <- cfgServiceRunner.Run(t.Context())
	}()

	// Wait for server to start
	require.Eventually(t, func() bool {
		return cfgServiceRunner.IsRunning()
	}, time.Second, 10*time.Millisecond, "gRPC config service should start")

	// Cleanup function
	defer func() {
		cfgServiceRunner.Stop()
		assert.Eventually(t, func() bool {
			return !cfgServiceRunner.IsRunning()
		}, time.Second, 10*time.Millisecond, "gRPC config service should stop")
	}()

	t.Run("valid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		err := validateRemote(t.Context(), configPath, grpcAddr, false)
		assert.NoError(t, err)

		// Make sure no transaction was sent to the siphon for validation
		select {
		case tx := <-txSiphon:
			t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
		case <-time.After(100 * time.Millisecond):
			// Expected - no transaction should be sent
		}
	})

	t.Run("invalid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, invalidConfigContent)

		err := validateRemote(t.Context(), configPath, grpcAddr, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "remote validation failed")
		assert.Contains(t, err.Error(), "duplicate ID")

		// Make sure no transaction was sent to the siphon for validation
		select {
		case tx := <-txSiphon:
			t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
		case <-time.After(100 * time.Millisecond):
			// Expected - no transaction should be sent
		}
	})

	t.Run("with_tree_view", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		err := validateRemote(t.Context(), configPath, grpcAddr, true)
		assert.NoError(t, err)
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		err := validateRemote(t.Context(), "/path/that/does/not/exist.toml", grpcAddr, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("connection_error", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		// Use an invalid server address
		err := validateRemote(t.Context(), configPath, "localhost:99999", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "remote validation failed")
	})
}

// TestValidateRemoteShutdownTiming verifies that the gRPC server can shutdown quickly
// after validation operations, confirming that ValidateConfig doesn't create transactions
// that get stuck in non-terminal states during shutdown
func TestValidateAction(t *testing.T) {
	// Create test config files
	validConfigPath := createTempConfigFile(t, validConfigContent)
	invalidConfigPath := createTempConfigFile(t, invalidConfigContent)
	anotherConfigPath := createTempConfigFile(t, validConfigContent)

	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "with_positional_argument",
			args:      []string{"test", validConfigPath},
			wantError: false,
		},
		{
			name:      "with_config_flag",
			args:      []string{"test", "--config", validConfigPath},
			wantError: false,
		},
		{
			name:      "with_config_flag_short",
			args:      []string{"test", "-c", validConfigPath},
			wantError: false,
		},
		{
			name:      "config_flag_takes_precedence",
			args:      []string{"test", "--config", validConfigPath, anotherConfigPath},
			wantError: false,
		},
		{
			name:      "no_config_provided",
			args:      []string{"test"},
			wantError: true,
			errorMsg:  "config file path required",
		},
		{
			name:      "with_tree_flag",
			args:      []string{"test", "--config", validConfigPath, "--tree"},
			wantError: false,
		},
		{
			name:      "with_tree_flag_positional",
			args:      []string{"test", validConfigPath, "--tree"},
			wantError: false,
		},
		{
			name:      "invalid_config",
			args:      []string{"test", "--config", invalidConfigPath},
			wantError: true,
			errorMsg:  "duplicate ID",
		},
		{
			name:      "invalid_config_positional",
			args:      []string{"test", invalidConfigPath},
			wantError: true,
			errorMsg:  "duplicate ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{
				Name:   "test",
				Action: validateCmd.Action,
				Flags:  validateCmd.Flags,
			}

			err := cmd.Run(t.Context(), tt.args)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRemoteShutdownTiming(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		validationCount int
	}{
		{"single_validation_shutdown", 1},
		{"multiple_validations_shutdown", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Start a test gRPC server
			txSiphon := make(chan *transaction.ConfigTransaction)
			grpcPort := testutil.GetRandomPort(t)
			grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

			cfgServiceRunner, err := cfgservice.NewRunner(grpcAddr, txSiphon)
			require.NoError(t, err)

			// Start the server
			go func() {
				_ = cfgServiceRunner.Run(t.Context())
			}()

			// Wait for server to start
			require.Eventually(t, func() bool {
				return cfgServiceRunner.IsRunning()
			}, time.Second, 10*time.Millisecond, "gRPC config service should start")

			configPath := createTempConfigFile(t, validConfigContent)

			// Perform validations using modern range syntax
			for i := range tc.validationCount {
				err := validateRemote(t.Context(), configPath, grpcAddr, false)
				assert.NoError(err, "Validation %d should succeed", i+1)
			}

			// Verify no transactions were sent to siphon using assert.Never
			assert.Never(func() bool {
				<-txSiphon
				return true
			}, 50*time.Millisecond, 10*time.Millisecond, "ValidateConfig should not send transactions to siphon")

			// Measure shutdown time
			shutdownStart := time.Now()
			cfgServiceRunner.Stop()

			// Verify shutdown completes quickly using assert.Eventually
			assert.Eventually(func() bool {
				return !cfgServiceRunner.IsRunning()
			}, 2*time.Second, 10*time.Millisecond, "gRPC config service should stop")

			shutdownDuration := time.Since(shutdownStart)
			t.Logf(
				"Server shutdown after %d validation(s) completed in %v",
				tc.validationCount,
				shutdownDuration,
			)

			// Shutdown should complete within 1 second as requested
			assert.Less(
				shutdownDuration,
				1*time.Second,
				"Server should shutdown quickly after %d validation(s), but took %v",
				tc.validationCount,
				shutdownDuration,
			)
		})
	}
}
