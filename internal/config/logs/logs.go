package logs

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
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

// FromProto converts a Protocol Buffer LogOptions to a domain Config
func FromProto(pbLog *pb.LogOptions) Config {
	if pbLog == nil {
		return Config{}
	}

	return Config{
		Format: protoFormatToFormat(pbLog.GetFormat()),
		Level:  protoLevelToLevel(pbLog.GetLevel()),
	}
}

// ToProto converts a domain Config to a Protocol Buffer LogOptions
func (lc Config) ToProto() *pb.LogOptions {
	pbLog := &pb.LogOptions{}

	pbFormat := formatToProto(lc.Format)
	pbLevel := levelToProto(lc.Level)

	pbLog.Format = &pbFormat
	pbLog.Level = &pbLevel

	return pbLog
}

// Convert Protocol Buffer enums to domain enums
func protoFormatToFormat(format pb.LogFormat) Format {
	switch format {
	case pb.LogFormat_LOG_FORMAT_JSON:
		return FormatJSON
	case pb.LogFormat_LOG_FORMAT_TXT:
		return FormatText
	default:
		return FormatUnspecified
	}
}

func protoLevelToLevel(level pb.LogLevel) Level {
	switch level {
	case pb.LogLevel_LOG_LEVEL_DEBUG:
		return LevelDebug
	case pb.LogLevel_LOG_LEVEL_INFO:
		return LevelInfo
	case pb.LogLevel_LOG_LEVEL_WARN:
		return LevelWarn
	case pb.LogLevel_LOG_LEVEL_ERROR:
		return LevelError
	case pb.LogLevel_LOG_LEVEL_FATAL:
		return LevelFatal
	default:
		return LevelUnspecified
	}
}

// Convert domain enums to Protocol Buffer enums
func formatToProto(format Format) pb.LogFormat {
	switch format {
	case FormatJSON:
		return pb.LogFormat_LOG_FORMAT_JSON
	case FormatText:
		return pb.LogFormat_LOG_FORMAT_TXT
	default:
		return pb.LogFormat_LOG_FORMAT_UNSPECIFIED
	}
}

func levelToProto(level Level) pb.LogLevel {
	switch level {
	case LevelDebug:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case LevelInfo:
		return pb.LogLevel_LOG_LEVEL_INFO
	case LevelWarn:
		return pb.LogLevel_LOG_LEVEL_WARN
	case LevelError:
		return pb.LogLevel_LOG_LEVEL_ERROR
	case LevelFatal:
		return pb.LogLevel_LOG_LEVEL_FATAL
	default:
		return pb.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}
