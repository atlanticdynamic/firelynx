package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// SetupHandlerText configures the default logger to use the provided handler
func SetupHandlerText(logLevel string) slog.Handler {
	reportCaller := false
	reportTimestamp := false
	lvl := log.InfoLevel
	switch strings.ToLower(logLevel) {
	case "trace":
		reportCaller = true
		reportTimestamp = true
		lvl = log.DebugLevel
	case "debug":
		reportTimestamp = true
		lvl = log.DebugLevel
	case "info":
		lvl = log.InfoLevel
	case "warn", "warning":
		lvl = log.WarnLevel
	case "error":
		lvl = log.ErrorLevel
	}

	return log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: reportTimestamp,
		ReportCaller:    reportCaller,
		Level:           lvl,
	})
}

// SetupHandlerJSON configures a JSON slog handler with the provided log level
func SetupHandlerJSON(logLevel string) slog.Handler {
	reportCaller := false
	var level slog.Level

	switch strings.ToLower(logLevel) {
	case "trace":
		reportCaller = true
		level = slog.LevelDebug
	case "debug":
		reportCaller = false
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: reportCaller,
	}

	return slog.NewJSONHandler(os.Stdout, opts)
}

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
	handler := SetupHandlerText(logLevel)
	slog.SetDefault(slog.New(handler))
}
