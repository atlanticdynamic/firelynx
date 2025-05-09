package apps

import (
	"context"
	"net/http"
)

// App is the interface that all application handlers must implement
type App interface {
	// ID returns the unique identifier of the application
	ID() string

	// HandleHTTP processes HTTP requests
	HandleHTTP(context.Context, http.ResponseWriter, *http.Request, map[string]any) error
}

// Registry maintains a collection of applications
type Registry interface {
	// GetApp retrieves an application by ID
	GetApp(id string) (App, bool)

	// RegisterApp adds or replaces an application in the registry
	RegisterApp(app App) error

	// UnregisterApp removes an application from the registry
	UnregisterApp(id string) error
}
