package routes

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/robbyt/protobaggins"
)

// ToProto converts a Route to a protobuf Route
func (r *Route) ToProto() *pb.Route {
	route := &pb.Route{
		AppId: protobaggins.StringToProto(r.AppID),
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

	// Convert middlewares if present
	if len(r.Middlewares) > 0 {
		route.Middlewares = r.Middlewares.ToProto()
	}

	return route
}

// RouteFromProto converts a pb.Route to a Route
func RouteFromProto(r *pb.Route) Route {
	if r == nil {
		return Route{}
	}

	route := Route{
		AppID: protobaggins.StringFromProto(r.AppId),
	}

	// Convert static data
	if r.StaticData != nil && len(r.StaticData.Data) > 0 {
		route.StaticData = protobaggins.StructValuesToMap(r.StaticData.Data)
	}

	// Convert condition using the conditions package
	route.Condition = conditions.FromProto(r)

	// Convert middlewares if present
	if len(r.Middlewares) > 0 {
		middlewares, err := middleware.FromProto(r.Middlewares)
		if err != nil {
			// Log error or handle appropriately - for now we'll just skip
			// In a real implementation, you might want to return an error
		} else {
			route.Middlewares = middlewares
		}
	}

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
