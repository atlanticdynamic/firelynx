package conditions

import "errors"

// Error types are defined in types.go for now.
// This file exists to allow adding more specific errors in the future
// without modifying the core types file.

// Common condition-specific error types
var (
	ErrInvalidHTTPCondition = errors.New("invalid HTTP path condition")
	ErrInvalidGRPCCondition = errors.New("invalid gRPC service condition")
	ErrEmptyValue           = errors.New("empty condition value")
	ErrInvalidConditionType = errors.New("invalid condition type")
)
