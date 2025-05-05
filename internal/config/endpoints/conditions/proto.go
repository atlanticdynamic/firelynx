package conditions

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// FromProto creates the appropriate condition based on a protobuf Route
func FromProto(route *pb.Route) Condition {
	if route == nil {
		return nil
	}

	// Check for HTTP path condition
	if path := route.GetHttpPath(); path != "" {
		return NewHTTP(path)
	}

	// Check for gRPC service condition
	if service := route.GetGrpcService(); service != "" {
		return NewGRPC(service)
	}

	// No condition found
	return nil
}

// ToProto converts a Condition to a pb.Route_Condition (oneof field)
func ToProto(cond Condition, route *pb.Route) {
	if cond == nil || route == nil {
		return
	}

	switch c := cond.(type) {
	case HTTP:
		route.Condition = &pb.Route_HttpPath{
			HttpPath: c.Path,
		}
	case GRPC:
		route.Condition = &pb.Route_GrpcService{
			GrpcService: c.Service,
		}
	}
}
