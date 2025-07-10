package middleware

import "errors"

var (
	// ErrDuplicateID indicates that a middleware ID is duplicated
	ErrDuplicateID = errors.New("duplicate ID")

	// ErrMissingMiddlewareConfig indicates that middleware config is missing
	ErrMissingMiddlewareConfig = errors.New("missing middleware config")

	// ErrMiddlewareNotFound indicates that a referenced middleware was not found
	ErrMiddlewareNotFound = errors.New("middleware not found")
)
