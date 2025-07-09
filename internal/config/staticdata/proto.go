package staticdata

import (
	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/robbyt/protobaggins"
)

// ToProto converts a StaticData to its protocol buffer representation.
func (sd *StaticData) ToProto() *settingsv1alpha1.StaticData {
	if sd == nil {
		return nil
	}

	proto := &settingsv1alpha1.StaticData{
		MergeMode: staticDataMergeModeToProto(sd.MergeMode),
	}

	// Convert the data map if it exists
	if sd.Data != nil {
		proto.Data = protobaggins.MapToStructValues(sd.Data)
	}

	return proto
}

// FromProto creates a StaticData from its protocol buffer representation.
func FromProto(proto *settingsv1alpha1.StaticData) (*StaticData, error) {
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
func staticDataMergeModeToProto(mode StaticDataMergeMode) *settingsv1alpha1.StaticData_MergeMode {
	var protoMode settingsv1alpha1.StaticData_MergeMode
	switch mode {
	case StaticDataMergeModeUnspecified:
		protoMode = settingsv1alpha1.StaticData_MERGE_MODE_UNSPECIFIED
	case StaticDataMergeModeLast:
		protoMode = settingsv1alpha1.StaticData_MERGE_MODE_LAST
	case StaticDataMergeModeUnique:
		protoMode = settingsv1alpha1.StaticData_MERGE_MODE_UNIQUE
	default:
		protoMode = settingsv1alpha1.StaticData_MERGE_MODE_UNSPECIFIED
	}
	return &protoMode
}

// protoToStaticDataMergeMode converts a protocol buffer StaticDataMergeMode to its domain model representation.
func protoToStaticDataMergeMode(mode settingsv1alpha1.StaticData_MergeMode) StaticDataMergeMode {
	switch mode {
	case settingsv1alpha1.StaticData_MERGE_MODE_UNSPECIFIED:
		return StaticDataMergeModeUnspecified
	case settingsv1alpha1.StaticData_MERGE_MODE_LAST:
		return StaticDataMergeModeLast
	case settingsv1alpha1.StaticData_MERGE_MODE_UNIQUE:
		return StaticDataMergeModeUnique
	default:
		return StaticDataMergeModeUnspecified
	}
}
