package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupHandlerText(t *testing.T) {
	tests := []struct {
		name            string
		logLevel        string
		writer          func() *bytes.Buffer
		expectedLevel   log.Level
		expectCaller    bool
		expectTimestamp bool
	}{
		{
			name:            "trace level",
			logLevel:        "trace",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.DebugLevel,
			expectCaller:    true,
			expectTimestamp: true,
		},
		{
			name:            "debug level",
			logLevel:        "debug",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.DebugLevel,
			expectCaller:    false,
			expectTimestamp: true,
		},
		{
			name:            "info level",
			logLevel:        "info",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.InfoLevel,
			expectCaller:    false,
			expectTimestamp: false,
		},
		{
			name:            "warn level",
			logLevel:        "warn",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.WarnLevel,
			expectCaller:    false,
			expectTimestamp: false,
		},
		{
			name:            "warning level",
			logLevel:        "warning",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.WarnLevel,
			expectCaller:    false,
			expectTimestamp: false,
		},
		{
			name:            "error level",
			logLevel:        "error",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.ErrorLevel,
			expectCaller:    false,
			expectTimestamp: false,
		},
		{
			name:            "uppercase level",
			logLevel:        "INFO",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.InfoLevel,
			expectCaller:    false,
			expectTimestamp: false,
		},
		{
			name:            "mixed case level",
			logLevel:        "DeBuG",
			writer:          func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel:   log.DebugLevel,
			expectCaller:    false,
			expectTimestamp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.writer()
			handler := SetupHandlerText(tt.logLevel, buf)

			// Verify handler is created
			require.NotNil(t, handler)

			// Test logging with the handler - use appropriate level for the configured level
			logger := slog.New(handler)

			// Choose log level that will actually output based on the configured level
			switch strings.ToLower(tt.logLevel) {
			case "trace", "debug", "info":
				logger.Info("test message", "key", "value")
			case "warn", "warning":
				logger.Warn("test message", "key", "value")
			case "error":
				logger.Error("test message", "key", "value")
			default:
				logger.Info("test message", "key", "value")
			}

			// Verify output was written
			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "test message")
			assert.Contains(t, output, "key")
			assert.Contains(t, output, "value")

			// Verify timestamp presence based on level
			if tt.expectTimestamp {
				// For text handler, timestamp appears in various formats
				// We just verify some time-related content is present
				hasTimeIndicator := strings.Contains(output, "202") || // year
					strings.Contains(output, ":") || // time separator
					strings.Contains(output, "T") // ISO format
				assert.True(
					t,
					hasTimeIndicator,
					"Expected timestamp in output for level %s",
					tt.logLevel,
				)
			}
		})
	}
}

func TestSetupHandlerText_NilWriter(t *testing.T) {
	// Test that nil writer defaults to os.Stderr
	handler := SetupHandlerText("info", nil)
	require.NotNil(t, handler)

	// We can't easily test that it uses os.Stderr without redirecting stderr,
	// but we can verify the handler works
	logger := slog.New(handler)
	// This should not panic and should write to stderr
	logger.Info("test message for stderr")
}

func TestSetupHandlerJSON(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		writer        func() *bytes.Buffer
		expectedLevel slog.Level
		expectCaller  bool
	}{
		{
			name:          "trace level",
			logLevel:      "trace",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelDebug,
			expectCaller:  true,
		},
		{
			name:          "debug level",
			logLevel:      "debug",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelDebug,
			expectCaller:  false,
		},
		{
			name:          "info level",
			logLevel:      "info",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelInfo,
			expectCaller:  false,
		},
		{
			name:          "warn level",
			logLevel:      "warn",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelWarn,
			expectCaller:  false,
		},
		{
			name:          "warning level",
			logLevel:      "warning",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelWarn,
			expectCaller:  false,
		},
		{
			name:          "error level",
			logLevel:      "error",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelError,
			expectCaller:  false,
		},
		{
			name:          "default level (unknown)",
			logLevel:      "unknown",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelInfo,
			expectCaller:  false,
		},
		{
			name:          "uppercase level",
			logLevel:      "ERROR",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelError,
			expectCaller:  false,
		},
		{
			name:          "empty level defaults to info",
			logLevel:      "",
			writer:        func() *bytes.Buffer { return &bytes.Buffer{} },
			expectedLevel: slog.LevelInfo,
			expectCaller:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.writer()
			handler := SetupHandlerJSON(tt.logLevel, buf)

			// Verify handler is created
			require.NotNil(t, handler)

			// Test logging with the handler - use appropriate level for the configured level
			logger := slog.New(handler)

			var expectedLevelStr string
			switch strings.ToLower(tt.logLevel) {
			case "trace", "debug", "info", "", "unknown":
				logger.Info("test message", "key", "value")
				expectedLevelStr = `"level":"INFO"`
			case "warn", "warning":
				logger.Warn("test message", "key", "value")
				expectedLevelStr = `"level":"WARN"`
			case "error":
				logger.Error("test message", "key", "value")
				expectedLevelStr = `"level":"ERROR"`
			default:
				logger.Info("test message", "key", "value")
				expectedLevelStr = `"level":"INFO"`
			}

			// Verify JSON output was written
			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, `"msg":"test message"`)
			assert.Contains(t, output, `"key":"value"`)
			assert.Contains(t, output, expectedLevelStr)

			// Verify source is included for trace level
			if tt.expectCaller {
				assert.Contains(t, output, `"source"`)
			}
		})
	}
}

func TestSetupHandlerJSON_NilWriter(t *testing.T) {
	// Test that nil writer defaults to os.Stdout
	handler := SetupHandlerJSON("info", nil)
	require.NotNil(t, handler)

	// We can't easily test that it uses os.Stdout without redirecting stdout,
	// but we can verify the handler works
	logger := slog.New(handler)
	// This should not panic and should write to stdout
	logger.Info("test message for stdout")
}

func TestSetupHandlerJSON_LevelFiltering(t *testing.T) {
	// Test that log level filtering works correctly
	buf := &bytes.Buffer{}
	handler := SetupHandlerJSON("warn", buf)
	logger := slog.New(handler)

	// These should not appear in output
	logger.Debug("debug message")
	logger.Info("info message")

	// These should appear in output
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	// Should not contain debug/info messages
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")

	// Should contain warn/error messages
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestSetupHandlerText_LevelFiltering(t *testing.T) {
	// Test that log level filtering works correctly for text handler
	buf := &bytes.Buffer{}
	handler := SetupHandlerText("error", buf)
	logger := slog.New(handler)

	// These should not appear in output
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	// This should appear in output
	logger.Error("error message")

	output := buf.String()

	// Should not contain debug/info/warn messages
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "warn message")

	// Should contain error message
	assert.Contains(t, output, "error message")
}

func TestSetupLogger(t *testing.T) {
	// Store original default logger to restore after test
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	tests := []struct {
		name     string
		logLevel string
	}{
		{
			name:     "debug level",
			logLevel: "debug",
		},
		{
			name:     "info level",
			logLevel: "info",
		},
		{
			name:     "warn level",
			logLevel: "warn",
		},
		{
			name:     "error level",
			logLevel: "error",
		},
		{
			name:     "trace level",
			logLevel: "trace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger
			SetupLogger(tt.logLevel)

			// Verify default logger was changed
			defaultLogger := slog.Default()
			assert.NotNil(t, defaultLogger)

			// Test that the logger works (this will output to stderr)
			// We just verify it doesn't panic
			defaultLogger.Info("test message from default logger")
		})
	}
}

func TestSetupLogger_Integration(t *testing.T) {
	// Store original default logger to restore after test
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	// Test the complete flow: setup logger and use it
	SetupLogger("debug")

	// Use the default logger that was configured
	ctx := context.Background()
	slog.InfoContext(ctx, "integration test message",
		"component", "logging",
		"test", true)

	// Verify logger was set up (no panic is a good sign)
	// The actual output goes to stderr which we can't easily capture in this test
}

func TestHandlerTypes(t *testing.T) {
	// Test that the correct handler types are returned
	// Factory Function Testing Pattern: Use assert.IsType() to verify concrete types
	buf := &bytes.Buffer{}

	textHandler := SetupHandlerText("info", buf)
	jsonHandler := SetupHandlerJSON("info", buf)

	// Verify they are different types
	assert.IsType(t, &log.Logger{}, textHandler)
	assert.IsType(t, &slog.JSONHandler{}, jsonHandler)
}
