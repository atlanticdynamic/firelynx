package protohelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertProtoValueToInterface(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result := ConvertProtoValueToInterface(nil)
		assert.Nil(t, result)
	})

	t.Run("null value", func(t *testing.T) {
		nullValue, err := structpb.NewValue(nil)
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(nullValue)
		assert.Nil(t, result)
	})

	t.Run("unknown kind", func(t *testing.T) {
		// Create a Value with empty Kind for testing default case
		unknownValue := &structpb.Value{}
		result := ConvertProtoValueToInterface(unknownValue)
		assert.Nil(t, result)
	})

	t.Run("number value", func(t *testing.T) {
		numberValue, err := structpb.NewValue(42.5)
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(numberValue)
		assert.Equal(t, 42.5, result)
	})

	t.Run("string value", func(t *testing.T) {
		stringValue, err := structpb.NewValue("test string")
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(stringValue)
		assert.Equal(t, "test string", result)
	})

	t.Run("bool value", func(t *testing.T) {
		boolValue, err := structpb.NewValue(true)
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(boolValue)
		assert.Equal(t, true, result)
	})

	t.Run("list value", func(t *testing.T) {
		listValue, err := structpb.NewValue([]interface{}{1, "two", true})
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(listValue)
		expected := []interface{}{float64(1), "two", true}
		assert.Equal(t, expected, result)
	})

	t.Run("list with nested values", func(t *testing.T) {
		listValue, err := structpb.NewValue([]interface{}{
			map[string]interface{}{"nested": "value"},
			[]interface{}{1, 2, 3},
		})
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(listValue)
		expected := []interface{}{
			map[string]interface{}{"nested": "value"},
			[]interface{}{float64(1), float64(2), float64(3)},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("struct value", func(t *testing.T) {
		structValue, err := structpb.NewValue(map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		})
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(structValue)
		expected := map[string]interface{}{
			"key1": "value1",
			"key2": float64(42),
			"key3": true,
		}
		assert.Equal(t, expected, result)
	})

	t.Run("struct with nested values", func(t *testing.T) {
		structValue, err := structpb.NewValue(map[string]interface{}{
			"nested": map[string]interface{}{
				"array": []interface{}{1, 2, 3},
			},
		})
		assert.NoError(t, err)
		result := ConvertProtoValueToInterface(structValue)
		expected := map[string]interface{}{
			"nested": map[string]interface{}{
				"array": []interface{}{float64(1), float64(2), float64(3)},
			},
		}
		assert.Equal(t, expected, result)
	})
}
