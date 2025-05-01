package config

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// ListenerType represents the protocol used by a listener
type ListenerType string

// Constants for ListenerType
const (
	ListenerTypeHTTP ListenerType = "http"
	ListenerTypeGRPC ListenerType = "grpc"
)

// ListenerOptions represents protocol-specific options for listeners
type ListenerOptions interface {
	Type() ListenerType
}

// HTTPListenerOptions contains HTTP-specific listener configuration
type HTTPListenerOptions struct {
	ReadTimeout  *durationpb.Duration
	WriteTimeout *durationpb.Duration
	DrainTimeout *durationpb.Duration
	IdleTimeout  *durationpb.Duration
}

func (h HTTPListenerOptions) Type() ListenerType { return ListenerTypeHTTP }

// GRPCListenerOptions contains gRPC-specific listener configuration
type GRPCListenerOptions struct {
	MaxConnectionIdle    *durationpb.Duration
	MaxConnectionAge     *durationpb.Duration
	MaxConcurrentStreams int
}

func (g GRPCListenerOptions) Type() ListenerType { return ListenerTypeGRPC }

// Listener represents a network listener configuration
type Listener struct {
	ID      string
	Address string
	Type    ListenerType
	Options ListenerOptions
}

// GetHTTPOptions safely extracts HTTPListenerOptions from a Listener
func (l *Listener) GetHTTPOptions() (HTTPListenerOptions, bool) {
	if l.Type != ListenerTypeHTTP || l.Options == nil {
		return HTTPListenerOptions{}, false
	}

	httpOpts, ok := l.Options.(HTTPListenerOptions)
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
