package transaction

import (
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/robbyt/go-loglater/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConfigTransaction_ToProto(t *testing.T) {
	t.Parallel()
	handler := logging.SetupHandlerText("debug")

	t.Run("converts transaction to protobuf", func(t *testing.T) {
		cfg := &config.Config{Version: version.Version}
		tx, err := FromAPI("test-request", cfg, handler)
		require.NoError(t, err)

		pbTx := tx.ToProto()
		require.NotNil(t, pbTx)

		assert.Equal(t, tx.ID.String(), pbTx.GetId())
		assert.Equal(t, "test-request", pbTx.GetRequestId())
		assert.Equal(t, pb.ConfigTransaction_SOURCE_API, pbTx.GetSource())
		assert.Equal(t, tx.GetState(), pbTx.GetState())
		assert.Equal(t, tx.IsValid.Load(), pbTx.GetIsValid())
		assert.NotNil(t, pbTx.GetCreatedAt())
	})

	t.Run("returns nil for nil transaction", func(t *testing.T) {
		var tx *ConfigTransaction
		pbTx := tx.ToProto()
		assert.Nil(t, pbTx)
	})

	t.Run("includes log records", func(t *testing.T) {
		cfg := &config.Config{Version: version.Version}
		tx, err := FromTest("test-request", cfg, handler)
		require.NoError(t, err)

		// Generate some log activity
		err = tx.RunValidation()
		require.NoError(t, err)

		pbTx := tx.ToProto()
		require.NotNil(t, pbTx)

		// Should have log records from validation
		assert.NotEmpty(t, pbTx.GetLogs())
		// Verify source type is correct
		assert.Equal(t, pb.ConfigTransaction_SOURCE_TEST, pbTx.GetSource())
	})

	t.Run("converts different source types", func(t *testing.T) {
		cfg := &config.Config{Version: version.Version}

		tests := []struct {
			name           string
			source         Source
			expectedSource pb.ConfigTransaction_Source
		}{
			{"file source", SourceFile, pb.ConfigTransaction_SOURCE_FILE},
			{"api source", SourceAPI, pb.ConfigTransaction_SOURCE_API},
			{"test source", SourceTest, pb.ConfigTransaction_SOURCE_TEST},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tx, err := New(tt.source, "detail", "request", cfg, handler)
				require.NoError(t, err)

				pbTx := tx.ToProto()
				require.NotNil(t, pbTx)
				assert.Equal(t, tt.expectedSource, pbTx.GetSource())
			})
		}
	})
}

func TestConvertStorageRecordToProto(t *testing.T) {
	t.Parallel()
	testTime := time.Now()

	tests := []struct {
		name   string
		record storage.Record
		want   *pb.LogRecord
	}{
		{
			name: "debug level record",
			record: storage.Record{
				Time:    testTime,
				Level:   slog.LevelDebug,
				Message: "debug message",
				Attrs: []slog.Attr{
					slog.String("key1", "value1"),
					slog.Int("key2", 42),
				},
			},
			want: &pb.LogRecord{
				Level:   pb.LogRecord_LEVEL_DEBUG.Enum(),
				Message: proto.String("debug message"),
				Attrs: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
				},
			},
		},
		{
			name: "info level record",
			record: storage.Record{
				Time:    testTime,
				Level:   slog.LevelInfo,
				Message: "info message",
				Attrs: []slog.Attr{
					slog.Bool("success", true),
				},
			},
			want: &pb.LogRecord{
				Level:   pb.LogRecord_LEVEL_INFO.Enum(),
				Message: proto.String("info message"),
				Attrs: map[string]*structpb.Value{
					"success": structpb.NewBoolValue(true),
				},
			},
		},
		{
			name: "warn level record",
			record: storage.Record{
				Time:    testTime,
				Level:   slog.LevelWarn,
				Message: "warning message",
				Attrs:   []slog.Attr{},
			},
			want: &pb.LogRecord{
				Level:   pb.LogRecord_LEVEL_WARN.Enum(),
				Message: proto.String("warning message"),
				Attrs:   map[string]*structpb.Value{},
			},
		},
		{
			name: "error level record",
			record: storage.Record{
				Time:    testTime,
				Level:   slog.LevelError,
				Message: "error message",
				Attrs: []slog.Attr{
					slog.Float64("duration", 123.456),
				},
			},
			want: &pb.LogRecord{
				Level:   pb.LogRecord_LEVEL_ERROR.Enum(),
				Message: proto.String("error message"),
				Attrs: map[string]*structpb.Value{
					"duration": structpb.NewNumberValue(123.456),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertStorageRecordToProto(tt.record)

			require.NotNil(t, got)
			assert.Equal(t, tt.want.GetLevel(), got.GetLevel())
			assert.Equal(t, tt.want.GetMessage(), got.GetMessage())
			assert.Equal(t, len(tt.want.Attrs), len(got.Attrs))

			// Check attributes
			for key, expectedVal := range tt.want.Attrs {
				gotVal, ok := got.Attrs[key]
				require.True(t, ok, "missing attribute %s", key)
				assert.Equal(t, expectedVal.String(), gotVal.String())
			}

			// Verify timestamp is set
			assert.NotNil(t, got.Time)
			assert.Equal(t, testTime.Unix(), got.Time.Seconds)
		})
	}
}

func TestGetLogsVsPlaybackLogs(t *testing.T) {
	// Create a config transaction
	cfg := &config.Config{
		Version: "v1",
	}

	// Enable debug logging to capture all log levels
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	// Use FromTest instead of FromFile to avoid file system dependencies
	tx, err := FromTest("log-comparison-test", cfg, handler)
	require.NoError(t, err)

	// Generate some log messages through the transaction lifecycle
	// This will ensure we have logs to capture
	require.NoError(t, tx.RunValidation())
	require.NoError(t, tx.BeginExecution())
	require.NoError(t, tx.MarkSucceeded())

	// Use GetLogs() to get log records
	storageRecords := tx.GetLogs()
	t.Logf("GetLogs returned %d storage records", len(storageRecords))
	logRecords := make([]*pb.LogRecord, 0, len(storageRecords))
	for _, record := range storageRecords {
		logRecords = append(logRecords, convertStorageRecordToProto(record))
	}

	// Verify we captured log records
	assert.Greater(t, len(logRecords), 0, "GetLogs should capture log records")

	// Verify the transaction converted to protobuf correctly
	pbTx := tx.ToProto()
	require.NotNil(t, pbTx)
	assert.Equal(t, pb.ConfigTransaction_SOURCE_TEST, pbTx.GetSource())
	assert.Equal(t, "log-comparison-test", pbTx.GetSourceDetail())

	// The conversion should produce valid protobuf records
	for i, record := range logRecords {
		assert.NotNil(t, record.Time, "record %d should have timestamp", i)
		assert.NotNil(t, record.Level, "record %d should have level", i)
		assert.NotNil(t, record.Message, "record %d should have message", i)
		assert.NotNil(t, record.Attrs, "record %d should have attrs", i)
	}
}

func TestConfigTransaction_ToProto_IncludesConfig(t *testing.T) {
	handler := logging.SetupHandlerText("debug")

	t.Run("includes config when domainConfig is set", func(t *testing.T) {
		cfg := &config.Config{Version: version.Version}
		tx, err := FromAPI("test-request", cfg, handler)
		require.NoError(t, err)

		pbTx := tx.ToProto()
		require.NotNil(t, pbTx)

		// Verify config field is populated
		assert.NotNil(t, pbTx.GetConfig(), "Config field should be populated")
		assert.Equal(
			t,
			version.Version,
			pbTx.GetConfig().GetVersion(),
			"Config version should match",
		)
	})

	t.Run("config is nil when domainConfig is nil", func(t *testing.T) {
		// Create a transaction without a valid config (this is a test scenario)
		cfg := &config.Config{Version: version.Version}
		tx, err := FromAPI("test-request", cfg, handler)
		require.NoError(t, err)

		// Artificially set domainConfig to nil for testing
		tx.domainConfig = nil

		pbTx := tx.ToProto()
		require.NotNil(t, pbTx)

		// Verify config field is nil
		assert.Nil(t, pbTx.GetConfig(), "Config field should be nil when domainConfig is nil")
	})
}
