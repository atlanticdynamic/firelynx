package routes

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	PathPrefix string
	Method     string
	AppID      string
	StaticData map[string]any
}
