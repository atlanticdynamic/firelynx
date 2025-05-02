package apps

import (
	"errors"
	"fmt"

	configerrz "github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// ValidationContext provides access to necessary context for app validation
type ValidationContext struct {
	AppIDs map[string]bool // Available app IDs
}

// NewValidationContext creates a new validation context
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

// ValidateAppReference checks if an app ID exists in the context
func (vc *ValidationContext) ValidateAppReference(appID string) error {
	if appID == "" {
		return fmt.Errorf("%w: app ID", configerrz.ErrEmptyID)
	}

	if !vc.AppIDs[appID] {
		return fmt.Errorf("%w: '%s'", configerrz.ErrAppNotFound, appID)
	}

	return nil
}

// ValidateRouteReferences checks if all route app references are valid
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
				configerrz.ErrAppNotFound, i, route.AppID))
		}
	}

	return errors.Join(errs...)
}

// ValidateCompositeAppReferences checks if all composite app references are valid
func (vc *ValidationContext) ValidateCompositeAppReferences(app App) error {
	composite, ok := app.Config.(CompositeScriptApp)
	if !ok {
		return nil // Not a composite app, nothing to validate
	}

	var errs []error
	for _, scriptAppID := range composite.ScriptAppIDs {
		if scriptAppID == "" {
			errs = append(errs, fmt.Errorf(
				"%w: composite app '%s' has script app ID reference",
				configerrz.ErrEmptyID, app.ID))
			continue
		}

		if !vc.AppIDs[scriptAppID] {
			errs = append(errs, fmt.Errorf(
				"%w: composite app '%s' references script app ID '%s'",
				configerrz.ErrAppNotFound, app.ID, scriptAppID))
		}
	}

	return errors.Join(errs...)
}
