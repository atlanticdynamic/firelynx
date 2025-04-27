package config

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// convertProtoValueToInterface converts a protobuf structpb.Value to a Go interface{}
func convertProtoValueToInterface(v *structpb.Value) interface{} {
	if v == nil {
		return nil
	}

	switch v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return v.GetNumberValue()
	case *structpb.Value_StringValue:
		return v.GetStringValue()
	case *structpb.Value_BoolValue:
		return v.GetBoolValue()
	case *structpb.Value_ListValue:
		list := v.GetListValue().GetValues()
		result := make([]interface{}, len(list))
		for i, item := range list {
			result[i] = convertProtoValueToInterface(item)
		}
		return result
	case *structpb.Value_StructValue:
		m := v.GetStructValue().GetFields()
		result := make(map[string]interface{})
		for k, v := range m {
			result[k] = convertProtoValueToInterface(v)
		}
		return result
	default:
		return nil
	}
}
