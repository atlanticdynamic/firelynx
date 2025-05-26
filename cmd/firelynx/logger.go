package main

import (
	"github.com/atlanticdynamic/firelynx/internal/logging"
)

// SetupLogger configures the default logger based on provided log level
func SetupLogger(logLevel string) {
	logging.SetupLogger(logLevel)
}
