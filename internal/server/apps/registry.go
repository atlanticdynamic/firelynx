// Package apps provides interfaces and registry for application handlers
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

// SimpleRegistry is a basic implementation of the Registry interface
type SimpleRegistry struct {
	apps map[string]App
}

// NewSimpleRegistry creates a new SimpleRegistry
func NewSimpleRegistry() *SimpleRegistry {
	return &SimpleRegistry{
		apps: make(map[string]App),
	}
}

// GetApp retrieves an application by ID
func (r *SimpleRegistry) GetApp(id string) (App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

// RegisterApp adds or replaces an application in the registry
func (r *SimpleRegistry) RegisterApp(app App) error {
	r.apps[app.ID()] = app
	return nil
}

// UnregisterApp removes an application from the registry
func (r *SimpleRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}
