package logs

import "errors"

// Standard errors for the logs package
var (
	// ErrInvalidLogFormat is returned when an unsupported log format is provided
	ErrInvalidLogFormat = errors.New("invalid log format")

	// ErrInvalidLogLevel is returned when an unsupported log level is provided
	ErrInvalidLogLevel = errors.New("invalid log level")
)
