//nolint:dupl
package conditions

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// HTTP contains HTTP-specific route condition configuration
type HTTP struct {
	Path string
}

// NewHTTP creates a new HTTP path condition
func NewHTTP(path string) HTTP {
	return HTTP{
		Path: path,
	}
}

// Type returns the condition type
func (h HTTP) Type() Type { return TypeHTTP }

// Value returns the path value
func (h HTTP) Value() string { return h.Path }

// Validate checks if the HTTP condition is valid
func (h HTTP) Validate() error {
	if h.Path == "" {
		return fmt.Errorf("%w: %w", ErrInvalidHTTPCondition, ErrEmptyValue)
	}

	// Additional validation logic can be added here
	// For example, check if the path starts with '/'
	if !strings.HasPrefix(h.Path, "/") {
		return fmt.Errorf("%w: path must start with '/'", ErrInvalidHTTPCondition)
	}

	return nil
}

// String returns a string representation of the HTTP condition
func (h HTTP) String() string {
	return fmt.Sprintf("HTTP Path: %s", h.Path)
}

// ToTree returns a tree representation of the HTTP condition
func (h HTTP) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("HTTP Path Condition")
	tree.AddChild(fmt.Sprintf("Path: %s", h.Path))
	return tree
}
