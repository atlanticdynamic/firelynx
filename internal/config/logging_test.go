package config

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestLogFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   LogFormat
		expected string
	}{
		{"Empty", LogFormatUnspecified, ""},
		{"Text", LogFormatText, "text"},
		{"JSON", LogFormatJSON, "json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.format.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{"Empty", LogLevelUnspecified, ""},
		{"Debug", LogLevelDebug, "debug"},
		{"Info", LogLevelInfo, "info"},
		{"Warn", LogLevelWarn, "warn"},
		{"Error", LogLevelError, "error"},
		{"Fatal", LogLevelFatal, "fatal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.level.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogFormat_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		format   LogFormat
		expected bool
	}{
		{"Empty", LogFormatUnspecified, true},
		{"Text", LogFormatText, true},
		{"JSON", LogFormatJSON, true},
		{"Invalid", LogFormat("invalid"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.format.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected bool
	}{
		{"Empty", LogLevelUnspecified, true},
		{"Debug", LogLevelDebug, true},
		{"Info", LogLevelInfo, true},
		{"Warn", LogLevelWarn, true},
		{"Error", LogLevelError, true},
		{"Fatal", LogLevelFatal, true},
		{"Invalid", LogLevel("invalid"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.level.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogFormatFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    LogFormat
		expectError bool
	}{
		{"Empty", "", LogFormatUnspecified, false},
		{"JSON", "json", LogFormatJSON, false},
		{"Text", "text", LogFormatText, false},
		{"Txt", "txt", LogFormatText, false},
		{"Invalid", "invalid", LogFormatUnspecified, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := LogFormatFromString(tc.input)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    LogLevel
		expectError bool
	}{
		{"Empty", "", LogLevelUnspecified, false},
		{"Debug", "debug", LogLevelDebug, false},
		{"Info", "info", LogLevelInfo, false},
		{"Warn", "warn", LogLevelWarn, false},
		{"Warning", "warning", LogLevelWarn, false},
		{"Error", "error", LogLevelError, false},
		{"Fatal", "fatal", LogLevelFatal, false},
		{"Invalid", "invalid", LogLevelUnspecified, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := LogLevelFromString(tc.input)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLoggingConfigFromProto(t *testing.T) {
	tests := []struct {
		name     string
		input    *pb.LogOptions
		expected LoggingConfig
	}{
		{
			name:     "Nil",
			input:    nil,
			expected: LoggingConfig{},
		},
		{
			name: "Empty",
			input: &pb.LogOptions{},
			expected: LoggingConfig{
				Format: LogFormatUnspecified,
				Level:  LogLevelUnspecified,
			},
		},
		{
			name: "With Values",
			input: func() *pb.LogOptions {
				format := pb.LogFormat_LOG_FORMAT_JSON
				level := pb.LogLevel_LOG_LEVEL_INFO
				return &pb.LogOptions{
					Format: &format,
					Level:  &level,
				}
			}(),
			expected: LoggingConfig{
				Format: LogFormatJSON,
				Level:  LogLevelInfo,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LoggingConfigFromProto(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLoggingConfig_ToProto(t *testing.T) {
	tests := []struct {
		name     string
		input    LoggingConfig
		expected *pb.LogOptions
	}{
		{
			name:  "Empty",
			input: LoggingConfig{},
			expected: func() *pb.LogOptions {
				format := pb.LogFormat_LOG_FORMAT_UNSPECIFIED
				level := pb.LogLevel_LOG_LEVEL_UNSPECIFIED
				return &pb.LogOptions{
					Format: &format,
					Level:  &level,
				}
			}(),
		},
		{
			name: "With Values",
			input: LoggingConfig{
				Format: LogFormatJSON,
				Level:  LogLevelInfo,
			},
			expected: func() *pb.LogOptions {
				format := pb.LogFormat_LOG_FORMAT_JSON
				level := pb.LogLevel_LOG_LEVEL_INFO
				return &pb.LogOptions{
					Format: &format,
					Level:  &level,
				}
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.ToProto()
			// Compare enums directly since the proto objects contain pointers
			assert.Equal(t, tc.expected.GetFormat(), result.GetFormat())
			assert.Equal(t, tc.expected.GetLevel(), result.GetLevel())
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	// Start with domain model
	original := LoggingConfig{
		Format: LogFormatJSON,
		Level:  LogLevelWarn,
	}
	
	// Convert to proto
	pbLogOptions := original.ToProto()
	
	// Convert back to domain model
	roundTrip := LoggingConfigFromProto(pbLogOptions)
	
	// Should be the same
	assert.Equal(t, original, roundTrip)
}