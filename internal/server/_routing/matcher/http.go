// Package matcher implements request matching functionality for different protocols.
package matcher

import (
	"net/http"
	"strings"
)

// HTTPPathMatcher matches HTTP requests based on path prefixes.
type HTTPPathMatcher struct {
	pathPrefix string
}

// NewHTTPPathMatcher creates a new HTTP path matcher for the given path prefix.
func NewHTTPPathMatcher(pathPrefix string) *HTTPPathMatcher {
	return &HTTPPathMatcher{
		pathPrefix: pathPrefix,
	}
}

// Matches checks if the HTTP request path matches this matcher's path prefix.
func (m *HTTPPathMatcher) Matches(r *http.Request) bool {
	if r == nil {
		return false
	}

	return strings.HasPrefix(r.URL.Path, m.pathPrefix)
}

// ExtractParams extracts path parameters from the request path.
// For prefix matching, it returns an empty map as no parameters are captured.
// This method is included for future expansion to support path parameters.
func (m *HTTPPathMatcher) ExtractParams(r *http.Request) map[string]string {
	return map[string]string{}
}

// PathPrefix returns the path prefix used by this matcher.
func (m *HTTPPathMatcher) PathPrefix() string {
	return m.pathPrefix
}
