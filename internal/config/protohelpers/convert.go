package protohelpers

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertProtoValueToInterface converts a protobuf structpb.Value to a Go any
func ConvertProtoValueToInterface(v *structpb.Value) any {
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
		result := make([]any, len(list))
		for i, item := range list {
			result[i] = ConvertProtoValueToInterface(item)
		}
		return result
	case *structpb.Value_StructValue:
		m := v.GetStructValue().GetFields()
		result := make(map[string]any)
		for k, v := range m {
			result[k] = ConvertProtoValueToInterface(v)
		}
		return result
	default:
		return nil
	}
}
