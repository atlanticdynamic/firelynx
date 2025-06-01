package conditions

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// FromProto creates the appropriate condition based on a protobuf Route
func FromProto(route *pb.Route) Condition {
	if route == nil {
		return nil
	}

	// Handle HTTP rule
	if httpRule := route.GetHttp(); httpRule != nil {
		pathPrefix := ""
		if httpRule.PathPrefix != nil {
			pathPrefix = *httpRule.PathPrefix
		}

		method := ""
		if httpRule.Method != nil {
			method = *httpRule.Method
		}

		return NewHTTP(pathPrefix, method)
	}

	// No condition found
	return nil
}

// ToProto converts a Condition to the appropriate protocol-specific rule
func ToProto(cond Condition, route *pb.Route) {
	if cond == nil || route == nil {
		return
	}

	switch c := cond.(type) {
	case HTTP:
		httpRule := &pb.HttpRule{
			PathPrefix: &c.PathPrefix,
		}
		if c.Method != "" {
			httpRule.Method = &c.Method
		}
		route.Rule = &pb.Route_Http{Http: httpRule}
	}
}
