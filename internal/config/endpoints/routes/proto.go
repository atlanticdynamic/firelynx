package routes

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/robbyt/protobaggins"
)

// ToProto converts a Route to a protobuf Route
func (r *Route) ToProto() *pb.Route {
	route := &pb.Route{
		AppId: &r.AppID,
	}

	// Convert static data if present
	if r.StaticData != nil {
		route.StaticData = &pb.StaticData{
			Data: protobaggins.MapToStructValues(r.StaticData),
		}
	}

	// Convert condition using the conditions package
	if r.Condition != nil {
		conditions.ToProto(r.Condition, route)
	}

	return route
}

// RouteFromProto converts a pb.Route to a Route
func RouteFromProto(r *pb.Route) Route {
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
		route.StaticData = protobaggins.StructValuesToMap(r.StaticData.Data)
	}

	// Convert condition using the conditions package
	route.Condition = conditions.FromProto(r)

	return route
}

// ToProto converts Routes to a slice of protobuf Routes
func (r RouteCollection) ToProto() []*pb.Route {
	pbRoutes := make([]*pb.Route, 0, len(r))
	for i := range r {
		pbRoute := r[i].ToProto()
		pbRoutes = append(pbRoutes, pbRoute)
	}
	return pbRoutes
}

// FromProto converts a slice of protobuf Route messages to a domain Routes collection.
func FromProto(pbRoutes []*pb.Route) RouteCollection {
	if len(pbRoutes) == 0 {
		return nil
	}

	routes := make(RouteCollection, 0, len(pbRoutes))
	for _, r := range pbRoutes {
		if r == nil {
			continue
		}
		routes = append(routes, RouteFromProto(r))
	}

	if len(routes) == 0 {
		return nil
	}

	return routes
}
