package logs

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   Format
		expected string
	}{
		{"Empty", FormatUnspecified, ""},
		{"Text", FormatText, "text"},
		{"JSON", FormatJSON, "json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.format.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{"Empty", LevelUnspecified, ""},
		{"Debug", LevelDebug, "debug"},
		{"Info", LevelInfo, "info"},
		{"Warn", LevelWarn, "warn"},
		{"Error", LevelError, "error"},
		{"Fatal", LevelFatal, "fatal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.level.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormat_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		format   Format
		expected bool
	}{
		{"Empty", FormatUnspecified, true},
		{"Text", FormatText, true},
		{"JSON", FormatJSON, true},
		{"Invalid", Format("invalid"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.format.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected bool
	}{
		{"Empty", LevelUnspecified, true},
		{"Debug", LevelDebug, true},
		{"Info", LevelInfo, true},
		{"Warn", LevelWarn, true},
		{"Error", LevelError, true},
		{"Fatal", LevelFatal, true},
		{"Invalid", Level("invalid"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.level.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Format
		expectError bool
	}{
		{"Empty", "", FormatUnspecified, false},
		{"JSON", "json", FormatJSON, false},
		{"Text", "text", FormatText, false},
		{"Txt", "txt", FormatText, false},
		{"Invalid", "invalid", FormatUnspecified, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FormatFromString(tc.input)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Level
		expectError bool
	}{
		{"Empty", "", LevelUnspecified, false},
		{"Debug", "debug", LevelDebug, false},
		{"Info", "info", LevelInfo, false},
		{"Warn", "warn", LevelWarn, false},
		{"Warning", "warning", LevelWarn, false},
		{"Error", "error", LevelError, false},
		{"Fatal", "fatal", LevelFatal, false},
		{"Invalid", "invalid", LevelUnspecified, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := LevelFromString(tc.input)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFromProto(t *testing.T) {
	tests := []struct {
		name     string
		input    *pb.LogOptions
		expected Config
	}{
		{
			name:     "Nil",
			input:    nil,
			expected: Config{},
		},
		{
			name:  "Empty",
			input: &pb.LogOptions{},
			expected: Config{
				Format: FormatUnspecified,
				Level:  LevelUnspecified,
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
			expected: Config{
				Format: FormatJSON,
				Level:  LevelInfo,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FromProto(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfig_ToProto(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected *pb.LogOptions
	}{
		{
			name:  "Empty",
			input: Config{},
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
			input: Config{
				Format: FormatJSON,
				Level:  LevelInfo,
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
	original := Config{
		Format: FormatJSON,
		Level:  LevelWarn,
	}

	// Convert to proto
	pbLogOptions := original.ToProto()

	// Convert back to domain model
	roundTrip := FromProto(pbLogOptions)

	// Should be the same
	assert.Equal(t, original, roundTrip)
}
