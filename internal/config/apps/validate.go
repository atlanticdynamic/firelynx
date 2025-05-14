package apps

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
)

// We've moved the App.Validate() method to apps.go
// The method below is now deprecated and should be removed

// validateAppConfig calls the Validate method on the app config
// All app config types now implement the Validate method directly

// ValidationContext facilitates cross-reference validation between apps and other components.
// It maintains a registry of known application IDs to verify that referenced apps exist
// when validating relationships such as:
//   - Routes referencing apps
//   - Composite apps referencing other script apps
//   - External systems referencing apps
//
// This context-based approach allows validation to occur without creating circular dependencies
// between packages, as the calling code provides the necessary context (list of app IDs) instead
// of the validation code needing to query other parts of the system directly.
type ValidationContext struct {
	AppIDs map[string]bool // Map of valid app IDs for existence checking
}

// NewValidationContext creates a new validation context with a copy of the provided app IDs.
// It always includes the built-in "echo" app in the list of valid app IDs regardless of input.
// The copied map prevents external modification of the context after creation.
func NewValidationContext(appIDs map[string]bool) *ValidationContext {
	// Make a copy to prevent external modification
	ids := make(map[string]bool, len(appIDs))
	for k, v := range appIDs {
		ids[k] = v
	}

	// Always include the built-in echo app
	ids["echo"] = true

	return &ValidationContext{
		AppIDs: ids,
	}
}

// ValidateAppReference checks if an app ID exists in the validation context.
// It returns an error if the app ID is empty or if it doesn't exist in the list of known apps.
// This method is used to validate a single app reference when validating external components
// that refer to apps.
func (vc *ValidationContext) ValidateAppReference(appID string) error {
	if appID == "" {
		return fmt.Errorf("%w: app ID", ErrEmptyID)
	}

	if !vc.AppIDs[appID] {
		return fmt.Errorf("%w: '%s'", ErrAppNotFound, appID)
	}

	return nil
}

// ValidateRouteReferences checks if all route app references point to valid apps.
// It accepts a list of route references (simplified to just contain the AppID field) and
// validates that each non-empty AppID exists in the ValidationContext's list of known apps.
// This is used during config validation to ensure routes don't reference non-existent apps.
// Empty app IDs are skipped as they are handled by other validation logic.
func (vc *ValidationContext) ValidateRouteReferences(routes []struct{ AppID string }) error {
	if len(routes) == 0 {
		return nil // No routes to validate
	}

	var errs []error
	for i, route := range routes {
		if route.AppID == "" {
			continue // Empty app IDs are handled by other validation
		}

		if !vc.AppIDs[route.AppID] {
			errs = append(errs, fmt.Errorf(
				"%w: route at index %d references app ID '%s'",
				ErrAppNotFound, i, route.AppID))
		}
	}

	return errors.Join(errs...)
}

// ValidateCompositeAppReferences validates that all scripts referenced by a composite app exist.
// CompositeScript types can reference other script apps by ID. This validation ensures that
// all those referenced script apps actually exist in the system. If the app is not a composite app,
// this validation succeeds immediately.
// This validation is needed because composite apps depend on other apps, creating a dependency graph
// that must be validated for consistency.
func (vc *ValidationContext) ValidateCompositeAppReferences(app App) error {
	composite, ok := app.Config.(*composite.CompositeScript)
	if !ok {
		return nil // Not a composite app, nothing to validate
	}

	var errs []error
	for _, scriptAppID := range composite.ScriptAppIDs {
		if scriptAppID == "" {
			errs = append(errs, fmt.Errorf(
				"%w: composite app '%s' has script app ID reference",
				ErrEmptyID, app.ID))
			continue
		}

		if !vc.AppIDs[scriptAppID] {
			errs = append(errs, fmt.Errorf(
				"%w: composite app '%s' references script app ID '%s'",
				ErrAppNotFound, app.ID, scriptAppID))
		}
	}

	return errors.Join(errs...)
}
