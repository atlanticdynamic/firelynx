package logs

import (
	"fmt"
)

// Constants for Format
const (
	FormatUnspecified Format = ""
	FormatText        Format = "text"
	FormatJSON        Format = "json"
)

// Constants for Level
const (
	LevelUnspecified Level = ""
	LevelDebug       Level = "debug"
	LevelInfo        Level = "info"
	LevelWarn        Level = "warn"
	LevelError       Level = "error"
	LevelFatal       Level = "fatal"
)

// Config contains logging-related configuration options
type Config struct {
	Format Format
	Level  Level
}

// Format represents the logging output format
type Format string

// Level represents the logging verbosity level
type Level string

// String returns the string representation of Format
func (f Format) String() string {
	return string(f)
}

// String returns the string representation of Level
func (l Level) String() string {
	return string(l)
}

// IsValid checks if the Format is valid
func (f Format) IsValid() bool {
	switch f {
	case FormatUnspecified, FormatText, FormatJSON:
		return true
	default:
		return false
	}
}

// IsValid checks if the Level is valid
func (l Level) IsValid() bool {
	switch l {
	case LevelUnspecified,
		LevelDebug,
		LevelInfo,
		LevelWarn,
		LevelError,
		LevelFatal:
		return true
	default:
		return false
	}
}

// FormatFromString converts a string to a Format
func FormatFromString(format string) (Format, error) {
	switch format {
	case "json":
		return FormatJSON, nil
	case "text", "txt":
		return FormatText, nil
	case "":
		return FormatUnspecified, nil
	default:
		return FormatUnspecified, fmt.Errorf("unknown log format: %s", format)
	}
}

// LevelFromString converts a string to a Level
func LevelFromString(level string) (Level, error) {
	switch level {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	case "":
		return LevelUnspecified, nil
	default:
		return LevelUnspecified, fmt.Errorf("unknown log level: %s", level)
	}
}
