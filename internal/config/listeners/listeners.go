package listeners

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// Type represents the protocol used by a listener
type Type string

// Constants for Type
const (
	TypeHTTP Type = "http"
	TypeGRPC Type = "grpc"
)

// Options represents protocol-specific options for listeners
type Options interface {
	Type() Type
}

// HTTPOptions contains HTTP-specific listener configuration
type HTTPOptions struct {
	ReadTimeout  *durationpb.Duration
	WriteTimeout *durationpb.Duration
	DrainTimeout *durationpb.Duration
	IdleTimeout  *durationpb.Duration
}

func (h HTTPOptions) Type() Type { return TypeHTTP }

// GRPCOptions contains gRPC-specific listener configuration
type GRPCOptions struct {
	MaxConnectionIdle    *durationpb.Duration
	MaxConnectionAge     *durationpb.Duration
	MaxConcurrentStreams int
}

func (g GRPCOptions) Type() Type { return TypeGRPC }

// Listeners is a collection of Listener objects
type Listeners []Listener

// Listener represents a network listener configuration
type Listener struct {
	ID      string
	Address string
	Type    Type
	Options Options
}

// GetHTTPOptions safely extracts HTTPOptions from a Listener
func (l *Listener) GetHTTPOptions() (HTTPOptions, bool) {
	if l.Type != TypeHTTP || l.Options == nil {
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
	GetEndpoints() []interface{} // We'll use interface{} to avoid import cycles
}

// GetEndpoints returns the endpoints attached to a specific listener
// This requires passing in the config object
func (l *Listener) GetEndpoints(config interface{}) []interface{} {
	// This implementation needs to be updated by callers to convert interface{} to the right type
	// This approach prevents import cycles
	return nil
}

// GetEndpointIDs returns the IDs of endpoints that are attached to this listener
func (l *Listener) GetEndpointIDs(config interface{}) []string {
	// This is a placeholder method that should be implemented by the client
	// using a type assertion to avoid import cycles
	return nil
}
