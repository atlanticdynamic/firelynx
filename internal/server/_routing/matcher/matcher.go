package matcher

import (
	"net/http"
)

// RequestMatcher defines the interface for request matching components.
// This interface is protocol-agnostic and can be implemented for different
// protocols (HTTP, gRPC, etc.)
type RequestMatcher interface {
	// Matches returns true if the request matches this matcher's criteria.
	// For HTTP matchers, this checks the request path.
	// For gRPC matchers (future), this would check the service name.
	Matches(r *http.Request) bool

	// ExtractParams extracts named parameters from the request.
	// For HTTP matchers, this might extract path parameters.
	// For gRPC matchers (future), this might extract metadata.
	ExtractParams(r *http.Request) map[string]string
}
