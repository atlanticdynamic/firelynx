package apps

import "errors"

// Sentinel errors for app instantiation
var (
	// ErrInvalidConfigType is returned when a nil or wrong type config is passed to an instantiator
	ErrInvalidConfigType = errors.New("invalid config type")

	// ErrConfigConversionFailed is returned when domain config conversion to DTO fails
	ErrConfigConversionFailed = errors.New("config conversion failed")
)
