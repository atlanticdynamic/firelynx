package main

import (
	"log/slog"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

func TestSetupLogger(t *testing.T) {
	// Save original default logger to restore after tests
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	tests := []struct {
		name          string
		logLevel      string
		expectedLevel log.Level
	}{
		{
			name:          "sets up logger with debug level",
			logLevel:      "debug",
			expectedLevel: log.DebugLevel,
		},
		{
			name:          "sets up logger with trace level",
			logLevel:      "trace",
			expectedLevel: log.DebugLevel, // Trace maps to DebugLevel in the implementation
		},
		{
			name:          "sets up logger with info level",
			logLevel:      "info",
			expectedLevel: log.InfoLevel,
		},
		{
			name:          "sets up logger with warn level",
			logLevel:      "warn",
			expectedLevel: log.WarnLevel,
		},
		{
			name:          "sets up logger with error level",
			logLevel:      "error",
			expectedLevel: log.ErrorLevel,
		},
		{
			name:          "sets up logger with default level when empty",
			logLevel:      "",
			expectedLevel: log.InfoLevel, // Default is InfoLevel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function being tested
			SetupLogger(tt.logLevel)

			// Get the default logger
			logger := slog.Default()

			// Rather than trying to directly extract the level from the slog handler,
			// we use a behavior-based approach to check which level messages will be logged at
			actualLevel := log.InfoLevel // Default

			// Get context from test for proper test timeout handling
			ctx := t.Context()

			// Debug messages work at Debug level but not at Info or higher
			if logger.Enabled(ctx, slog.LevelDebug) {
				actualLevel = log.DebugLevel
			} else if logger.Enabled(ctx, slog.LevelInfo) {
				actualLevel = log.InfoLevel
			} else if logger.Enabled(ctx, slog.LevelWarn) {
				actualLevel = log.WarnLevel
			} else if logger.Enabled(ctx, slog.LevelError) {
				actualLevel = log.ErrorLevel
			}

			// Verify the level was set correctly
			assert.Equal(t, tt.expectedLevel, actualLevel,
				"Expected log level %s for input '%s', but got %s",
				tt.expectedLevel, tt.logLevel, actualLevel)
		})
	}
}
