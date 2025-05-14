// Package apps provides interfaces and implementations for firelynx applications.
//
// This file defines the source of truth for all available app implementations
// in the codebase. It provides an immutable map of app types to their creator
// functions, which ensures that all available app types are registered in a
// single location.
package apps

import (
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
)

// AppCreator creates a server app instance with the given ID and configuration.
// The config parameter type depends on the specific app implementation.
type AppCreator func(id string, config any) (App, error)

// AvailableAppImplementations is an immutable map of app types to creation functions.
// This serves as the source of truth for what app types are implemented in the codebase.
// Any new app implementation should be added to this map.
var AvailableAppImplementations = map[string]AppCreator{
	"echo": createEchoApp,
	// Future implementations would be added here, such as:
	// "script": createScriptApp,
	// "composite_script": createCompositeScriptApp,
}

// createEchoApp creates a new Echo app instance.
// The config parameter is currently ignored as Echo apps don't require configuration.
func createEchoApp(id string, _ any) (App, error) {
	return echo.New(id), nil
}
