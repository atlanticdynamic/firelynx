package staticdata

import "fmt"

// String returns a string representation of the StaticDataMergeMode.
func (m StaticDataMergeMode) String() string {
	switch m {
	case StaticDataMergeModeUnspecified:
		return "unspecified"
	case StaticDataMergeModeLast:
		return "last"
	case StaticDataMergeModeUnique:
		return "unique"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

// String returns a string representation of the StaticData.
func (sd StaticData) String() string {
	if sd.Data == nil {
		return fmt.Sprintf("StaticData{data: nil, merge_mode: %s}", sd.MergeMode)
	}
	return fmt.Sprintf("StaticData{data: %d items, merge_mode: %s}", len(sd.Data), sd.MergeMode)
}
