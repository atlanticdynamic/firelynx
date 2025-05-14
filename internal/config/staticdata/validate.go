package staticdata

import "fmt"

// Validate checks if the StaticData is valid.
// Currently only validates that the MergeMode is a known value.
func (sd *StaticData) Validate() error {
	// Check merge mode is valid
	if !isValidMergeMode(sd.MergeMode) {
		return NewInvalidMergeModeError(sd.MergeMode)
	}

	// Data can be nil, which is valid (empty data)
	return nil
}

// isValidMergeMode checks if the given merge mode is valid.
func isValidMergeMode(mode StaticDataMergeMode) bool {
	switch mode {
	case StaticDataMergeModeUnspecified, StaticDataMergeModeLast, StaticDataMergeModeUnique:
		return true
	default:
		return false
	}
}

// ValidateMergeMode validates the merge mode value and returns an error if invalid.
func ValidateMergeMode(mode StaticDataMergeMode) error {
	if !isValidMergeMode(mode) {
		return fmt.Errorf("%w: %v", ErrInvalidMergeMode, mode)
	}
	return nil
}
