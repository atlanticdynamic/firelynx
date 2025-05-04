package listeners

import (
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
)

// Listeners is a collection of Listener objects
type Listeners []Listener

// Listener represents a network listener configuration
type Listener struct {
	ID      string
	Address string
	Options options.Options
}

// GetType returns the type of the listener
func (l *Listener) GetType() options.Type {
	if l.Options == nil {
		return options.Unknown
	}

	return l.Options.Type()
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

// GetHTTPOptions safely extracts HTTPOptions from a Listener
func (l *Listener) GetHTTPOptions() (options.HTTP, bool) {
	if l.Options == nil || l.Options.Type() != options.TypeHTTP {
		return options.HTTP{}, false
	}

	httpOpts, ok := l.Options.(options.HTTP)
	return httpOpts, ok
}

// GetReadTimeout extracts the read timeout with a fallback to default value
func (l *Listener) GetReadTimeout() time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok {
		return options.DefaultHTTPReadTimeout
	}

	return httpOpts.GetReadTimeout()
}

// GetWriteTimeout extracts the write timeout with a fallback to default value
func (l *Listener) GetWriteTimeout() time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok {
		return options.DefaultHTTPWriteTimeout
	}

	return httpOpts.GetWriteTimeout()
}

// GetDrainTimeout extracts the drain timeout with a fallback to default value
func (l *Listener) GetDrainTimeout() time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok {
		return options.DefaultHTTPDrainTimeout
	}

	return httpOpts.GetDrainTimeout()
}

// GetIdleTimeout extracts the idle timeout with a fallback to default value
func (l *Listener) GetIdleTimeout() time.Duration {
	httpOpts, ok := l.GetHTTPOptions()
	if !ok {
		return options.DefaultHTTPIdleTimeout
	}

	return httpOpts.GetIdleTimeout()
}

// GetGRPCOptions safely extracts GRPCOptions from a Listener
func (l *Listener) GetGRPCOptions() (options.GRPC, bool) {
	if l.Options == nil || l.Options.Type() != options.TypeGRPC {
		return options.GRPC{}, false
	}

	grpcOpts, ok := l.Options.(options.GRPC)
	return grpcOpts, ok
}
