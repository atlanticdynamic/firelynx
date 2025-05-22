// Package apps provides interfaces and implementations for firelynx applications.
//
// This file defines the source of truth for all available app implementations
// in the codebase. It provides an immutable map of app types to their creator
// functions, which ensures that all available app types are registered in a
// single location.
package apps

import (
	"maps"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
)

// AppCreator creates a server app instance with the given ID and configuration.
// The config parameter type depends on the specific app implementation.
type AppCreator func(id string, config any) (App, error)

// AvailableAppImplementations is an immutable map of app types to creation functions.
// This serves as the source of truth for what app types are implemented in the codebase.
// Any new app implementation should be added to this map.
var AvailableAppImplementations = map[string]AppCreator{
	"echo": func(id string, _ any) (App, error) {
		return echo.New(id), nil
	},
	// Future implementations would be added here, such as:
	// "script": createScriptApp,
	// "composite_script": createCompositeScriptApp,
}

// GetAllAppIDs returns a list of all app IDs that have implementations. These apps may not be
// enabled or included in the config, but are available for use. This is used for validation
// in the config layer, when the config lists several app instances, we need to validate the
// app ID against this list.
func GetAllAppIDs() []string {
	types := make([]string, 0, len(AvailableAppImplementations))
	for k := range maps.Keys(AvailableAppImplementations) {
		types = append(types, k)
	}
	return types
}

// GetAllAppImplementations returns a list of all app implementations.
func GetAllAppImplementations() map[string]AppCreator {
	return maps.Clone(AvailableAppImplementations)
}
