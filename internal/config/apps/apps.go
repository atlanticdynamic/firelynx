// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"google.golang.org/protobuf/types/known/durationpb"
)

// App represents an application definition
type App struct {
	ID     string
	Config AppConfig
}

// Apps is a collection of App definitions
type Apps []App

// AppConfig represents application-specific configuration
type AppConfig interface {
	Type() string
}

// StaticDataMergeMode represents strategies for merging static data
type StaticDataMergeMode string

// Constants for StaticDataMergeMode
const (
	StaticDataMergeModeUnspecified StaticDataMergeMode = ""
	StaticDataMergeModeLast        StaticDataMergeMode = "last"
	StaticDataMergeModeUnique      StaticDataMergeMode = "unique"
)

// StaticData represents configuration data passed to applications
type StaticData struct {
	Data      map[string]any
	MergeMode StaticDataMergeMode
}

// ScriptApp represents a script-based application
type ScriptApp struct {
	StaticData StaticData
	Evaluator  ScriptEvaluator
}

func (s ScriptApp) Type() string { return "script" }

// ScriptEvaluator represents a script execution engine
type ScriptEvaluator interface {
	Type() string
}

// RisorEvaluator executes Risor scripts
type RisorEvaluator struct {
	Code    string
	Timeout *durationpb.Duration
}

func (r RisorEvaluator) Type() string { return "risor" }

// StarlarkEvaluator executes Starlark scripts
type StarlarkEvaluator struct {
	Code    string
	Timeout *durationpb.Duration
}

func (s StarlarkEvaluator) Type() string { return "starlark" }

// ExtismEvaluator executes WebAssembly scripts
type ExtismEvaluator struct {
	Code       string
	Entrypoint string
}

func (e ExtismEvaluator) Type() string { return "extism" }

// CompositeScriptApp represents an application composed of multiple scripts
type CompositeScriptApp struct {
	ScriptAppIDs []string
	StaticData   StaticData
}

func (c CompositeScriptApp) Type() string { return "composite_script" }

// FindByID finds an app by its ID
func (a Apps) FindByID(id string) *App {
	for i, app := range a {
		if app.ID == id {
			return &a[i]
		}
	}
	return nil
}

// Validate checks that app configurations are valid
func (a Apps) Validate() error {
	if len(a) == 0 {
		return nil // Empty app list is valid
	}

	var errs []error

	// Create map of app IDs for reference validation
	appIDs := make(map[string]bool)
	for _, app := range a {
		if app.ID == "" {
			errs = append(errs, fmt.Errorf("%w: app ID", errz.ErrEmptyID))
			continue
		}

		if appIDs[app.ID] {
			errs = append(errs, fmt.Errorf("%w: app ID '%s'", errz.ErrDuplicateID, app.ID))
			continue
		}

		appIDs[app.ID] = true
	}

	// Validate each app's configuration
	for _, app := range a {
		if app.Config == nil {
			errs = append(errs, fmt.Errorf("%w: app '%s' has no configuration",
				errz.ErrMissingRequiredField, app.ID))
			continue
		}

		// Type-specific validation
		switch cfg := app.Config.(type) {
		case ScriptApp:
			// Validate evaluator exists
			if cfg.Evaluator == nil {
				errs = append(errs, fmt.Errorf("%w: script app '%s'",
					errz.ErrMissingEvaluator, app.ID))
				continue
			}

			// Validate evaluator by type
			switch eval := cfg.Evaluator.(type) {
			case RisorEvaluator:
				if eval.Code == "" {
					errs = append(errs, fmt.Errorf("%w: app '%s' Risor evaluator",
						errz.ErrEmptyCode, app.ID))
				}
			case StarlarkEvaluator:
				if eval.Code == "" {
					errs = append(errs, fmt.Errorf("%w: app '%s' Starlark evaluator",
						errz.ErrEmptyCode, app.ID))
				}
			case ExtismEvaluator:
				if eval.Code == "" {
					errs = append(errs, fmt.Errorf("%w: app '%s' Extism evaluator",
						errz.ErrEmptyCode, app.ID))
				}
				if eval.Entrypoint == "" {
					errs = append(errs, fmt.Errorf("%w: app '%s' Extism evaluator",
						errz.ErrEmptyEntrypoint, app.ID))
				}
			default:
				errs = append(errs, fmt.Errorf("%w: app '%s' has unknown evaluator type %T",
					errz.ErrInvalidEvaluator, app.ID, cfg.Evaluator))
			}

		case CompositeScriptApp:
			if len(cfg.ScriptAppIDs) == 0 {
				errs = append(errs, fmt.Errorf("%w: app '%s' composite script app requires at least one script app ID",
					errz.ErrMissingRequiredField, app.ID))
				continue
			}

			// Validate all referenced script apps exist
			for _, scriptAppID := range cfg.ScriptAppIDs {
				if scriptAppID == "" {
					errs = append(errs, fmt.Errorf("%w: in app '%s' composite script reference",
						errz.ErrEmptyID, app.ID))
					continue
				}

				if !appIDs[scriptAppID] {
					errs = append(errs, fmt.Errorf("%w: app '%s' references script app ID '%s'",
						errz.ErrAppNotFound, app.ID, scriptAppID))
				}
			}

		default:
			errs = append(errs, fmt.Errorf("%w: app '%s' has unknown config type %T",
				errz.ErrInvalidAppType, app.ID, app.Config))
		}
	}

	return errors.Join(errs...)
}

// ValidateRouteAppReferences ensures all routes reference valid apps
func (a Apps) ValidateRouteAppReferences(routes []struct{ AppID string }) error {
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
				errz.ErrAppNotFound, i, route.AppID))
		}
	}

	return errors.Join(errs...)
}

// AppsToInstances converts app definitions to running instances
func AppsToInstances(appDefs Apps) (map[string]apps.App, error) {
	// Validate app definitions first
	if err := appDefs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid app configuration: %w", err)
	}

	// Create instances map
	instances := make(map[string]apps.App)

	// Always register the built-in echo app
	instances["echo"] = echo.New("echo")

	// For now, all apps just create echo instances with their ID
	// This will be replaced with actual implementation later
	for _, app := range appDefs {
		instances[app.ID] = echo.New(app.ID)
	}

	return instances, nil
}
