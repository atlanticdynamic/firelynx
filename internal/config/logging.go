package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// LoggingConfig contains logging-related configuration options
type LoggingConfig struct {
	Format LogFormat
	Level  LogLevel
}

// LogFormat represents the logging output format
type LogFormat string

// LogLevel represents the logging verbosity level
type LogLevel string

// Constants for LogFormat
const (
	LogFormatUnspecified LogFormat = ""
	LogFormatText        LogFormat = "text"
	LogFormatJSON        LogFormat = "json"
)

// Constants for LogLevel
const (
	LogLevelUnspecified LogLevel = ""
	LogLevelDebug       LogLevel = "debug"
	LogLevelInfo        LogLevel = "info"
	LogLevelWarn        LogLevel = "warn"
	LogLevelError       LogLevel = "error"
	LogLevelFatal       LogLevel = "fatal"
)

// String returns the string representation of LogFormat
func (f LogFormat) String() string {
	return string(f)
}

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	return string(l)
}

// IsValid checks if the LogFormat is valid
func (f LogFormat) IsValid() bool {
	switch f {
	case LogFormatUnspecified, LogFormatText, LogFormatJSON:
		return true
	default:
		return false
	}
}

// IsValid checks if the LogLevel is valid
func (l LogLevel) IsValid() bool {
	switch l {
	case LogLevelUnspecified, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal:
		return true
	default:
		return false
	}
}

// LogFormatFromString converts a string to a LogFormat
func LogFormatFromString(format string) (LogFormat, error) {
	switch format {
	case "json":
		return LogFormatJSON, nil
	case "text", "txt":
		return LogFormatText, nil
	case "":
		return LogFormatUnspecified, nil
	default:
		return LogFormatUnspecified, fmt.Errorf("unknown log format: %s", format)
	}
}

// LogLevelFromString converts a string to a LogLevel
func LogLevelFromString(level string) (LogLevel, error) {
	switch level {
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "warn", "warning":
		return LogLevelWarn, nil
	case "error":
		return LogLevelError, nil
	case "fatal":
		return LogLevelFatal, nil
	case "":
		return LogLevelUnspecified, nil
	default:
		return LogLevelUnspecified, fmt.Errorf("unknown log level: %s", level)
	}
}

// FromProto converts a Protocol Buffer LogOptions to a domain LoggingConfig
func LoggingConfigFromProto(pbLog *pb.LogOptions) LoggingConfig {
	if pbLog == nil {
		return LoggingConfig{}
	}

	return LoggingConfig{
		Format: protoFormatToLogFormat(pbLog.GetFormat()),
		Level:  protoLevelToLogLevel(pbLog.GetLevel()),
	}
}

// ToProto converts a domain LoggingConfig to a Protocol Buffer LogOptions
func (lc LoggingConfig) ToProto() *pb.LogOptions {
	pbLog := &pb.LogOptions{}
	
	pbFormat := logFormatToProto(lc.Format)
	pbLevel := logLevelToProto(lc.Level)
	
	pbLog.Format = &pbFormat
	pbLog.Level = &pbLevel
	
	return pbLog
}

// Convert Protocol Buffer enums to domain enums
func protoFormatToLogFormat(format pb.LogFormat) LogFormat {
	switch format {
	case pb.LogFormat_LOG_FORMAT_JSON:
		return LogFormatJSON
	case pb.LogFormat_LOG_FORMAT_TXT:
		return LogFormatText
	default:
		return LogFormatUnspecified
	}
}

func protoLevelToLogLevel(level pb.LogLevel) LogLevel {
	switch level {
	case pb.LogLevel_LOG_LEVEL_DEBUG:
		return LogLevelDebug
	case pb.LogLevel_LOG_LEVEL_INFO:
		return LogLevelInfo
	case pb.LogLevel_LOG_LEVEL_WARN:
		return LogLevelWarn
	case pb.LogLevel_LOG_LEVEL_ERROR:
		return LogLevelError
	case pb.LogLevel_LOG_LEVEL_FATAL:
		return LogLevelFatal
	default:
		return LogLevelUnspecified
	}
}

// Convert domain enums to Protocol Buffer enums
func logFormatToProto(format LogFormat) pb.LogFormat {
	switch format {
	case LogFormatJSON:
		return pb.LogFormat_LOG_FORMAT_JSON
	case LogFormatText:
		return pb.LogFormat_LOG_FORMAT_TXT
	default:
		return pb.LogFormat_LOG_FORMAT_UNSPECIFIED
	}
}

func logLevelToProto(level LogLevel) pb.LogLevel {
	switch level {
	case LogLevelDebug:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case LogLevelInfo:
		return pb.LogLevel_LOG_LEVEL_INFO
	case LogLevelWarn:
		return pb.LogLevel_LOG_LEVEL_WARN
	case LogLevelError:
		return pb.LogLevel_LOG_LEVEL_ERROR
	case LogLevelFatal:
		return pb.LogLevel_LOG_LEVEL_FATAL
	default:
		return pb.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}