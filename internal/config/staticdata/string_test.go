package staticdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticDataMergeModeString(t *testing.T) {
	tests := []struct {
		name     string
		mode     StaticDataMergeMode
		expected string
	}{
		{
			name:     "Unspecified",
			mode:     StaticDataMergeModeUnspecified,
			expected: "unspecified",
		},
		{
			name:     "Last",
			mode:     StaticDataMergeModeLast,
			expected: "last",
		},
		{
			name:     "Unique",
			mode:     StaticDataMergeModeUnique,
			expected: "unique",
		},
		{
			name:     "Invalid",
			mode:     StaticDataMergeMode(999),
			expected: "unknown(999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticDataString(t *testing.T) {
	t.Run("WithData", func(t *testing.T) {
		sd := StaticData{
			Data: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			MergeMode: StaticDataMergeModeLast,
		}
		assert.Equal(t, "StaticData{data: 2 items, merge_mode: last}", sd.String())
	})

	t.Run("WithoutData", func(t *testing.T) {
		sd := StaticData{
			Data:      nil,
			MergeMode: StaticDataMergeModeLast,
		}
		assert.Equal(t, "StaticData{data: nil, merge_mode: last}", sd.String())
	})
}
