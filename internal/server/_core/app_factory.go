// Package core provides adapters between domain config and runtime components.
// This is the ONLY package that should import from internal/config.
package core

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// CreateAppInstances converts domain app configurations to runtime app instances.
// This function implements the adapter pattern, bridging between domain config and runtime.
func CreateAppInstances(appDefs apps.AppCollection) (map[string]serverApps.App, error) {
	// Validate app definitions first
	if err := appDefs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid app configuration: %w", err)
	}

	// Create instances map
	instances := make(map[string]serverApps.App)

	// Process each app definition
	for _, appDef := range appDefs {
		// Skip composite apps for now as they may reference other apps
		if appDef.Config.Type() == "composite_script" {
			continue
		}

		// Create app instance based on type
		creator, exists := serverApps.AvailableAppImplementations[appDef.Config.Type()]
		if !exists {
			return nil, fmt.Errorf("unknown app type: %s", appDef.Config.Type())
		}

		app, err := creator(appDef.ID, appDef.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create app %s: %w", appDef.ID, err)
		}

		// Store the instance
		instances[appDef.ID] = app
	}

	// Second pass: create composite app instances (if any)
	// Implementation would go here for composite apps

	return instances, nil
}
