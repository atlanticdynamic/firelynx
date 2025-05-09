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

// HTTPToProto converts a HTTP condition to a pb.Route_Condition oneof field
func HTTPToProto(h HTTP, route *pb.Route) {
	if route == nil {
		return
	}

	route.Condition = &pb.Route_HttpPath{
		HttpPath: h.Path,
	}
}

// GRPCToProto converts a GRPC condition to a pb.Route_Condition oneof field
func GRPCToProto(g GRPC, route *pb.Route) {
	if route == nil {
		return
	}

	route.Condition = &pb.Route_GrpcService{
		GrpcService: g.Service,
	}
}

// ToProto converts a Condition to a pb.Route_Condition (oneof field)
func ToProto(cond Condition, route *pb.Route) {
	if cond == nil || route == nil {
		return
	}

	switch c := cond.(type) {
	case HTTP:
		HTTPToProto(c, route)
	case GRPC:
		GRPCToProto(c, route)
	}
}
