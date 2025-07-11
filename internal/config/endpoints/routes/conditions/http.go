package conditions

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// HTTP contains HTTP-specific route condition configuration
type HTTP struct {
	PathPrefix string `env_interpolation:"yes"`
	Method     string `env_interpolation:"no"`
}

// NewHTTP creates a new HTTP path condition
func NewHTTP(pathPrefix string, method string) HTTP {
	return HTTP{
		PathPrefix: pathPrefix,
		Method:     method,
	}
}

// Type returns the condition type
func (h HTTP) Type() Type { return TypeHTTP }

// Value returns a representative value
func (h HTTP) Value() string {
	if h.Method != "" {
		return h.PathPrefix + " (" + h.Method + ")"
	}
	return h.PathPrefix
}

// Validate checks if the HTTP condition is valid
func (h HTTP) Validate() error {
	if h.PathPrefix == "" {
		return fmt.Errorf("%w: %w", ErrInvalidHTTPCondition, ErrEmptyValue)
	}

	// Check if the path starts with '/'
	if !strings.HasPrefix(h.PathPrefix, "/") {
		return fmt.Errorf("%w: path must start with '/'", ErrInvalidHTTPCondition)
	}

	return nil
}

// String returns a string representation of the HTTP condition
func (h HTTP) String() string {
	if h.Method != "" {
		return fmt.Sprintf("HTTP: %s %s", h.Method, h.PathPrefix)
	}
	return fmt.Sprintf("HTTP Path: %s", h.PathPrefix)
}

// ToTree returns a tree representation of the HTTP condition
func (h HTTP) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("HTTP Rule")
	tree.AddChild(fmt.Sprintf("Path Prefix: %s", h.PathPrefix))
	if h.Method != "" {
		tree.AddChild(fmt.Sprintf("Method: %s", h.Method))
	}
	return tree
}
