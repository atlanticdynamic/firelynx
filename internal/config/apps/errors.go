// Package apps provides types and functionality for application configuration.
package apps

// Import errors from the centralized errz package
import (
	"errors"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Re-export errors for backward compatibility and convenience
var (
	// Validation error types
	ErrEmptyID              = errz.ErrEmptyID
	ErrDuplicateID          = errz.ErrDuplicateID
	ErrMissingRequiredField = errz.ErrMissingRequiredField
	ErrInvalidAppType       = errz.ErrInvalidAppType
	ErrAppNotFound          = errz.ErrAppNotFound
	ErrInvalidEvaluator     = errz.ErrInvalidEvaluator
	ErrMissingEvaluator     = errz.ErrMissingEvaluator
	ErrEmptyCode            = errz.ErrEmptyCode
	ErrEmptyEntrypoint      = errz.ErrEmptyEntrypoint
	ErrMissingAppConfig     = errz.ErrMissingAppConfig

	// Proto conversion errors
	ErrAppDefinitionNil  = errors.New("app definition is nil")
	ErrNoConfigSpecified = errors.New("no config specified")
	ErrUnknownConfigType = errors.New("unknown config type")
	ErrTypeMismatch      = errors.New("declared type does not match actual config type")
	ErrMissingConfig     = errors.New("declared type has no corresponding config")
)
