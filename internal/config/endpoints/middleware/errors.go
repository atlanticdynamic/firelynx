package middleware

import "errors"

var (
	// ErrEmptyID indicates that a middleware ID is empty
	ErrEmptyID = errors.New("empty ID")

	// ErrDuplicateID indicates that a middleware ID is duplicated
	ErrDuplicateID = errors.New("duplicate ID")

	// ErrMissingMiddlewareConfig indicates that middleware config is missing
	ErrMissingMiddlewareConfig = errors.New("missing middleware config")

	// ErrMiddlewareNotFound indicates that a referenced middleware was not found
	ErrMiddlewareNotFound = errors.New("middleware not found")
)
