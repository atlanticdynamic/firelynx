package options

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// GRPC default values
const (
	DefaultGRPCMaxConnectionIdle = 10 * time.Minute
	DefaultGRPCMaxConnectionAge  = 30 * time.Minute
)

// GRPC contains gRPC-specific listener configuration
type GRPC struct {
	MaxConnectionIdle    time.Duration
	MaxConnectionAge     time.Duration
	MaxConcurrentStreams int
}

// NewGRPC creates a new GRPC with default values
func NewGRPC() GRPC {
	return GRPC{
		MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
		MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
		MaxConcurrentStreams: 0, // 0 means use gRPC's default
	}
}

// Type returns the listener type this options is for
func (g GRPC) Type() Type { return TypeGRPC }

// Validate checks GRPC for any configuration errors
func (g GRPC) Validate() error {
	var errs []error

	// Validate connection timeouts (should be positive)
	if g.MaxConnectionIdle <= 0 {
		errs = append(errs, fmt.Errorf("%w: gRPC max connection idle timeout must be positive",
			errz.ErrInvalidValue))
	}

	if g.MaxConnectionAge <= 0 {
		errs = append(errs, fmt.Errorf("%w: gRPC max connection age must be positive",
			errz.ErrInvalidValue))
	}

	// Validate MaxConcurrentStreams if set
	if g.MaxConcurrentStreams < 0 {
		errs = append(errs, fmt.Errorf("%w: gRPC max concurrent streams cannot be negative",
			errz.ErrInvalidValue))
	}

	return errors.Join(errs...)
}

// GetMaxConnectionIdle returns the max connection idle timeout
func (g GRPC) GetMaxConnectionIdle() time.Duration {
	if g.MaxConnectionIdle <= 0 {
		return DefaultGRPCMaxConnectionIdle
	}
	return g.MaxConnectionIdle
}

// GetMaxConnectionAge returns the max connection age
func (g GRPC) GetMaxConnectionAge() time.Duration {
	if g.MaxConnectionAge <= 0 {
		return DefaultGRPCMaxConnectionAge
	}
	return g.MaxConnectionAge
}

// String returns a string representation of GRPC options
func (g GRPC) String() string {
	var b strings.Builder

	if g.MaxConnectionIdle > 0 {
		fmt.Fprintf(&b, "MaxConnectionIdle: %v, ", g.MaxConnectionIdle)
	}
	if g.MaxConnectionAge > 0 {
		fmt.Fprintf(&b, "MaxConnectionAge: %v, ", g.MaxConnectionAge)
	}
	if g.MaxConcurrentStreams > 0 {
		fmt.Fprintf(&b, "MaxConcurrentStreams: %d, ", g.MaxConcurrentStreams)
	}

	str := b.String()
	if len(str) > 2 {
		// Remove trailing comma and space
		return str[:len(str)-2]
	}
	return str
}

// ToTree returns a tree visualization of GRPC options
func (g GRPC) ToTree() *fancy.ComponentTree {
	// Create a base tree for the GRPC options
	tree := fancy.NewComponentTree("GRPC Options")

	if g.MaxConnectionIdle > 0 {
		tree.AddChild(fmt.Sprintf("MaxConnectionIdle: %v", g.MaxConnectionIdle))
	}
	if g.MaxConnectionAge > 0 {
		tree.AddChild(fmt.Sprintf("MaxConnectionAge: %v", g.MaxConnectionAge))
	}
	if g.MaxConcurrentStreams > 0 {
		tree.AddChild(fmt.Sprintf("MaxConcurrentStreams: %d", g.MaxConcurrentStreams))
	}

	return tree
}
