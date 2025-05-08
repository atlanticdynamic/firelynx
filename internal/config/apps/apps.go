// Package apps provides types and functionality for application configuration
// in the firelynx server.
//
// This package defines the domain model for application configurations, including
// various app types (scripts, composite scripts) and their validation logic.
// It serves as the boundary between configuration and runtime systems.
//
// The main types include:
// - App: Represents a single application configuration with ID and type-specific config
// - AppCollection: A slice of App objects with validation and lookup methods
// - AppConfig: Interface implemented by all app type configs (scripts, composite, etc.)
//
// Thread Safety:
// The types in this package are not inherently thread-safe and should be protected
// when used concurrently. Typically, these configuration objects are loaded during
// startup or config reload events, which should be properly synchronized.
//
// Usage Example:
//
//	// Create an app collection with a script app
//	scriptApp := &apps.App{
//	    ID: "my-script",
//	    Config: scripts.NewAppScript(staticData, evaluator),
//	}
//	appCollection := apps.AppCollection{scriptApp}
//
//	// Validate the configuration
//	if err := appCollection.Validate(); err != nil {
//	    return err
//	}
//
//	// Convert to runtime instances
//	instances, err := apps.AppsToInstances(appCollection)
package apps

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
)

// App represents an application definition
type App struct {
	ID     string
	Config AppConfig
}

// AppCollection is a collection of App definitions
type AppCollection []App

// AppConfig represents application-specific configuration
type AppConfig interface {
	Type() string
	Validate() error
}

// Validate validates a single app definition
func (a App) Validate() error {
	var errs []error

	// ID is required
	if a.ID == "" {
		errs = append(errs, ErrEmptyID)
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

// FindByID finds an app by its ID
func (a AppCollection) FindByID(id string) *App {
	for i, app := range a {
		if app.ID == id {
			return &a[i]
		}
	}
	return nil
}

// Validate checks that app configurations are valid
func (a AppCollection) Validate() error {
	if len(a) == 0 {
		return nil // Empty app list is valid
	}

	var errs []error

	// Create map of app IDs for reference validation
	appIDs := make(map[string]bool)

	// First pass: Validate IDs and check for duplicates
	for _, app := range a {
		if app.ID == "" {
			errs = append(errs, fmt.Errorf("%w: app ID", ErrEmptyID))
			continue
		}

		if appIDs[app.ID] {
			errs = append(errs, fmt.Errorf("%w: app ID '%s'", ErrDuplicateID, app.ID))
			continue
		}

		appIDs[app.ID] = true
	}

	// Second pass: Validate each app individually and handle cross-references
	for i, app := range a {
		// Skip apps with empty IDs as those are already reported
		if app.ID == "" {
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
				if scriptAppID == "" {
					errs = append(errs, fmt.Errorf("%w: in app '%s' composite script reference",
						ErrEmptyID, app.ID))
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
func (a AppCollection) ValidateRouteAppReferences(routes []struct{ AppID string }) error {
	// Build map of app IDs for quick lookup
	appIDs := make(map[string]bool)
	for _, app := range a {
		appIDs[app.ID] = true
	}
	// Always include the built-in echo app
	appIDs["echo"] = true

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

// AppsToInstances converts app definitions to running instances
func AppsToInstances(appDefs AppCollection) (map[string]serverApps.App, error) {
	// Validate app definitions first
	if err := appDefs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid app configuration: %w", err)
	}

	// Create instances map
	instances := make(map[string]serverApps.App)

	// Always register the built-in echo app
	instances["echo"] = echo.New("echo")

	// First pass: create script app instances
	for _, appDef := range appDefs {
		// Skip composite apps for now as they may reference other apps
		if appDef.Config.Type() == "composite_script" {
			continue
		}

		// Create app instance based on type
		var instance serverApps.App
		switch appDef.Config.Type() {
		case "script":
			// For now, script apps are also implemented as echo apps
			// In a future implementation, these would use actual script evaluators
			instance = echo.New(appDef.ID)
		default:
			// Unknown app types default to echo for now
			instance = echo.New(appDef.ID)
		}

		// Store the instance
		instances[appDef.ID] = instance
	}

	// Second pass: create composite app instances
	// Now all referenced script apps should exist
	for _, appDef := range appDefs {
		// Only process composite apps
		if appDef.Config.Type() != "composite_script" {
			continue
		}

		// Cast to composite script config
		compositeConfig, ok := appDef.Config.(*composite.CompositeScript)
		if !ok {
			return nil, fmt.Errorf("invalid composite script configuration for app '%s'", appDef.ID)
		}

		// Verify that all referenced script apps exist
		for _, scriptAppID := range compositeConfig.ScriptAppIDs {
			if _, exists := instances[scriptAppID]; !exists {
				return nil, fmt.Errorf("composite app '%s' references non-existent script app '%s'",
					appDef.ID, scriptAppID)
			}
		}

		// For now, composite apps are also implemented as echo apps
		// In a future implementation, these would chain multiple script evaluators
		instances[appDef.ID] = echo.New(appDef.ID)
	}

	return instances, nil
}
