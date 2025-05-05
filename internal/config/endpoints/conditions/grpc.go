//nolint:dupl
package conditions

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// GRPC contains gRPC-specific route condition configuration
type GRPC struct {
	Service string
}

// NewGRPC creates a new gRPC service condition
func NewGRPC(service string) GRPC {
	return GRPC{
		Service: service,
	}
}

// Type returns the condition type
func (g GRPC) Type() Type { return TypeGRPC }

// Value returns the service value
func (g GRPC) Value() string { return g.Service }

// Validate checks if the gRPC condition is valid
func (g GRPC) Validate() error {
	if g.Service == "" {
		return fmt.Errorf("%w: %w", ErrInvalidGRPCCondition, ErrEmptyValue)
	}

	// Additional validation logic can be added here
	// For example, check if the service follows protobuf naming convention
	if !strings.Contains(g.Service, ".") {
		return fmt.Errorf(
			"%w: service should be fully-qualified (package.Service)",
			ErrInvalidGRPCCondition,
		)
	}

	return nil
}

// String returns a string representation of the gRPC condition
func (g GRPC) String() string {
	return fmt.Sprintf("gRPC Service: %s", g.Service)
}

// ToTree returns a tree representation of the gRPC condition
func (g GRPC) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("gRPC Service Condition")
	tree.AddChild(fmt.Sprintf("Service: %s", g.Service))
	return tree
}
