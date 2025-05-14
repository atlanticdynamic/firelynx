package loader

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Loader-specific errors
var (
	ErrFailedToLoadConfig   = errors.New("failed to load config")
	ErrNoSourceProvided     = errors.New("no source provided to loader")
	ErrUnsupportedExtension = errors.New("unsupported file extension")
	ErrPostProcessConfig    = errors.New("failed to post-process config")
	ErrUnsupportedConfigVer = errz.ErrUnsupportedConfigVer
)

// FormatFileError creates an error with file path context
func FormatFileError(err error, path string) error {
	return fmt.Errorf("%w: %s", err, path)
}

// FormatValidationError wraps a base error with context about where the validation failed
func FormatValidationError(baseErr error, context string) error {
	return fmt.Errorf("%s: %w", context, baseErr)
}

// FormatListenerError wraps an error with listener context
func FormatListenerError(baseErr error, index int) error {
	return FormatValidationError(baseErr, fmt.Sprintf("listener at index %d", index))
}

// FormatEndpointError wraps an error with endpoint context
func FormatEndpointError(baseErr error, index int) error {
	return FormatValidationError(baseErr, fmt.Sprintf("endpoint at index %d", index))
}

// FormatRouteError wraps an error with route context
func FormatRouteError(baseErr error, routeIndex int, endpointId string) error {
	return FormatValidationError(
		baseErr,
		fmt.Sprintf("route %d in endpoint '%s'", routeIndex, endpointId),
	)
}

// FormatAppError wraps an error with app context
func FormatAppError(baseErr error, index int) error {
	return FormatValidationError(baseErr, fmt.Sprintf("app at index %d", index))
}
