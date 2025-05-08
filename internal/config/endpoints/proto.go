package endpoints

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/robbyt/protobaggins"
)

// ToProto converts an Endpoints collection to a slice of protobuf Endpoints
func (endpoints EndpointCollection) ToProto() []*pb.Endpoint {
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
		Id:          protobaggins.StringToProto(e.ID),
		ListenerIds: e.ListenerIDs,
	}

	// Convert routes if present
	if len(e.Routes) > 0 {
		pbEndpoint.Routes = e.Routes.ToProto()
	}

	return pbEndpoint
}

// FromProto converts protobuf Endpoint messages to a domain Endpoints collection.
// If no endpoints are provided, it returns nil.
// Returns an error if any endpoint validation fails (like missing ID or empty listener IDs).
func FromProto(pbEndpoints []*pb.Endpoint) (EndpointCollection, error) {
	if len(pbEndpoints) == 0 {
		return nil, nil
	}

	endpoints := make(EndpointCollection, 0, len(pbEndpoints))
	for _, e := range pbEndpoints {
		if e == nil {
			continue
		}

		id := protobaggins.StringFromProto(e.Id)

		if id == "" {
			return nil, fmt.Errorf("endpoint has nil or empty ID")
		}

		if len(e.ListenerIds) == 0 {
			return nil, fmt.Errorf("endpoint '%s' has empty listener IDs", id)
		}

		ep := Endpoint{
			ID:          id,
			ListenerIDs: e.ListenerIds,
		}

		// Convert routes
		if len(e.Routes) > 0 {
			ep.Routes = routes.FromProto(e.Routes)
		}

		endpoints = append(endpoints, ep)
	}

	if len(endpoints) == 0 {
		return nil, nil
	}

	return endpoints, nil
}
