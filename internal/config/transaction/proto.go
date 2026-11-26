// Package transaction provides domain model for configuration transactions
package transaction

import (
	"log/slog"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/robbyt/go-loglater/storage"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProto converts an internal ConfigTransaction to protobuf format
func (tx *ConfigTransaction) ToProto() *pb.ConfigTransaction {
	if tx == nil {
		return nil
	}

	// Convert source type
	var source pb.ConfigTransaction_Source
	switch tx.Source {
	case SourceFile:
		source = pb.ConfigTransaction_SOURCE_FILE
	case SourceAPI:
		source = pb.ConfigTransaction_SOURCE_API
	case SourceTest:
		source = pb.ConfigTransaction_SOURCE_TEST
	default:
		source = pb.ConfigTransaction_SOURCE_UNSPECIFIED
	}

	// Get log records from the transaction using GetLogs()
	storageRecords := tx.GetLogs()
	logs := make([]*pb.LogRecord, 0, len(storageRecords))
	for _, record := range storageRecords {
		logs = append(logs, convertStorageRecordToProto(record))
	}

	// Convert domain config to protobuf
	var config *pb.ServerConfig
	if tx.domainConfig != nil {
		config = tx.domainConfig.ToProto()
	}

	return &pb.ConfigTransaction{
		Id:           new(tx.ID.String()),
		Source:       &source,
		SourceDetail: new(tx.SourceDetail),
		RequestId:    new(tx.RequestID),
		CreatedAt:    timestamppb.New(tx.CreatedAt),
		State:        new(tx.GetState()),
		IsValid:      new(tx.IsValid.Load()),
		Logs:         logs,
		Config:       config,
	}
}

// attrToValue converts an slog.Attr to a protobuf Value
func attrToValue(a slog.Attr) (*structpb.Value, error) {
	switch a.Value.Kind() {
	case slog.KindBool:
		return structpb.NewBoolValue(a.Value.Bool()), nil
	case slog.KindInt64:
		return structpb.NewNumberValue(float64(a.Value.Int64())), nil
	case slog.KindFloat64:
		return structpb.NewNumberValue(a.Value.Float64()), nil
	case slog.KindString:
		return structpb.NewStringValue(a.Value.String()), nil
	default:
		// For other types, convert to string
		return structpb.NewStringValue(a.Value.String()), nil
	}
}

// convertStorageRecordToProto converts a storage.Record to a pb.LogRecord
func convertStorageRecordToProto(record storage.Record) *pb.LogRecord {
	// Convert slog level to protobuf level
	var level pb.LogRecord_Level
	switch {
	case record.Level < slog.LevelDebug:
		level = pb.LogRecord_LEVEL_DEBUG
	case record.Level < slog.LevelInfo:
		level = pb.LogRecord_LEVEL_DEBUG
	case record.Level < slog.LevelWarn:
		level = pb.LogRecord_LEVEL_INFO
	case record.Level < slog.LevelError:
		level = pb.LogRecord_LEVEL_WARN
	default:
		level = pb.LogRecord_LEVEL_ERROR
	}

	// Convert attributes to map
	attrs := make(map[string]*structpb.Value)
	for _, attr := range record.Attrs {
		if val, err := attrToValue(attr); err == nil {
			attrs[attr.Key] = val
		}
	}

	return &pb.LogRecord{
		Time:    timestamppb.New(record.Time),
		Level:   &level,
		Message: new(record.Message),
		Attrs:   attrs,
	}
}
