package loader

import (
	"errors"
	"fmt"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
)

// TestFormatFileError tests the FormatFileError function
func TestFormatFileError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("base error")

	// Format the error with a file path
	formattedErr := FormatFileError(baseErr, "/path/to/file.txt")

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "/path/to/file.txt")
	assert.Contains(t, formattedErr.Error(), "base error")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestFormatValidationError tests the FormatValidationError function
func TestFormatValidationError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("validation failed")

	// Format the error with context
	formattedErr := FormatValidationError(baseErr, "config validation")

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "config validation")
	assert.Contains(t, formattedErr.Error(), "validation failed")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestFormatListenerError tests the FormatListenerError function
func TestFormatListenerError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("listener error")

	// Format the error with a listener index
	formattedErr := FormatListenerError(baseErr, 2)

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "listener at index 2")
	assert.Contains(t, formattedErr.Error(), "listener error")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestFormatEndpointError tests the FormatEndpointError function
func TestFormatEndpointError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("endpoint error")

	// Format the error with an endpoint index
	formattedErr := FormatEndpointError(baseErr, 3)

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "endpoint at index 3")
	assert.Contains(t, formattedErr.Error(), "endpoint error")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestFormatRouteError tests the FormatRouteError function
func TestFormatRouteError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("route error")

	// Format the error with a route index and endpoint ID
	formattedErr := FormatRouteError(baseErr, 1, "test-endpoint")

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "route 1 in endpoint 'test-endpoint'")
	assert.Contains(t, formattedErr.Error(), "route error")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestFormatAppError tests the FormatAppError function
func TestFormatAppError(t *testing.T) {
	// Create a test error
	baseErr := errors.New("app error")

	// Format the error with an app index
	formattedErr := FormatAppError(baseErr, 4)

	// Check the error message
	assert.Contains(t, formattedErr.Error(), "app at index 4")
	assert.Contains(t, formattedErr.Error(), "app error")
	assert.ErrorIs(t, formattedErr, baseErr)
}

// TestNestedErrorFormatting tests the composition of multiple error formatting functions
func TestNestedErrorFormatting(t *testing.T) {
	// Create a base error
	baseErr := errors.New("field is required")

	// Apply multiple layers of formatting
	appErr := FormatAppError(baseErr, 2)
	listenerErr := FormatListenerError(appErr, 1)
	fileErr := FormatFileError(listenerErr, "config.toml")

	// Check that all layers are present in the message
	assert.Contains(t, fileErr.Error(), "app at index 2")
	assert.Contains(t, fileErr.Error(), "listener at index 1")
	assert.Contains(t, fileErr.Error(), "config.toml")
	assert.Contains(t, fileErr.Error(), "field is required")

	// Check that we can unwrap to the base error
	assert.ErrorIs(t, fileErr, baseErr)
}

// TestErrorJoining tests how errors can be joined together
func TestErrorJoining(t *testing.T) {
	baseErr1 := fmt.Errorf("error 1: %w", errz.ErrEmptyID)
	baseErr2 := fmt.Errorf("error 2: %w", errz.ErrMissingRequiredField)

	errList := []error{
		FormatListenerError(baseErr1, 0),
		FormatEndpointError(baseErr2, 1),
	}

	joinedErr := errors.Join(errList...)

	// Test the joined error contains both messages
	errStr := joinedErr.Error()
	assert.Contains(t, errStr, "listener at index 0")
	assert.Contains(t, errStr, "endpoint at index 1")
	assert.Contains(t, errStr, "error 1")
	assert.Contains(t, errStr, "error 2")

	// Test we can still use errors.Is with the wrapped errors
	assert.ErrorIs(t, joinedErr, errz.ErrEmptyID)
	assert.ErrorIs(t, joinedErr, errz.ErrMissingRequiredField)
}
