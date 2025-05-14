// Package staticdata provides types and utilities for handling static data
// that can be passed to apps and routes in firelynx.
package staticdata

import (
	"errors"
	"fmt"
)

var (
	// ErrStaticData is the base error type for staticdata package errors.
	ErrStaticData = errors.New("static data error")

	// ErrInvalidMergeMode indicates an invalid merge mode was specified.
	ErrInvalidMergeMode = fmt.Errorf("%w: invalid merge mode", ErrStaticData)

	// ErrInvalidData indicates that the provided data is invalid.
	ErrInvalidData = fmt.Errorf("%w: invalid data", ErrStaticData)
)

// NewInvalidMergeModeError returns a new error for an invalid merge mode with the provided value.
func NewInvalidMergeModeError(value any) error {
	return fmt.Errorf("%w: %v", ErrInvalidMergeMode, value)
}
