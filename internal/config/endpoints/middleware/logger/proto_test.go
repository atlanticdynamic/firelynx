package logger

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsoleLogger_ToProto(t *testing.T) {
	t.Parallel()

	t.Run("Complete configuration", func(t *testing.T) {
		t.Parallel()

		config := &ConsoleLogger{
			Options: LogOptionsGeneral{
				Format: FormatJSON,
				Level:  LevelDebug,
			},
			Fields: LogOptionsHTTP{
				Method:      true,
				Path:        true,
				ClientIP:    true,
				QueryParams: true,
				Protocol:    true,
				Host:        true,
				Scheme:      true,
				StatusCode:  true,
				Duration:    true,
				Request: DirectionConfig{
					Enabled:        true,
					Body:           true,
					MaxBodySize:    1024,
					BodySize:       true,
					Headers:        true,
					IncludeHeaders: []string{"Content-Type", "Authorization"},
					ExcludeHeaders: []string{"X-Secret"},
				},
				Response: DirectionConfig{
					Enabled:        true,
					Body:           true,
					MaxBodySize:    2048,
					BodySize:       true,
					Headers:        false,
					IncludeHeaders: []string{"Content-Type"},
					ExcludeHeaders: []string{"Set-Cookie"},
				},
			},
			IncludeOnlyPaths:   []string{"/api", "/v1"},
			ExcludePaths:       []string{"/health", "/metrics"},
			IncludeOnlyMethods: []string{"GET", "POST"},
			ExcludeMethods:     []string{"OPTIONS"},
		}

		result := config.ToProto()
		require.NotNil(t, result)

		pbConfig, ok := result.(*pb.ConsoleLoggerConfig)
		require.True(t, ok)

		// Verify general options
		assert.Equal(t, pb.LogOptionsGeneral_FORMAT_JSON, pbConfig.Options.GetFormat())
		assert.Equal(t, pb.LogOptionsGeneral_LEVEL_DEBUG, pbConfig.Options.GetLevel())

		// Verify HTTP fields
		assert.True(t, pbConfig.Fields.GetMethod())
		assert.True(t, pbConfig.Fields.GetPath())
		assert.True(t, pbConfig.Fields.GetClientIp())
		assert.True(t, pbConfig.Fields.GetQueryParams())
		assert.True(t, pbConfig.Fields.GetProtocol())
		assert.True(t, pbConfig.Fields.GetHost())
		assert.True(t, pbConfig.Fields.GetScheme())
		assert.True(t, pbConfig.Fields.GetStatusCode())
		assert.True(t, pbConfig.Fields.GetDuration())

		// Verify request config
		reqConfig := pbConfig.Fields.Request
		assert.True(t, reqConfig.GetEnabled())
		assert.True(t, reqConfig.GetBody())
		assert.Equal(t, int32(1024), reqConfig.GetMaxBodySize())
		assert.True(t, reqConfig.GetBodySize())
		assert.True(t, reqConfig.GetHeaders())
		assert.Equal(t, []string{"Content-Type", "Authorization"}, reqConfig.IncludeHeaders)
		assert.Equal(t, []string{"X-Secret"}, reqConfig.ExcludeHeaders)

		// Verify response config
		respConfig := pbConfig.Fields.Response
		assert.True(t, respConfig.GetEnabled())
		assert.True(t, respConfig.GetBody())
		assert.Equal(t, int32(2048), respConfig.GetMaxBodySize())
		assert.True(t, respConfig.GetBodySize())
		assert.False(t, respConfig.GetHeaders())
		assert.Equal(t, []string{"Content-Type"}, respConfig.IncludeHeaders)
		assert.Equal(t, []string{"Set-Cookie"}, respConfig.ExcludeHeaders)

		// Verify filtering
		assert.Equal(t, []string{"/api", "/v1"}, pbConfig.IncludeOnlyPaths)
		assert.Equal(t, []string{"/health", "/metrics"}, pbConfig.ExcludePaths)
		assert.Equal(t, []string{"GET", "POST"}, pbConfig.IncludeOnlyMethods)
		assert.Equal(t, []string{"OPTIONS"}, pbConfig.ExcludeMethods)
	})

	t.Run("Minimal configuration", func(t *testing.T) {
		t.Parallel()

		config := NewConsoleLogger()

		result := config.ToProto()
		require.NotNil(t, result)

		pbConfig, ok := result.(*pb.ConsoleLoggerConfig)
		require.True(t, ok)

		// Should have default values from NewConsoleLogger()
		assert.Equal(t, pb.LogOptionsGeneral_FORMAT_JSON, pbConfig.Options.GetFormat())
		assert.Equal(t, pb.LogOptionsGeneral_LEVEL_INFO, pbConfig.Options.GetLevel())

		// Empty slices should not be set
		assert.Empty(t, pbConfig.IncludeOnlyPaths)
		assert.Empty(t, pbConfig.ExcludePaths)
		assert.Empty(t, pbConfig.IncludeOnlyMethods)
		assert.Empty(t, pbConfig.ExcludeMethods)
	})
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("Complete protobuf configuration", func(t *testing.T) {
		t.Parallel()

		pbConfig := &pb.ConsoleLoggerConfig{
			Options: &pb.LogOptionsGeneral{
				Format: func() *pb.LogOptionsGeneral_Format {
					f := pb.LogOptionsGeneral_FORMAT_JSON
					return &f
				}(),
				Level: func() *pb.LogOptionsGeneral_Level {
					l := pb.LogOptionsGeneral_LEVEL_WARN
					return &l
				}(),
			},
			Fields: &pb.LogOptionsHTTP{
				Method:      func() *bool { b := false; return &b }(),
				Path:        func() *bool { b := true; return &b }(),
				ClientIp:    func() *bool { b := true; return &b }(),
				QueryParams: func() *bool { b := false; return &b }(),
				Protocol:    func() *bool { b := true; return &b }(),
				Host:        func() *bool { b := true; return &b }(),
				Scheme:      func() *bool { b := false; return &b }(),
				StatusCode:  func() *bool { b := true; return &b }(),
				Duration:    func() *bool { b := true; return &b }(),
				Request: &pb.LogOptionsHTTP_DirectionConfig{
					Enabled:        func() *bool { b := true; return &b }(),
					Body:           func() *bool { b := true; return &b }(),
					MaxBodySize:    func() *int32 { s := int32(512); return &s }(),
					BodySize:       func() *bool { b := false; return &b }(),
					Headers:        func() *bool { b := true; return &b }(),
					IncludeHeaders: []string{"Accept", "User-Agent"},
					ExcludeHeaders: []string{"Authorization"},
				},
				Response: &pb.LogOptionsHTTP_DirectionConfig{
					Enabled:        func() *bool { b := false; return &b }(),
					Body:           func() *bool { b := false; return &b }(),
					MaxBodySize:    func() *int32 { s := int32(1024); return &s }(),
					BodySize:       func() *bool { b := true; return &b }(),
					Headers:        func() *bool { b := false; return &b }(),
					IncludeHeaders: []string{"Content-Type"},
					ExcludeHeaders: []string{},
				},
			},
			IncludeOnlyPaths:   []string{"/api/v1", "/admin"},
			ExcludePaths:       []string{"/health"},
			IncludeOnlyMethods: []string{"GET"},
			ExcludeMethods:     []string{"HEAD", "OPTIONS"},
		}

		config, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify general options
		assert.Equal(t, FormatJSON, config.Options.Format)
		assert.Equal(t, LevelWarn, config.Options.Level)

		// Verify HTTP fields
		assert.False(t, config.Fields.Method)
		assert.True(t, config.Fields.Path)
		assert.True(t, config.Fields.ClientIP)
		assert.False(t, config.Fields.QueryParams)
		assert.True(t, config.Fields.Protocol)
		assert.True(t, config.Fields.Host)
		assert.False(t, config.Fields.Scheme)
		assert.True(t, config.Fields.StatusCode)
		assert.True(t, config.Fields.Duration)

		// Verify request config
		assert.True(t, config.Fields.Request.Enabled)
		assert.True(t, config.Fields.Request.Body)
		assert.Equal(t, int32(512), config.Fields.Request.MaxBodySize)
		assert.False(t, config.Fields.Request.BodySize)
		assert.True(t, config.Fields.Request.Headers)
		assert.Equal(t, []string{"Accept", "User-Agent"}, config.Fields.Request.IncludeHeaders)
		assert.Equal(t, []string{"Authorization"}, config.Fields.Request.ExcludeHeaders)

		// Verify response config
		assert.False(t, config.Fields.Response.Enabled)
		assert.False(t, config.Fields.Response.Body)
		assert.Equal(t, int32(1024), config.Fields.Response.MaxBodySize)
		assert.True(t, config.Fields.Response.BodySize)
		assert.False(t, config.Fields.Response.Headers)
		assert.Equal(t, []string{"Content-Type"}, config.Fields.Response.IncludeHeaders)
		assert.Empty(t, config.Fields.Response.ExcludeHeaders)

		// Verify filtering
		assert.Equal(t, []string{"/api/v1", "/admin"}, config.IncludeOnlyPaths)
		assert.Equal(t, []string{"/health"}, config.ExcludePaths)
		assert.Equal(t, []string{"GET"}, config.IncludeOnlyMethods)
		assert.Equal(t, []string{"HEAD", "OPTIONS"}, config.ExcludeMethods)
	})

	t.Run("Minimal protobuf configuration", func(t *testing.T) {
		t.Parallel()

		pbConfig := &pb.ConsoleLoggerConfig{}

		config, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Should have zero values (empty protobuf means unspecified enum values map to empty string)
		assert.Equal(t, Format(""), config.Options.Format)
		assert.Equal(t, Level(""), config.Options.Level)
		assert.Empty(t, config.IncludeOnlyPaths)
		assert.Empty(t, config.ExcludePaths)
		assert.Empty(t, config.IncludeOnlyMethods)
		assert.Empty(t, config.ExcludeMethods)
	})

	t.Run("Nil protobuf configuration", func(t *testing.T) {
		t.Parallel()

		config, err := FromProto(nil)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "nil console logger config")
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("Complete configuration round trip", func(t *testing.T) {
		t.Parallel()

		original := &ConsoleLogger{
			Options: LogOptionsGeneral{
				Format: FormatJSON,
				Level:  LevelError,
			},
			Fields: LogOptionsHTTP{
				Method:      false,
				Path:        true,
				ClientIP:    true,
				QueryParams: false,
				Protocol:    true,
				Host:        false,
				Scheme:      true,
				StatusCode:  true,
				Duration:    false,
				Request: DirectionConfig{
					Enabled:        true,
					Body:           false,
					MaxBodySize:    256,
					BodySize:       true,
					Headers:        true,
					IncludeHeaders: []string{"X-Custom"},
					ExcludeHeaders: []string{"Cookie"},
				},
				Response: DirectionConfig{
					Enabled:        false,
					Body:           true,
					MaxBodySize:    512,
					BodySize:       false,
					Headers:        true,
					IncludeHeaders: nil,
					ExcludeHeaders: []string{"X-Internal"},
				},
			},
			IncludeOnlyPaths:   []string{"/secure"},
			ExcludePaths:       []string{"/public"},
			IncludeOnlyMethods: []string{"POST", "PUT"},
			ExcludeMethods:     []string{"DELETE"},
		}

		// Convert to protobuf
		protoResult := original.ToProto()
		pbConfig, ok := protoResult.(*pb.ConsoleLoggerConfig)
		require.True(t, ok)

		// Convert back to domain
		restored, err := FromProto(pbConfig)
		require.NoError(t, err)

		// Verify complete equality
		assert.Equal(t, original.Options, restored.Options)
		assert.Equal(t, original.Fields, restored.Fields)
		assert.Equal(t, original.IncludeOnlyPaths, restored.IncludeOnlyPaths)
		assert.Equal(t, original.ExcludePaths, restored.ExcludePaths)
		assert.Equal(t, original.IncludeOnlyMethods, restored.IncludeOnlyMethods)
		assert.Equal(t, original.ExcludeMethods, restored.ExcludeMethods)
	})

	t.Run("Default configuration round trip", func(t *testing.T) {
		t.Parallel()

		original := NewConsoleLogger()

		// Convert to protobuf
		protoResult := original.ToProto()
		pbConfig, ok := protoResult.(*pb.ConsoleLoggerConfig)
		require.True(t, ok)

		// Convert back to domain
		restored, err := FromProto(pbConfig)
		require.NoError(t, err)

		// Verify equality
		assert.Equal(t, original.Options, restored.Options)
		assert.Equal(t, original.Fields, restored.Fields)
		assert.Equal(t, original.IncludeOnlyPaths, restored.IncludeOnlyPaths)
		assert.Equal(t, original.ExcludePaths, restored.ExcludePaths)
		assert.Equal(t, original.IncludeOnlyMethods, restored.IncludeOnlyMethods)
		assert.Equal(t, original.ExcludeMethods, restored.ExcludeMethods)
	})
}

func TestFormatConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		domain Format
		proto  pb.LogOptionsGeneral_Format
	}{
		{FormatTxt, pb.LogOptionsGeneral_FORMAT_TXT},
		{FormatJSON, pb.LogOptionsGeneral_FORMAT_JSON},
		{FormatUnspecified, pb.LogOptionsGeneral_FORMAT_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			// Test domain to proto
			protoFormat := formatToProto(tt.domain)
			assert.Equal(t, tt.proto, protoFormat)

			// Test proto to domain
			domainFormat := formatFromProto(tt.proto)
			assert.Equal(t, tt.domain, domainFormat)
		})
	}

	t.Run("Invalid format defaults to unspecified", func(t *testing.T) {
		invalidFormat := Format("invalid")
		protoFormat := formatToProto(invalidFormat)
		assert.Equal(t, pb.LogOptionsGeneral_FORMAT_UNSPECIFIED, protoFormat)

		domainFormat := formatFromProto(pb.LogOptionsGeneral_Format(999))
		assert.Equal(t, FormatUnspecified, domainFormat)
	})
}

func TestLevelConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		domain Level
		proto  pb.LogOptionsGeneral_Level
	}{
		{LevelDebug, pb.LogOptionsGeneral_LEVEL_DEBUG},
		{LevelInfo, pb.LogOptionsGeneral_LEVEL_INFO},
		{LevelWarn, pb.LogOptionsGeneral_LEVEL_WARN},
		{LevelError, pb.LogOptionsGeneral_LEVEL_ERROR},
		{LevelFatal, pb.LogOptionsGeneral_LEVEL_FATAL},
		{LevelUnspecified, pb.LogOptionsGeneral_LEVEL_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			// Test domain to proto
			protoLevel := levelToProto(tt.domain)
			assert.Equal(t, tt.proto, protoLevel)

			// Test proto to domain
			domainLevel := levelFromProto(tt.proto)
			assert.Equal(t, tt.domain, domainLevel)
		})
	}

	t.Run("Invalid level defaults to unspecified", func(t *testing.T) {
		invalidLevel := Level("invalid")
		protoLevel := levelToProto(invalidLevel)
		assert.Equal(t, pb.LogOptionsGeneral_LEVEL_UNSPECIFIED, protoLevel)

		domainLevel := levelFromProto(pb.LogOptionsGeneral_Level(999))
		assert.Equal(t, LevelUnspecified, domainLevel)
	})
}

func TestFromProtoFormatConversion(t *testing.T) {
	t.Parallel()

	t.Run("JSON format conversion from proto", func(t *testing.T) {
		pbConfig := &pb.ConsoleLoggerConfig{
			Options: &pb.LogOptionsGeneral{
				Format: func() *pb.LogOptionsGeneral_Format {
					f := pb.LogOptionsGeneral_FORMAT_JSON
					return &f
				}(),
			},
		}

		config, err := FromProto(pbConfig)
		require.NoError(t, err)
		assert.Equal(t, FormatJSON, config.Options.Format)
	})

	t.Run("TXT format conversion from proto", func(t *testing.T) {
		pbConfig := &pb.ConsoleLoggerConfig{
			Options: &pb.LogOptionsGeneral{
				Format: func() *pb.LogOptionsGeneral_Format {
					f := pb.LogOptionsGeneral_FORMAT_TXT
					return &f
				}(),
			},
		}

		config, err := FromProto(pbConfig)
		require.NoError(t, err)
		assert.Equal(t, FormatTxt, config.Options.Format)
	})
}

func TestDirectionConfigConversion(t *testing.T) {
	t.Parallel()

	t.Run("Complete direction config", func(t *testing.T) {
		t.Parallel()

		domain := DirectionConfig{
			Enabled:        true,
			Body:           false,
			MaxBodySize:    1024,
			BodySize:       true,
			Headers:        false,
			IncludeHeaders: []string{"Content-Type", "Accept"},
			ExcludeHeaders: []string{"Authorization"},
		}

		// Convert to proto
		proto := directionConfigToProto(domain)
		assert.True(t, proto.GetEnabled())
		assert.False(t, proto.GetBody())
		assert.Equal(t, int32(1024), proto.GetMaxBodySize())
		assert.True(t, proto.GetBodySize())
		assert.False(t, proto.GetHeaders())
		assert.Equal(t, []string{"Content-Type", "Accept"}, proto.IncludeHeaders)
		assert.Equal(t, []string{"Authorization"}, proto.ExcludeHeaders)

		// Convert back to domain
		restored := directionConfigFromProto(proto)
		assert.Equal(t, domain, restored)
	})

	t.Run("Empty direction config", func(t *testing.T) {
		t.Parallel()

		domain := DirectionConfig{}

		// Convert to proto
		proto := directionConfigToProto(domain)
		assert.False(t, proto.GetEnabled())
		assert.False(t, proto.GetBody())
		assert.Equal(t, int32(0), proto.GetMaxBodySize())
		assert.False(t, proto.GetBodySize())
		assert.False(t, proto.GetHeaders())
		assert.Empty(t, proto.IncludeHeaders)
		assert.Empty(t, proto.ExcludeHeaders)

		// Convert back to domain
		restored := directionConfigFromProto(proto)
		assert.Equal(t, domain, restored)
	})
}
