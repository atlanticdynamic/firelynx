package apps

import (
	"context"
	"fmt"
	"net/http"
)

// App is the interface that all application handlers must implement
type App interface {
	// ID returns the unique identifier of the application
	ID() string

	// HandleHTTP processes HTTP requests
	HandleHTTP(context.Context, http.ResponseWriter, *http.Request, map[string]any) error
}

// AppCollection is an immutable collection of application instances
type AppCollection struct {
	// apps is a map of app ID to app instance
	apps map[string]App
}

// GetApp retrieves an application instance by ID
func (c *AppCollection) GetApp(id string) (App, bool) {
	app, exists := c.apps[id]
	return app, exists
}

// NewAppCollection creates a new AppCollection from a slice of App instances
func NewAppCollection(apps []App) (*AppCollection, error) {
	appMap := make(map[string]App, len(apps))

	// Index apps by their ID
	for _, app := range apps {
		id := app.ID()
		if _, exists := appMap[id]; exists {
			return nil, fmt.Errorf("duplicate app ID: %s", id)
		}
		appMap[id] = app
	}

	return &AppCollection{apps: appMap}, nil
}

// Registry represents a read-only view of a collection of applications
type Registry interface {
	// GetApp retrieves an application by ID
	GetApp(id string) (App, bool)
}

// String returns a string representation of the app collection
func (c *AppCollection) String() string {
	if c == nil || len(c.apps) == 0 {
		return "AppCollection{empty}"
	}

	ids := make([]string, 0, len(c.apps))
	for id := range c.apps {
		ids = append(ids, id)
	}

	return fmt.Sprintf("AppCollection{apps: %v}", ids)
}

// Ensure AppCollection implements Registry
var _ Registry = (*AppCollection)(nil)
