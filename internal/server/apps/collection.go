package apps

import "fmt"

// Ensure AppCollection implements Registry
var _ Registry = (*AppCollection)(nil)

// AppCollection is an immutable collection of application instances
type AppCollection struct {
	// apps is a map of app ID to app instance
	apps map[string]App
}

// NewAppCollection creates a new AppCollection from a slice of App instances
func NewAppCollection(apps []App) (*AppCollection, error) {
	appMap := make(map[string]App, len(apps))

	// Index apps by their ID
	for _, app := range apps {
		id := app.String()
		if _, exists := appMap[id]; exists {
			return nil, fmt.Errorf("duplicate app ID: %s", id)
		}
		appMap[id] = app
	}

	return &AppCollection{apps: appMap}, nil
}

// GetApp retrieves an application instance by ID
func (c *AppCollection) GetApp(id string) (App, bool) {
	app, exists := c.apps[id]
	return app, exists
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
