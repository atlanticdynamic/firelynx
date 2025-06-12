package cfgservice

import (
	"log/slog"
	"os"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/robbyt/go-loglater/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertStorageRecordToProto(t *testing.T) {
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
				Message: stringPtr("debug message"),
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
				Message: stringPtr("info message"),
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
				Message: stringPtr("warning message"),
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
				Message: stringPtr("error message"),
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

func stringPtr(s string) *string {
	return &s
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
	tx, err := transaction.New(
		transaction.SourceTest,
		"TestGetLogsVsPlaybackLogs",
		"test-request",
		cfg,
		handler,
	)
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

	// The conversion should produce valid protobuf records
	for i, record := range logRecords {
		assert.NotNil(t, record.Time, "record %d should have timestamp", i)
		assert.NotNil(t, record.Level, "record %d should have level", i)
		assert.NotNil(t, record.Message, "record %d should have message", i)
		assert.NotNil(t, record.Attrs, "record %d should have attrs", i)
	}
}
