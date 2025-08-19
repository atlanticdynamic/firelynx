package config

import (
	"fmt"
	"maps"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// expandAppsForRoutes assigns app instances to routes with merged static data.
// Each route gets its own app instance with route-specific static data merged in.
// Expanded apps get unique IDs to avoid conflicts in the server registry.
func expandAppsForRoutes(appCollection *apps.AppCollection, endpoints endpoints.EndpointCollection) {
	if appCollection == nil || appCollection.Len() == 0 || len(endpoints) == 0 {
		return
	}

	// Process each endpoint
	for endpointIndex := range endpoints {
		endpoint := &endpoints[endpointIndex]

		// Process each route in the endpoint
		for routeIndex := range endpoint.Routes {
			route := &endpoint.Routes[routeIndex]

			if route.AppID == "" {
				continue
			}

			// Find the original app using the collection's FindByID method
			originalApp, exists := appCollection.FindByID(route.AppID)
			if !exists {
				continue // Skip invalid app references, validation will catch this
			}

			// Create unique app ID for this route's app instance
			expandedAppID := fmt.Sprintf("%s#%d:%d", route.AppID, endpointIndex, routeIndex)

			// Clone the app and merge route-specific static data
			routeApp := cloneAppWithMergedStaticData(originalApp, route.StaticData, expandedAppID)
			route.App = &routeApp

			// Keep route.AppID as original for validation, but store expanded app separately
		}
	}
}

// cloneAppWithMergedStaticData creates a copy of an app with route static data merged in
func cloneAppWithMergedStaticData(
	originalApp apps.App,
	routeStaticData map[string]any,
	newAppID string,
) apps.App {
	// Create a new app with the unique expanded ID
	clonedApp := apps.App{
		ID: newAppID,
	}

	// Clone the config based on app type
	switch config := originalApp.Config.(type) {
	case *scripts.AppScript:
		clonedConfig := &scripts.AppScript{
			Evaluator: config.Evaluator, // Evaluator can be shared (immutable)
		}

		// Merge static data
		clonedConfig.StaticData = mergeStaticDataForApp(config.StaticData, routeStaticData)
		clonedApp.Config = clonedConfig

	default:
		// For other app types (echo, composite), just copy the config
		// They don't support static data merging yet
		clonedApp.Config = originalApp.Config
	}

	return clonedApp
}

// mergeStaticDataForApp merges route-level static data into app-level static data
func mergeStaticDataForApp(
	appStaticData *staticdata.StaticData,
	routeStaticData map[string]any,
) *staticdata.StaticData {
	if len(routeStaticData) == 0 {
		return appStaticData
	}

	// Start with app-level data (if any)
	mergedData := make(map[string]any)
	if appStaticData != nil && appStaticData.Data != nil {
		maps.Copy(mergedData, appStaticData.Data)
	}

	// Route data overwrites app data for the same keys
	maps.Copy(mergedData, routeStaticData)

	// Determine merge mode
	mergeMode := staticdata.StaticDataMergeModeUnique // Default
	if appStaticData != nil {
		mergeMode = appStaticData.MergeMode
	}

	return &staticdata.StaticData{
		Data:      mergedData,
		MergeMode: mergeMode,
	}
}
