package options

import (
	"errors"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Type represents the protocol used by a listener
type Type string

// Constants for Type
const (
	Unknown  Type = ""
	TypeHTTP Type = "http"
	TypeGRPC Type = "grpc"
)

// Options represents protocol-specific options for listeners
type Options interface {
	Type() Type
	Validate() error
	String() string
	ToTree() *fancy.ComponentTree
}

// Common option-specific error types
var (
	ErrInvalidHTTPOptions = errors.New("invalid HTTP listener options")
	ErrInvalidGRPCOptions = errors.New("invalid gRPC listener options")
)
