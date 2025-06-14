package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// SetupHandlerText configures a text slog handler with the provided writer and log level
func SetupHandlerText(logLevel string, writer io.Writer) slog.Handler {
	if writer == nil {
		writer = os.Stderr
	}

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

	return log.NewWithOptions(writer, log.Options{
		ReportTimestamp: reportTimestamp,
		ReportCaller:    reportCaller,
		Level:           lvl,
	})
}

// SetupHandlerJSON configures a JSON slog handler with the provided writer and log level
func SetupHandlerJSON(logLevel string, writer io.Writer) slog.Handler {
	if writer == nil {
		writer = os.Stdout
	}

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

	return slog.NewJSONHandler(writer, opts)
}

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
	handler := SetupHandlerText(logLevel, nil)
	slog.SetDefault(slog.New(handler))
}
