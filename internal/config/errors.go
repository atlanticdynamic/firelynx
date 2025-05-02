// Package config provides configuration management for the firelynx server.
package config

// Import errors from the centralized errz package
import (
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Re-export errors for backward compatibility and convenience
var (
	// Top-level error categories
	ErrFailedToLoadConfig     = errz.ErrFailedToLoadConfig
	ErrFailedToConvertConfig  = errz.ErrFailedToConvertConfig
	ErrFailedToValidateConfig = errz.ErrFailedToValidateConfig
	ErrUnsupportedConfigVer   = errz.ErrUnsupportedConfigVer

	// Validation specific errors
	ErrDuplicateID          = errz.ErrDuplicateID
	ErrEmptyID              = errz.ErrEmptyID
	ErrInvalidReference     = errz.ErrInvalidReference
	ErrMissingRequiredField = errz.ErrMissingRequiredField
	ErrRouteConflict        = errz.ErrRouteConflict

	// Type specific errors
	ErrInvalidListenerType = errz.ErrInvalidListenerType
	ErrInvalidRouteType    = errz.ErrInvalidRouteType
	ErrInvalidAppType      = errz.ErrInvalidAppType
	ErrInvalidEvaluator    = errz.ErrInvalidEvaluator

	// Reference specific errors
	ErrListenerNotFound = errz.ErrListenerNotFound
	ErrAppNotFound      = errz.ErrAppNotFound
	ErrEndpointNotFound = errz.ErrEndpointNotFound
	ErrRouteNotFound    = errz.ErrRouteNotFound
)
