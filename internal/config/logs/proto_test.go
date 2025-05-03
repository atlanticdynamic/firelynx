package logs

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestProtoFormatConversions(t *testing.T) {
	t.Parallel()

	// Test all format conversions from proto to domain
	formatTests := []struct {
		protoFormat  pb.LogFormat
		domainFormat Format
	}{
		{pb.LogFormat_LOG_FORMAT_UNSPECIFIED, FormatUnspecified},
		{pb.LogFormat_LOG_FORMAT_JSON, FormatJSON},
		{pb.LogFormat_LOG_FORMAT_TXT, FormatText},
		{pb.LogFormat(999), FormatUnspecified}, // Test invalid/unknown enum value
	}

	for _, tc := range formatTests {
		t.Run(tc.protoFormat.String(), func(t *testing.T) {
			result := protoFormatToFormat(tc.protoFormat)
			assert.Equal(t, tc.domainFormat, result)

			// Round trip test (except for unknown values)
			if tc.protoFormat != pb.LogFormat(999) {
				roundTrip := formatToProto(result)
				assert.Equal(t, tc.protoFormat, roundTrip)
			}
		})
	}

	// Test all domain formats to proto
	domainToProtoTests := []struct {
		domainFormat Format
		protoFormat  pb.LogFormat
	}{
		{FormatUnspecified, pb.LogFormat_LOG_FORMAT_UNSPECIFIED},
		{FormatJSON, pb.LogFormat_LOG_FORMAT_JSON},
		{FormatText, pb.LogFormat_LOG_FORMAT_TXT},
		{Format("custom"), pb.LogFormat_LOG_FORMAT_UNSPECIFIED}, // Test invalid domain format
	}

	for _, tc := range domainToProtoTests {
		t.Run(string(tc.domainFormat), func(t *testing.T) {
			result := formatToProto(tc.domainFormat)
			assert.Equal(t, tc.protoFormat, result)
		})
	}
}

func TestProtoLevelConversions(t *testing.T) {
	t.Parallel()

	// Test all level conversions from proto to domain
	levelTests := []struct {
		protoLevel  pb.LogLevel
		domainLevel Level
	}{
		{pb.LogLevel_LOG_LEVEL_UNSPECIFIED, LevelUnspecified},
		{pb.LogLevel_LOG_LEVEL_DEBUG, LevelDebug},
		{pb.LogLevel_LOG_LEVEL_INFO, LevelInfo},
		{pb.LogLevel_LOG_LEVEL_WARN, LevelWarn},
		{pb.LogLevel_LOG_LEVEL_ERROR, LevelError},
		{pb.LogLevel_LOG_LEVEL_FATAL, LevelFatal},
		{pb.LogLevel(999), LevelUnspecified}, // Test invalid/unknown enum value
	}

	for _, tc := range levelTests {
		t.Run(tc.protoLevel.String(), func(t *testing.T) {
			result := protoLevelToLevel(tc.protoLevel)
			assert.Equal(t, tc.domainLevel, result)

			// Round trip test (except for unknown values)
			if tc.protoLevel != pb.LogLevel(999) {
				roundTrip := levelToProto(result)
				assert.Equal(t, tc.protoLevel, roundTrip)
			}
		})
	}

	// Test all domain levels to proto
	domainToProtoTests := []struct {
		domainLevel Level
		protoLevel  pb.LogLevel
	}{
		{LevelUnspecified, pb.LogLevel_LOG_LEVEL_UNSPECIFIED},
		{LevelDebug, pb.LogLevel_LOG_LEVEL_DEBUG},
		{LevelInfo, pb.LogLevel_LOG_LEVEL_INFO},
		{LevelWarn, pb.LogLevel_LOG_LEVEL_WARN},
		{LevelError, pb.LogLevel_LOG_LEVEL_ERROR},
		{LevelFatal, pb.LogLevel_LOG_LEVEL_FATAL},
		{Level("trace"), pb.LogLevel_LOG_LEVEL_UNSPECIFIED}, // Test invalid domain level
	}

	for _, tc := range domainToProtoTests {
		t.Run(string(tc.domainLevel), func(t *testing.T) {
			result := levelToProto(tc.domainLevel)
			assert.Equal(t, tc.protoLevel, result)
		})
	}
}

func TestNilPointerHandling(t *testing.T) {
	t.Parallel()

	// Test nil format pointer
	pbLog := &pb.LogOptions{
		// Level is set but Format is nil
		Level: func() *pb.LogLevel {
			l := pb.LogLevel_LOG_LEVEL_INFO
			return &l
		}(),
	}
	result := FromProto(pbLog)
	assert.Equal(t, FormatUnspecified, result.Format)
	assert.Equal(t, LevelInfo, result.Level)

	// Test nil level pointer
	pbLog = &pb.LogOptions{
		// Format is set but Level is nil
		Format: func() *pb.LogFormat {
			f := pb.LogFormat_LOG_FORMAT_JSON
			return &f
		}(),
	}
	result = FromProto(pbLog)
	assert.Equal(t, FormatJSON, result.Format)
	assert.Equal(t, LevelUnspecified, result.Level)

	// Test both nil pointers
	pbLog = &pb.LogOptions{}
	result = FromProto(pbLog)
	assert.Equal(t, FormatUnspecified, result.Format)
	assert.Equal(t, LevelUnspecified, result.Level)
}

func TestEdgeCaseConversions(t *testing.T) {
	t.Parallel()

	// Test conversion with empty but non-nil proto
	pbLog := &pb.LogOptions{}
	result := FromProto(pbLog)
	assert.Equal(t, Config{Format: FormatUnspecified, Level: LevelUnspecified}, result)

	// Test conversion of empty Config to proto
	empty := Config{}
	proto := empty.ToProto()
	assert.NotNil(t, proto)
	assert.NotNil(t, proto.Format)
	assert.NotNil(t, proto.Level)
	assert.Equal(t, pb.LogFormat_LOG_FORMAT_UNSPECIFIED, proto.GetFormat())
	assert.Equal(t, pb.LogLevel_LOG_LEVEL_UNSPECIFIED, proto.GetLevel())

	// Test conversion of invalid Config to proto
	invalid := Config{
		Format: Format("yaml"),
		Level:  Level("trace"),
	}
	proto = invalid.ToProto()
	assert.NotNil(t, proto)
	assert.Equal(t, pb.LogFormat_LOG_FORMAT_UNSPECIFIED, proto.GetFormat())
	assert.Equal(t, pb.LogLevel_LOG_LEVEL_UNSPECIFIED, proto.GetLevel())
}
