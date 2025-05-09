package logs

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

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
