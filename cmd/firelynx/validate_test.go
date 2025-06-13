//go:build integration
// +build integration

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestValidateLocal(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		results := validateLocal(t.Context(), []string{configPath})
		assert.Len(t, results, 1)
		assert.True(t, results[0].Valid)
		assert.NoError(t, results[0].Error)
		assert.NotNil(t, results[0].Config)
		assert.Equal(t, configPath, results[0].Path)
		assert.False(t, results[0].Remote)
	})

	t.Run("invalid_config", func(t *testing.T) {
		configPath := createTempConfigFile(t, invalidConfigContent)

		results := validateLocal(t.Context(), []string{configPath})
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "duplicate ID")
		assert.Equal(t, configPath, results[0].Path)
		assert.False(t, results[0].Remote)
	})

	t.Run("multiple_configs", func(t *testing.T) {
		configPath1 := createTempConfigFile(t, validConfigContent)
		configPath2 := createTempConfigFile(t, validConfigContent)

		results := validateLocal(t.Context(), []string{configPath1, configPath2})
		assert.Len(t, results, 2)
		assert.True(t, results[0].Valid)
		assert.True(t, results[1].Valid)
		assert.NoError(t, results[0].Error)
		assert.NoError(t, results[1].Error)
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		results := validateLocal(t.Context(), []string{"/path/that/does/not/exist.toml"})
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "no such file or directory")
	})

	t.Run("canceled_context", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		// Create a canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		results := validateLocal(ctx, []string{configPath})
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "validation canceled")
		assert.Contains(t, results[0].Error.Error(), "context canceled")
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

		results := validateRemote(t.Context(), []string{configPath}, grpcAddr)
		assert.Len(t, results, 1)
		assert.True(t, results[0].Valid)
		assert.NoError(t, results[0].Error)
		assert.NotNil(t, results[0].Config)
		assert.Equal(t, configPath, results[0].Path)
		assert.True(t, results[0].Remote)

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

		results := validateRemote(t.Context(), []string{configPath}, grpcAddr)
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "remote validation failed")
		assert.Contains(t, results[0].Error.Error(), "duplicate ID")
		assert.Equal(t, configPath, results[0].Path)
		assert.True(t, results[0].Remote)

		// Make sure no transaction was sent to the siphon for validation
		select {
		case tx := <-txSiphon:
			t.Fatalf("ValidateConfig should not send transaction to siphon, but got: %v", tx)
		case <-time.After(100 * time.Millisecond):
			// Expected - no transaction should be sent
		}
	})

	t.Run("multiple_configs", func(t *testing.T) {
		configPath1 := createTempConfigFile(t, validConfigContent)
		configPath2 := createTempConfigFile(t, validConfigContent)

		results := validateRemote(t.Context(), []string{configPath1, configPath2}, grpcAddr)
		assert.Len(t, results, 2)
		assert.True(t, results[0].Valid)
		assert.True(t, results[1].Valid)
		assert.NoError(t, results[0].Error)
		assert.NoError(t, results[1].Error)
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		results := validateRemote(t.Context(), []string{"/path/that/does/not/exist.toml"}, grpcAddr)
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "no such file or directory")
	})

	t.Run("connection_error", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		// Use an invalid server address
		results := validateRemote(t.Context(), []string{configPath}, "localhost:99999")
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "remote validation failed")
	})

	t.Run("canceled_context", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfigContent)

		// Create a canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		results := validateRemote(ctx, []string{configPath}, grpcAddr)
		assert.Len(t, results, 1)
		assert.False(t, results[0].Valid)
		assert.Error(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "validation canceled")
		assert.Contains(t, results[0].Error.Error(), "context canceled")
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
			errorMsg:  "validation failed",
		},
		{
			name:      "invalid_config_positional",
			args:      []string{"test", invalidConfigPath},
			wantError: true,
			errorMsg:  "validation failed",
		},
		{
			name:      "multiple_configs_mixed",
			args:      []string{"test", validConfigPath, invalidConfigPath},
			wantError: true,
			errorMsg:  "validation failed",
		},
		{
			name:      "multiple_configs_all_valid",
			args:      []string{"test", validConfigPath, anotherConfigPath},
			wantError: false,
		},
		{
			name:      "with_quiet_flag",
			args:      []string{"test", "--quiet", validConfigPath},
			wantError: false,
		},
		{
			name:      "with_summary_flag",
			args:      []string{"test", "--summary", validConfigPath, invalidConfigPath},
			wantError: true,
			errorMsg:  "validation failed",
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
			for range tc.validationCount {
				results := validateRemote(t.Context(), []string{configPath}, grpcAddr)
				assert.Len(results, 1)
				assert.True(results[0].Valid)
				assert.NoError(results[0].Error)
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
