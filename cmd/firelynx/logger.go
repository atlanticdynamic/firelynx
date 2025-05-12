package main

import (
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
	level := slog.LevelInfo // Default level

	// Parse log level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Configure and set the default logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}
