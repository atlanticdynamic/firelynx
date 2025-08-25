package staticdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticDataMergeModeValues(t *testing.T) {
	// Verify that our constant values match the expected values
	assert.Equal(t, StaticDataMergeModeUnspecified, StaticDataMergeMode(0))
	assert.Equal(t, StaticDataMergeModeLast, StaticDataMergeMode(1))
	assert.Equal(t, StaticDataMergeModeUnique, StaticDataMergeMode(2))
}

func TestStaticDataStructure(t *testing.T) {
	// Test creating and using a StaticData struct
	sd := StaticData{
		Data: map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
		MergeMode: StaticDataMergeModeLast,
	}

	// Check data access
	assert.Equal(t, "value1", sd.Data["key1"])
	assert.Equal(t, 42, sd.Data["key2"])
	assert.Equal(t, true, sd.Data["key3"])
	assert.Equal(t, StaticDataMergeModeLast, sd.MergeMode)
}
