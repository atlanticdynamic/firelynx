package endpoints

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/protohelpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToProto converts an Endpoints collection to a slice of protobuf Endpoints
func (endpoints Endpoints) ToProto() []*pb.Endpoint {
	pbEndpoints := make([]*pb.Endpoint, 0, len(endpoints))
	for _, e := range endpoints {
		pbEndpoint := e.ToProto()
		pbEndpoints = append(pbEndpoints, pbEndpoint)
	}
	return pbEndpoints
}

// ToProto converts an Endpoint to a protobuf Endpoint
func (e *Endpoint) ToProto() *pb.Endpoint {
	pbEndpoint := &pb.Endpoint{
		Id:          &e.ID,
		ListenerIds: e.ListenerIDs,
	}

	// Convert routes
	for _, r := range e.Routes {
		pbRoute := r.ToProto()
		pbEndpoint.Routes = append(pbEndpoint.Routes, pbRoute)
	}

	return pbEndpoint
}

// ToProto converts a Route to a protobuf Route
func (r *Route) ToProto() *pb.Route {
	route := &pb.Route{
		AppId: &r.AppID,
	}

	// Convert static data if present
	if r.StaticData != nil {
		route.StaticData = &pb.StaticData{
			Data: make(map[string]*structpb.Value),
		}
		for k, v := range r.StaticData {
			val, err := structpb.NewValue(v)
			if err == nil {
				route.StaticData.Data[k] = val
			}
		}
	}

	// Convert condition
	switch cond := r.Condition.(type) {
	case HTTPPathCondition:
		route.Condition = &pb.Route_HttpPath{
			HttpPath: cond.Path,
		}
	case GRPCServiceCondition:
		route.Condition = &pb.Route_GrpcService{
			GrpcService: cond.Service,
		}
	}

	return route
}

// NewEndpointsFromProto converts one or more protobuf Endpoints to an Endpoints collection.
// It can handle a single Endpoint or multiple Endpoints via variadic arguments.
// If no endpoints are provided, it returns nil.
// If a single endpoint is provided, it returns an Endpoints collection with one item.
// If multiple endpoints are provided, it returns an Endpoints collection with all items.
func NewEndpointsFromProto(pbEndpoints ...*pb.Endpoint) Endpoints {
	if len(pbEndpoints) == 0 {
		return nil
	}

	endpoints := make(Endpoints, 0, len(pbEndpoints))
	for _, e := range pbEndpoints {
		if e == nil {
			continue
		}

		var id string
		if e.Id != nil {
			id = *e.Id
		}

		ep := Endpoint{
			ID:          id,
			ListenerIDs: e.ListenerIds,
		}

		// Convert routes
		if len(e.Routes) > 0 {
			ep.Routes = make([]Route, 0, len(e.Routes))
			for _, r := range e.Routes {
				route := NewRouteFromProto(r)
				ep.Routes = append(ep.Routes, route)
			}
		}

		endpoints = append(endpoints, ep)
	}

	if len(endpoints) == 0 {
		return nil
	}

	return endpoints
}

// NewRouteFromProto converts a pb.Route to a Route
func NewRouteFromProto(r *pb.Route) Route {
	if r == nil {
		return Route{}
	}

	var appID string
	if r.AppId != nil {
		appID = *r.AppId
	}

	route := Route{
		AppID: appID,
	}

	// Convert static data
	if r.StaticData != nil && len(r.StaticData.Data) > 0 {
		route.StaticData = make(map[string]any)
		for k, v := range r.StaticData.Data {
			route.StaticData[k] = protohelpers.ConvertProtoValueToInterface(v)
		}
	}

	// Convert condition
	if path := r.GetHttpPath(); path != "" {
		route.Condition = HTTPPathCondition{
			Path: path,
		}
	} else if service := r.GetGrpcService(); service != "" {
		route.Condition = GRPCServiceCondition{
			Service: service,
		}
	}

	return route
}
