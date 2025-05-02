package listeners

import (
	"errors"
)

// Common listener-specific error types
var (
	ErrInvalidHTTPOptions = errors.New("invalid HTTP listener options")
	ErrInvalidGRPCOptions = errors.New("invalid gRPC listener options")
)
