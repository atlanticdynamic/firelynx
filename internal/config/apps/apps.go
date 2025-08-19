// Package apps provides types and functionality for application configuration
// in the firelynx server.
//
// This package defines the domain model for application configurations, including
// various app types (scripts, composite scripts) and their validation logic.
//
// The main types include:
// - App: Represents a single application configuration with ID and type-specific config
// - AppCollection: A struct containing a slice of App objects with validation and lookup methods
// - AppConfig: Interface implemented by all app type configs (scripts, composite, etc.)
//
// AppCollection provides centralized management of app definitions with duplicate ID detection,
// lookup by ID, and validation of cross-references between apps.
package apps

import (
	"errors"
	"fmt"
	"iter"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/validation"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// App represents an application definition
type App struct {
	ID     string
	Config AppConfig
}

// AppCollection is a collection of App definitions with centralized management
type AppCollection struct {
	Apps []App
}

// AppConfig represents application-specific configuration
type AppConfig interface {
	Type() string
	Validate() error
	ToProto() any
	String() string
	ToTree() *fancy.ComponentTree
}

// Validate validates a single app definition
func (a App) Validate() error {
	var errs []error

	// Validate ID
	if err := validation.ValidateID(a.ID, "app ID"); err != nil {
		errs = append(errs, err)
	}

	// Config validation
	if a.Config == nil {
		errs = append(errs, fmt.Errorf("%w: app '%s'", ErrMissingAppConfig, a.ID))
	} else {
		if err := a.Config.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("config for app '%s': %w", a.ID, err))
		}
	}

	return errors.Join(errs...)
}

// NewAppCollection creates a new AppCollection with the given apps
func NewAppCollection(apps ...App) *AppCollection {
	return &AppCollection{
		Apps: apps,
	}
}

// FindByID finds an app by its ID. Returns a copy to prevent external mutations.
func (ac *AppCollection) FindByID(id string) (App, bool) {
	for _, app := range ac.Apps {
		if app.ID == id {
			return app, true
		}
	}
	return App{}, false
}

// Len returns the number of apps in the collection
func (ac *AppCollection) Len() int {
	return len(ac.Apps)
}

// Get returns the app at the specified index
func (ac *AppCollection) Get(index int) App {
	return ac.Apps[index]
}

// All returns an iterator over all apps in the collection.
// This enables clean iteration: for app := range collection.All() { ... }
func (ac *AppCollection) All() iter.Seq[App] {
	return func(yield func(App) bool) {
		for _, app := range ac.Apps {
			if !yield(app) {
				return // Early termination support
			}
		}
	}
}

// Validate checks that app configurations are valid
func (ac *AppCollection) Validate() error {
	if len(ac.Apps) == 0 {
		return nil // Empty app list is valid
	}

	var errs []error

	// Create map of app IDs for reference validation
	appIDs := make(map[string]bool)

	// First pass: Validate IDs and check for duplicates
	for _, app := range ac.Apps {
		if err := validation.ValidateID(app.ID, "app ID"); err != nil {
			errs = append(errs, err)
			continue
		}

		if appIDs[app.ID] {
			errs = append(errs, fmt.Errorf("%w: app ID '%s'", ErrDuplicateID, app.ID))
			continue
		}

		appIDs[app.ID] = true
	}

	// Second pass: Validate each app individually and handle cross-references
	for i, app := range ac.Apps {
		// Skip apps with invalid IDs as those are already reported
		if err := validation.ValidateID(app.ID, "app ID"); err != nil {
			continue
		}

		// Validate the app itself
		if err := app.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("app at index %d: %w", i, err))
		}

		// Handle cross-references for composite apps
		// This can't be done in App.Validate() since it requires knowledge of all app IDs
		if comp, isComposite := app.Config.(*composite.CompositeScript); isComposite {
			// Validate all referenced script apps exist
			for _, scriptAppID := range comp.ScriptAppIDs {
				if err := validation.ValidateID(scriptAppID, "script app ID"); err != nil {
					errs = append(
						errs,
						fmt.Errorf("in app '%s' composite script reference: %w", app.ID, err),
					)
					continue
				}

				if !appIDs[scriptAppID] {
					errs = append(errs, fmt.Errorf("%w: app '%s' references script app ID '%s'",
						ErrAppNotFound, app.ID, scriptAppID))
				}
			}
		}
	}

	return errors.Join(errs...)
}

// ValidateRouteAppReferences ensures all routes reference valid apps
func (ac *AppCollection) ValidateRouteAppReferences(routes []struct{ AppID string }) error {
	// Build map of app IDs for quick lookup
	appIDs := make(map[string]bool)
	for _, app := range ac.Apps {
		appIDs[app.ID] = true
	}

	// Check each route's app ID
	var errs []error
	for i, route := range routes {
		if route.AppID == "" {
			continue // Empty app IDs are handled elsewhere
		}

		if !appIDs[route.AppID] {
			errs = append(errs, fmt.Errorf("%w: route at index %d references app ID '%s'",
				ErrAppNotFound, i, route.AppID))
		}
	}

	return errors.Join(errs...)
}
