package staticdata

import (
	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	"github.com/robbyt/protobaggins"
)

// ToProto converts a StaticData to its protocol buffer representation.
func (sd *StaticData) ToProto() *pbData.StaticData {
	if sd == nil {
		return nil
	}

	proto := &pbData.StaticData{
		MergeMode: staticDataMergeModeToProto(sd.MergeMode),
	}

	// Convert the data map if it exists
	if sd.Data != nil {
		proto.Data = protobaggins.MapToStructValues(sd.Data)
	}

	return proto
}

// FromProto creates a StaticData from its protocol buffer representation.
func FromProto(proto *pbData.StaticData) (*StaticData, error) {
	if proto == nil {
		return nil, nil
	}

	sd := &StaticData{
		MergeMode: protoToStaticDataMergeMode(proto.GetMergeMode()),
	}

	// Convert the data map if it exists
	if proto.Data != nil {
		sd.Data = protobaggins.StructValuesToMap(proto.Data)
	}

	return sd, nil
}

// staticDataMergeModeToProto converts a StaticDataMergeMode to its protocol buffer representation.
func staticDataMergeModeToProto(mode StaticDataMergeMode) *pbData.StaticData_MergeMode {
	var protoMode pbData.StaticData_MergeMode
	switch mode {
	case StaticDataMergeModeUnspecified:
		protoMode = pbData.StaticData_MERGE_MODE_UNSPECIFIED
	case StaticDataMergeModeLast:
		protoMode = pbData.StaticData_MERGE_MODE_LAST
	case StaticDataMergeModeUnique:
		protoMode = pbData.StaticData_MERGE_MODE_UNIQUE
	default:
		protoMode = pbData.StaticData_MERGE_MODE_UNSPECIFIED
	}
	return &protoMode
}

// protoToStaticDataMergeMode converts a protocol buffer StaticDataMergeMode to its domain model representation.
func protoToStaticDataMergeMode(mode pbData.StaticData_MergeMode) StaticDataMergeMode {
	switch mode {
	case pbData.StaticData_MERGE_MODE_UNSPECIFIED:
		return StaticDataMergeModeUnspecified
	case pbData.StaticData_MERGE_MODE_LAST:
		return StaticDataMergeModeLast
	case pbData.StaticData_MERGE_MODE_UNIQUE:
		return StaticDataMergeModeUnique
	default:
		return StaticDataMergeModeUnspecified
	}
}
