// Package errz provides shared error definitions for the config package and its subpackages.
package errz

import "errors"

// Top-level error categories
var (
	ErrFailedToLoadConfig     = errors.New("failed to load config")
	ErrFailedToConvertConfig  = errors.New("failed to convert config from proto")
	ErrFailedToValidateConfig = errors.New("failed to validate config")
	ErrUnsupportedConfigVer   = errors.New("unsupported config version")
)

// Validation specific errors
var (
	ErrDuplicateID          = errors.New("duplicate ID")
	ErrEmptyID              = errors.New("empty ID")
	ErrInvalidReference     = errors.New("invalid reference")
	ErrInvalidValue         = errors.New("invalid value")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrRouteConflict        = errors.New("route conflict")
)

// Type specific errors
var (
	ErrInvalidListenerType = errors.New("invalid listener type")
	ErrInvalidRouteType    = errors.New("invalid route type")
	ErrInvalidAppType      = errors.New("invalid app type")
	ErrInvalidEvaluator    = errors.New("invalid evaluator")
)

// Reference specific errors
var (
	ErrListenerNotFound = errors.New("listener not found")
	ErrAppNotFound      = errors.New("app not found")
	ErrEndpointNotFound = errors.New("endpoint not found")
	ErrRouteNotFound    = errors.New("route not found")
)

// Script specific errors
var (
	ErrMissingEvaluator = errors.New("missing evaluator")
	ErrMissingAppConfig = errors.New("missing app configuration")
	ErrEmptyCode        = errors.New("empty code")
	ErrEmptyEntrypoint  = errors.New("empty entrypoint")
)
