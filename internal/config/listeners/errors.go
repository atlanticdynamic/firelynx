package listeners

import (
	"errors"
)

// Common listener-specific error types
var (
	ErrInvalidListenerType = errors.New("invalid listener type")
)
