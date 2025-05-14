package routes

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	PathPrefix string
	Method     string
	AppID      string
	StaticData map[string]any
}

// GRPCRoute represents a gRPC-specific route derived from a domain route
type GRPCRoute struct {
	Service    string
	Method     string
	AppID      string
	StaticData map[string]any
}
