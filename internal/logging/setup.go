package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
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

	handler := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: reportTimestamp,
		ReportCaller:    reportCaller,
		Level:           lvl,
	})

	slog.SetDefault(slog.New(handler))
}
