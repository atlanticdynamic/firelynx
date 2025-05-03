package listeners

import (
	"errors"
	"fmt"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Type represents the protocol used by a listener
type Type string

// Constants for Type
const (
	TypeUnknown Type = ""
	TypeHTTP    Type = "http"
	TypeGRPC    Type = "grpc"
)

// Options represents protocol-specific options for listeners
type Options interface {
	Type() Type
	Validate() error
}

// HTTPOptions contains HTTP-specific listener configuration
type HTTPOptions struct {
	ReadTimeout  *durationpb.Duration
	WriteTimeout *durationpb.Duration
	DrainTimeout *durationpb.Duration
	IdleTimeout  *durationpb.Duration
}

func (h HTTPOptions) Type() Type { return TypeHTTP }

// Validate checks HTTPOptions for any configuration errors
func (h HTTPOptions) Validate() error {
	var errs []error
	
	// Validate timeouts (optional, but should be positive if set)
	if h.ReadTimeout != nil && h.ReadTimeout.AsDuration() <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP read timeout must be positive", 
			errz.ErrInvalidValue))
	}
	
	if h.WriteTimeout != nil && h.WriteTimeout.AsDuration() <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP write timeout must be positive", 
			errz.ErrInvalidValue))
	}
	
	if h.DrainTimeout != nil && h.DrainTimeout.AsDuration() <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP drain timeout must be positive", 
			errz.ErrInvalidValue))
	}
	
	if h.IdleTimeout != nil && h.IdleTimeout.AsDuration() <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP idle timeout must be positive", 
			errz.ErrInvalidValue))
	}
	
	return errors.Join(errs...)
}

// GRPCOptions contains gRPC-specific listener configuration
type GRPCOptions struct {
	MaxConnectionIdle    *durationpb.Duration
	MaxConnectionAge     *durationpb.Duration
	MaxConcurrentStreams int
}

func (g GRPCOptions) Type() Type { return TypeGRPC }

// Validate checks GRPCOptions for any configuration errors
func (g GRPCOptions) Validate() error {
	var errs []error
	
	// Validate connection timeouts (optional, but should be positive if set)
	if g.MaxConnectionIdle != nil && g.MaxConnectionIdle.AsDuration() <= 0 {
		errs = append(errs, fmt.Errorf("%w: gRPC max connection idle timeout must be positive", 
			errz.ErrInvalidValue))
	}
	
	if g.MaxConnectionAge != nil && g.MaxConnectionAge.AsDuration() <= 0 {
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

// Listeners is a collection of Listener objects
type Listeners []Listener

// Listener represents a network listener configuration
type Listener struct {
	ID      string
	Address string
	Options Options
}

// GetType returns the type of the listener
func (l *Listener) GetType() Type {
	if l.Options == nil {
		return TypeUnknown
	}

	return l.Options.Type()
}

// GetHTTPOptions safely extracts HTTPOptions from a Listener
func (l *Listener) GetHTTPOptions() (HTTPOptions, bool) {
	if l.Options == nil || l.Options.Type() != TypeHTTP {
		return HTTPOptions{}, false
	}

	httpOpts, ok := l.Options.(HTTPOptions)
	return httpOpts, ok
}

// GetReadTimeout extracts the read timeout with a fallback to default value
func (l *Listener) GetReadTimeout(defaultDuration time.Duration) time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok || httpOpts.ReadTimeout == nil {
		return defaultDuration
	}

	duration := httpOpts.ReadTimeout.AsDuration()
	if duration <= 0 {
		return defaultDuration
	}

	return duration
}

// GetWriteTimeout extracts the write timeout with a fallback to default value
func (l *Listener) GetWriteTimeout(defaultDuration time.Duration) time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok || httpOpts.WriteTimeout == nil {
		return defaultDuration
	}

	duration := httpOpts.WriteTimeout.AsDuration()
	if duration <= 0 {
		return defaultDuration
	}

	return duration
}

// GetDrainTimeout extracts the drain timeout with a fallback to default value
func (l *Listener) GetDrainTimeout(defaultDuration time.Duration) time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok || httpOpts.DrainTimeout == nil {
		return defaultDuration
	}

	duration := httpOpts.DrainTimeout.AsDuration()
	if duration <= 0 {
		return defaultDuration
	}

	return duration
}

// GetIdleTimeout extracts the idle timeout with a fallback to default value
func (l *Listener) GetIdleTimeout(defaultDuration time.Duration) time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok || httpOpts.IdleTimeout == nil {
		return defaultDuration
	}

	duration := httpOpts.IdleTimeout.AsDuration()
	if duration <= 0 {
		return defaultDuration
	}

	return duration
}

// Config represents the interface needed from a Config object to query endpoints
type Config interface {
	GetEndpoints() []any // avoid import cycles
}

// GetEndpoints returns the endpoints attached to a specific listener
// This requires passing in the config object
func (l *Listener) GetEndpoints(config any) []any {
	// This implementation needs to be updated by callers to convert any to the right type
	// This approach prevents import cycles
	return nil
}

// GetEndpointIDs returns the IDs of endpoints that are attached to this listener
func (l *Listener) GetEndpointIDs(config any) []string {
	// This is a placeholder method that should be implemented by the client
	// using a type assertion to avoid import cycles
	return nil
}
