// Package listeners provides configuration types and utilities for network listeners
// in the firelynx server.
//
// This package defines the domain model for listener configurations, supporting multiple
// protocol types through the options sub-package. It handles validation, conversion
// between domain and protocol buffer representations, and provides helper methods for
// accessing protocol-specific configurations.
//
// The main types include:
// - Listener: Represents a single listener with an ID, address, and protocol-specific options
// - ListenerCollection: A slice of Listener objects with validation and conversion methods
//
// Relationship with Endpoints:
// To find endpoints for a specific listener, use the methods in the config package:
//   - config.GetEndpointsForListener(listenerID string)
//   - config.GetEndpointsByListenerID(listenerID string)
//   - config.GetEndpointIDsForListener(listenerID string)
//
// Thread Safety:
// The listener configuration objects are not thread-safe and should be protected when
// accessed concurrently. These objects are typically loaded during startup or configuration
// reload operations, which should be synchronized.
//
// Usage Example:
//
//	// Create an HTTP listener
//	httpListener := listeners.Listener{
//	    ID:      "http-main",
//	    Address: "0.0.0.0:8080",
//	    Type:    TypeHTTP,
//	    Options: options.HTTP{
//	        ReadTimeout:  time.Second * 30,
//	        WriteTimeout: time.Second * 30,
//	    },
//	}
//
//	// Create a collection
//	listenerCollection := listeners.ListenerCollection{httpListener}
//
//	// Validate the configuration
//	if err := listenerCollection.Validate(); err != nil {
//	    return err
//	}
package listeners

import (
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
)

// Type represents the listener type
type Type int32

// Listener types
const (
	TypeUnspecified Type = 0
	TypeHTTP        Type = 1
)

// ListenerCollection is a collection of Listener objects
type ListenerCollection []Listener

// Listener represents a network listener configuration
type Listener struct {
	ID      string `env_interpolation:"no"`
	Address string `env_interpolation:"yes"`
	Type    Type
	Options options.Options
}

// GetOptionsType returns the type of the listener options
func (l *Listener) GetOptionsType() options.Type {
	if l.Options == nil {
		return options.Unknown
	}

	return l.Options.Type()
}

// GetTypeString returns a string representation of the listener type
func (l *Listener) GetTypeString() string {
	switch l.Type {
	case TypeHTTP:
		return "HTTP"
	default:
		return "Unknown"
	}
}

// GetHTTPOptions extracts HTTPOptions from a Listener
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
