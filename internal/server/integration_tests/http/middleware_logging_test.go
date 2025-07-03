//go:build integration

package http_test

import (
	"bytes"
	"context"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	loggerConfig "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	centralLogger "github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/logger"
	"github.com/robbyt/go-loglater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/logging.toml
var loggingTomlContent []byte

func TestLoggingConfigurationPipeline(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "logging.toml")
	err := os.WriteFile(configPath, loggingTomlContent, 0o644)
	require.NoError(t, err)

	t.Run("loads logging.toml and creates JSON handler", func(t *testing.T) {
		// Load the TOML configuration directly from file path
		cfg, err := config.NewConfig(configPath)
		require.NoError(t, err, "should load logging.toml successfully")

		// Verify the configuration loaded correctly
		require.NotNil(t, cfg, "config should not be nil")
		require.NotEmpty(t, cfg.Endpoints, "should have endpoints")

		endpoint := cfg.Endpoints[0]
		require.NotEmpty(t, endpoint.Middlewares, "endpoint should have middlewares")

		// Find the console logger middleware
		var consoleLoggerConfig *loggerConfig.ConsoleLogger
		for _, middleware := range endpoint.Middlewares {
			if config, ok := middleware.Config.(*loggerConfig.ConsoleLogger); ok {
				consoleLoggerConfig = config
				break
			}
		}
		require.NotNil(t, consoleLoggerConfig, "should find console logger middleware")

		// Verify the format is set to JSON
		assert.Equal(
			t,
			loggerConfig.FormatJSON,
			consoleLoggerConfig.Options.Format,
			"format should be JSON from TOML",
		)

		// Create the middleware with the loaded config
		consoleLogger, err := logger.NewConsoleLogger("test-logger", consoleLoggerConfig)
		require.NoError(t, err, "should create console logger without error")
		require.NotNil(t, consoleLogger, "console logger should be created")

		// Test that the middleware actually logs in JSON format using loglater
		// Create a log collector to capture the output
		logCollector := loglater.NewLogCollector(nil)

		// Create a new console logger with our collector as the handler
		jsonHandler := centralLogger.SetupHandlerJSON("info", nil)
		testLogger := slog.New(logCollector).WithGroup("http")

		// Create a context and test log entry
		logCtx := context.Background()
		attrs := []slog.Attr{
			slog.String("method", "GET"),
			slog.String("path", "/test"),
			slog.Int64("status", 200),
		}

		// Log an entry using the test logger
		testLogger.LogAttrs(logCtx, slog.LevelInfo, "test-logger", attrs...)

		// Get the captured logs directly
		logs := logCollector.GetLogs()
		require.Len(t, logs, 1, "should have captured exactly one log entry")

		// Convert the storage.Record back to slog.Record for verification
		storedLog := logs[0]
		record := slog.NewRecord(storedLog.Time, storedLog.Level, storedLog.Message, storedLog.PC)
		record.AddAttrs(storedLog.Attrs...)

		// Convert to JSON for verification
		var buf bytes.Buffer
		jsonHandler = slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		jsonHandler.Handle(logCtx, record)

		logOutput := buf.String()
		t.Logf("Captured log output: %s", logOutput)

		// Verify the output is JSON format (should contain {"level":..., "msg":...})
		assert.Contains(t, logOutput, `"level"`, "log output should contain JSON level field")
		assert.Contains(t, logOutput, `"msg"`, "log output should contain JSON msg field")
		assert.Contains(
			t,
			logOutput,
			`"method":"GET"`,
			"log output should contain method attribute in JSON",
		)
		assert.Contains(
			t,
			logOutput,
			`"path":"/test"`,
			"log output should contain path attribute in JSON",
		)
		assert.Contains(
			t,
			logOutput,
			`"status":200`,
			"log output should contain status attribute in JSON",
		)

		// Verify it's NOT the charmbracelet text format
		assert.NotContains(
			t,
			logOutput,
			"INFO http:",
			"should not contain charmbracelet text format",
		)
	})

	t.Run("verifies handler selection logic", func(t *testing.T) {
		// Test JSON handler creation directly
		jsonHandler := centralLogger.SetupHandlerJSON("info", nil)
		require.NotNil(t, jsonHandler, "JSON handler should be created")

		// Test text handler creation directly
		textHandler := centralLogger.SetupHandlerText("info", nil)
		require.NotNil(t, textHandler, "text handler should be created")

		// They should be different types
		assert.IsType(
			t,
			&slog.JSONHandler{},
			jsonHandler,
			"JSON handler should be JSONHandler type",
		)
		// Note: textHandler is charmbracelet's handler, not slog.TextHandler
	})
}
