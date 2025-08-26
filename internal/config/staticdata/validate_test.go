package staticdata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		staticData  StaticData
		expectError bool
	}{
		{
			name: "Valid with unspecified merge mode",
			staticData: StaticData{
				Data:      map[string]any{"key": "value"},
				MergeMode: StaticDataMergeModeUnspecified,
			},
			expectError: false,
		},
		{
			name: "Valid with last merge mode",
			staticData: StaticData{
				Data:      map[string]any{"key": "value"},
				MergeMode: StaticDataMergeModeLast,
			},
			expectError: false,
		},
		{
			name: "Valid with unique merge mode",
			staticData: StaticData{
				Data:      map[string]any{"key": "value"},
				MergeMode: StaticDataMergeModeUnique,
			},
			expectError: false,
		},
		{
			name: "Valid with nil data",
			staticData: StaticData{
				Data:      nil,
				MergeMode: StaticDataMergeModeUnspecified,
			},
			expectError: false,
		},
		{
			name: "Invalid merge mode",
			staticData: StaticData{
				Data:      map[string]any{"key": "value"},
				MergeMode: StaticDataMergeMode(999),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staticData.Validate()
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidMergeMode)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMergeMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        StaticDataMergeMode
		expectError bool
	}{
		{
			name:        "Valid unspecified",
			mode:        StaticDataMergeModeUnspecified,
			expectError: false,
		},
		{
			name:        "Valid last",
			mode:        StaticDataMergeModeLast,
			expectError: false,
		},
		{
			name:        "Valid unique",
			mode:        StaticDataMergeModeUnique,
			expectError: false,
		},
		{
			name:        "Invalid mode",
			mode:        StaticDataMergeMode(999),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMergeMode(tt.mode)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidMergeMode)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
