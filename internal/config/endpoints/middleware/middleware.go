// Package middleware provides types and functionality for middleware configuration
// in the firelynx server.
//
// This package defines the domain model for middleware configurations, including
// various middleware types (console logger, etc.) and their validation logic.
// It serves as the boundary between configuration and runtime systems.
//
// The main types include:
// - Middleware: Represents a single middleware configuration with ID and type-specific config
// - MiddlewareCollection: A slice of Middleware objects with validation and lookup methods
// - MiddlewareConfig: Interface implemented by all middleware type configs
//
// Thread Safety:
// The types in this package are not inherently thread-safe and should be protected
// when used concurrently. Typically, these configuration objects are loaded during
// startup or config reload events, which should be synchronized.
//
// Usage Example:
//
//	// Create a middleware collection with a console logger
//	consoleLogger := &middleware.Middleware{
//	    ID: "request-logger",
//	    Config: logger.NewConsoleLogger(options),
//	}
//	middlewareCollection := middleware.MiddlewareCollection{consoleLogger}
//
//	// Validate the configuration
//	if err := middlewareCollection.Validate(); err != nil {
//	    return err
//	}
package middleware

import (
	"errors"
	"fmt"
	"sort"

	"github.com/atlanticdynamic/firelynx/internal/config/validation"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Middleware represents a middleware definition
type Middleware struct {
	ID     string
	Config MiddlewareConfig
}

// MiddlewareCollection is a collection of Middleware definitions
type MiddlewareCollection []Middleware

// MiddlewareConfig represents middleware-specific configuration
type MiddlewareConfig interface {
	Type() string
	Validate() error
	ToProto() any
	String() string
	ToTree() *fancy.ComponentTree
}

// Validate validates a single middleware definition
func (m Middleware) Validate() error {
	var errs []error

	// Validate ID
	if err := validation.ValidateID(m.ID, "middleware ID"); err != nil {
		errs = append(errs, err)
	}

	// Config validation
	if m.Config == nil {
		errs = append(errs, fmt.Errorf("%w: middleware '%s'", ErrMissingMiddlewareConfig, m.ID))
	} else {
		if err := m.Config.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("config for middleware '%s': %w", m.ID, err))
		}
	}

	return errors.Join(errs...)
}

// Merge merges multiple middleware collections into a single deduplicated collection.
// Middleware appearing later in the argument list take precedence over earlier ones
// when they share the same ID. The result is sorted alphabetically by ID.
//
// This enables ordering middleware using sortable names like:
// - "00-authentication"
// - "01-logger"
// - "02-rate-limiter"
func (mc MiddlewareCollection) Merge(others ...MiddlewareCollection) MiddlewareCollection {
	// Create a map to track middleware by ID for deduplication
	middlewareMap := make(map[string]Middleware)

	// First add this collection's middleware
	for _, mw := range mc {
		middlewareMap[mw.ID] = mw
	}

	// Then add middleware from other collections (later collections override earlier)
	for _, collection := range others {
		for _, mw := range collection {
			middlewareMap[mw.ID] = mw
		}
	}

	// Convert map back to slice
	result := make(MiddlewareCollection, 0, len(middlewareMap))
	for _, mw := range middlewareMap {
		result = append(result, mw)
	}

	// Sort alphabetically by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// FindByID finds a middleware by its ID
func (mc MiddlewareCollection) FindByID(id string) *Middleware {
	for i, middleware := range mc {
		if middleware.ID == id {
			return &mc[i]
		}
	}
	return nil
}

// Validate checks that middleware configurations are valid
func (mc MiddlewareCollection) Validate() error {
	if len(mc) == 0 {
		return nil // Empty middleware list is valid
	}

	var errs []error

	// Create map of middleware IDs for reference validation
	middlewareIDs := make(map[string]bool)

	// First pass: Validate IDs and check for duplicates
	for _, middleware := range mc {
		if err := validation.ValidateID(middleware.ID, "middleware ID"); err != nil {
			errs = append(errs, err)
			continue
		}

		if middlewareIDs[middleware.ID] {
			errs = append(errs, fmt.Errorf("%w: middleware ID '%s'", ErrDuplicateID, middleware.ID))
			continue
		}

		middlewareIDs[middleware.ID] = true
	}

	// Second pass: Validate each middleware individually
	for i, middleware := range mc {
		// Skip middlewares with invalid IDs as those are already reported
		if err := validation.ValidateID(middleware.ID, "middleware ID"); err != nil {
			continue
		}

		// Validate the middleware itself
		if err := middleware.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("middleware at index %d: %w", i, err))
		}
	}

	return errors.Join(errs...)
}
