package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// SetupHandler configures the default logger to use the provided handler
func SetupHandler(logLevel string) slog.Handler {
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

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
	handler := SetupHandler(logLevel)
	slog.SetDefault(slog.New(handler))
}
