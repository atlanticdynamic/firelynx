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
	ErrRouteTypeMismatch   = errors.New("route type mismatch with listener")
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

// TOML loader specific errors
var (
	// ErrLoader is the base error type for all loader errors
	ErrLoader = errors.New("config loader error")

	// ErrTomlLoader is the base error type for TOML loader errors
	ErrTomlLoader = errors.New("TOML loader error")

	// ErrNoSourceData is returned when no source data is provided to the loader
	ErrNoSourceData = errors.New("no source data provided")

	// ErrJsonConversion is returned when TOML to JSON conversion fails
	ErrJsonConversion = errors.New("failed to convert TOML to JSON")

	// ErrUnmarshalProto is returned when JSON to proto unmarshaling fails
	ErrUnmarshalProto = errors.New("failed to unmarshal proto")

	// ErrPostProcessConfig is returned when post-processing config fails
	ErrPostProcessConfig = errors.New("failed to post-process config")

	// ErrInvalidListenerFormat is returned when a listener has invalid format
	ErrInvalidListenerFormat = errors.New("invalid listener format")

	// ErrUnsupportedListenerType is returned when a listener type is unsupported
	ErrUnsupportedListenerType = errors.New("unsupported listener type")

	// ErrInvalidEndpointFormat is returned when an endpoint has invalid format
	ErrInvalidEndpointFormat = errors.New("invalid endpoint format")

	// ErrInvalidAppFormat is returned when an app has invalid format
	ErrInvalidAppFormat = errors.New("invalid app format")
)
