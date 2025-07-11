package logger

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// ToProto converts ConsoleLogger to protobuf format
func (c *ConsoleLogger) ToProto() any {
	format := formatToProto(c.Options.Format)
	level := levelToProto(c.Options.Level)
	preset := presetToProto(c.Preset)

	config := &pb.ConsoleLoggerConfig{
		Options: &pb.LogOptionsGeneral{
			Format: &format,
			Level:  &level,
		},
		Fields: &pb.LogOptionsHTTP{
			Method:      &c.Fields.Method,
			Path:        &c.Fields.Path,
			ClientIp:    &c.Fields.ClientIP,
			QueryParams: &c.Fields.QueryParams,
			Protocol:    &c.Fields.Protocol,
			Host:        &c.Fields.Host,
			Scheme:      &c.Fields.Scheme,
			StatusCode:  &c.Fields.StatusCode,
			Duration:    &c.Fields.Duration,
			Request:     directionConfigToProto(c.Fields.Request),
			Response:    directionConfigToProto(c.Fields.Response),
		},
		Output: &c.Output,
		Preset: &preset,
	}

	// Add path filtering
	if len(c.IncludeOnlyPaths) > 0 {
		config.IncludeOnlyPaths = c.IncludeOnlyPaths
	}
	if len(c.ExcludePaths) > 0 {
		config.ExcludePaths = c.ExcludePaths
	}

	// Add method filtering
	if len(c.IncludeOnlyMethods) > 0 {
		config.IncludeOnlyMethods = c.IncludeOnlyMethods
	}
	if len(c.ExcludeMethods) > 0 {
		config.ExcludeMethods = c.ExcludeMethods
	}

	return config
}

// FromProto converts protobuf ConsoleLoggerConfig to domain ConsoleLogger
func FromProto(pbConfig *pb.ConsoleLoggerConfig) (*ConsoleLogger, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil console logger config")
	}

	config := &ConsoleLogger{}

	// Convert general options
	if pbConfig.Options != nil {
		config.Options = LogOptionsGeneral{
			Format: formatFromProto(pbConfig.Options.GetFormat()),
			Level:  levelFromProto(pbConfig.Options.GetLevel()),
		}
	}

	// Convert output with environment variable interpolation
	// Empty string defaults to stdout in CreateWriter
	expandedOutput, err := interpolation.ExpandEnvVarsWithDefaults(pbConfig.GetOutput())
	if err != nil {
		return nil, fmt.Errorf("environment variable expansion failed: %w", err)
	}
	config.Output = expandedOutput

	// Convert preset
	config.Preset = presetFromProto(pbConfig.GetPreset())

	// Convert HTTP fields
	if pbConfig.Fields != nil {
		config.Fields = LogOptionsHTTP{
			Method:      pbConfig.Fields.GetMethod(),
			Path:        pbConfig.Fields.GetPath(),
			ClientIP:    pbConfig.Fields.GetClientIp(),
			QueryParams: pbConfig.Fields.GetQueryParams(),
			Protocol:    pbConfig.Fields.GetProtocol(),
			Host:        pbConfig.Fields.GetHost(),
			Scheme:      pbConfig.Fields.GetScheme(),
			StatusCode:  pbConfig.Fields.GetStatusCode(),
			Duration:    pbConfig.Fields.GetDuration(),
		}

		if pbConfig.Fields.Request != nil {
			config.Fields.Request = directionConfigFromProto(pbConfig.Fields.Request)
		}
		if pbConfig.Fields.Response != nil {
			config.Fields.Response = directionConfigFromProto(pbConfig.Fields.Response)
		}
	}

	// Convert filtering options
	if len(pbConfig.IncludeOnlyPaths) > 0 {
		config.IncludeOnlyPaths = pbConfig.IncludeOnlyPaths
	}
	if len(pbConfig.ExcludePaths) > 0 {
		config.ExcludePaths = pbConfig.ExcludePaths
	}
	if len(pbConfig.IncludeOnlyMethods) > 0 {
		config.IncludeOnlyMethods = pbConfig.IncludeOnlyMethods
	}
	if len(pbConfig.ExcludeMethods) > 0 {
		config.ExcludeMethods = pbConfig.ExcludeMethods
	}

	return config, nil
}

// Helper functions for format conversion
func formatToProto(format Format) pb.LogOptionsGeneral_Format {
	switch format {
	case FormatTxt:
		return pb.LogOptionsGeneral_FORMAT_TXT
	case FormatJSON:
		return pb.LogOptionsGeneral_FORMAT_JSON
	default:
		return pb.LogOptionsGeneral_FORMAT_UNSPECIFIED
	}
}

func formatFromProto(pbFormat pb.LogOptionsGeneral_Format) Format {
	switch pbFormat {
	case pb.LogOptionsGeneral_FORMAT_TXT:
		return FormatTxt
	case pb.LogOptionsGeneral_FORMAT_JSON:
		return FormatJSON
	default:
		return FormatUnspecified
	}
}

// Helper functions for level conversion
func levelToProto(level Level) pb.LogOptionsGeneral_Level {
	switch level {
	case LevelDebug:
		return pb.LogOptionsGeneral_LEVEL_DEBUG
	case LevelInfo:
		return pb.LogOptionsGeneral_LEVEL_INFO
	case LevelWarn:
		return pb.LogOptionsGeneral_LEVEL_WARN
	case LevelError:
		return pb.LogOptionsGeneral_LEVEL_ERROR
	case LevelFatal:
		return pb.LogOptionsGeneral_LEVEL_FATAL
	default:
		return pb.LogOptionsGeneral_LEVEL_UNSPECIFIED
	}
}

func levelFromProto(pbLevel pb.LogOptionsGeneral_Level) Level {
	switch pbLevel {
	case pb.LogOptionsGeneral_LEVEL_DEBUG:
		return LevelDebug
	case pb.LogOptionsGeneral_LEVEL_INFO:
		return LevelInfo
	case pb.LogOptionsGeneral_LEVEL_WARN:
		return LevelWarn
	case pb.LogOptionsGeneral_LEVEL_ERROR:
		return LevelError
	case pb.LogOptionsGeneral_LEVEL_FATAL:
		return LevelFatal
	default:
		return LevelUnspecified
	}
}

// Helper functions for direction config conversion
func directionConfigToProto(config DirectionConfig) *pb.LogOptionsHTTP_DirectionConfig {
	return &pb.LogOptionsHTTP_DirectionConfig{
		Enabled:        &config.Enabled,
		Body:           &config.Body,
		MaxBodySize:    &config.MaxBodySize,
		BodySize:       &config.BodySize,
		Headers:        &config.Headers,
		IncludeHeaders: config.IncludeHeaders,
		ExcludeHeaders: config.ExcludeHeaders,
	}
}

func directionConfigFromProto(pbConfig *pb.LogOptionsHTTP_DirectionConfig) DirectionConfig {
	config := DirectionConfig{
		Enabled:     pbConfig.GetEnabled(),
		Body:        pbConfig.GetBody(),
		MaxBodySize: pbConfig.GetMaxBodySize(),
		BodySize:    pbConfig.GetBodySize(),
		Headers:     pbConfig.GetHeaders(),
	}

	if len(pbConfig.IncludeHeaders) > 0 {
		config.IncludeHeaders = pbConfig.IncludeHeaders
	}
	if len(pbConfig.ExcludeHeaders) > 0 {
		config.ExcludeHeaders = pbConfig.ExcludeHeaders
	}

	return config
}

// Helper functions for preset conversion
func presetToProto(preset Preset) pb.ConsoleLoggerConfig_LogPreset {
	switch preset {
	case PresetMinimal:
		return pb.ConsoleLoggerConfig_PRESET_MINIMAL
	case PresetStandard:
		return pb.ConsoleLoggerConfig_PRESET_STANDARD
	case PresetDetailed:
		return pb.ConsoleLoggerConfig_PRESET_DETAILED
	case PresetDebug:
		return pb.ConsoleLoggerConfig_PRESET_DEBUG
	default:
		return pb.ConsoleLoggerConfig_PRESET_UNSPECIFIED
	}
}

func presetFromProto(pbPreset pb.ConsoleLoggerConfig_LogPreset) Preset {
	switch pbPreset {
	case pb.ConsoleLoggerConfig_PRESET_MINIMAL:
		return PresetMinimal
	case pb.ConsoleLoggerConfig_PRESET_STANDARD:
		return PresetStandard
	case pb.ConsoleLoggerConfig_PRESET_DETAILED:
		return PresetDetailed
	case pb.ConsoleLoggerConfig_PRESET_DEBUG:
		return PresetDebug
	default:
		return PresetUnspecified
	}
}
