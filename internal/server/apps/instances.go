package apps

import "fmt"

// Ensure AppInstances implements AppLookup
var _ AppLookup = (*AppInstances)(nil)

// AppInstances is an immutable collection of application instances
type AppInstances struct {
	// apps is a map of app ID to app instance
	apps map[string]App
}

// NewAppInstances creates a new AppInstances from a slice of App instances
func NewAppInstances(apps []App) (*AppInstances, error) {
	appMap := make(map[string]App, len(apps))

	// Index apps by their ID
	for _, app := range apps {
		id := app.String()
		if _, exists := appMap[id]; exists {
			return nil, fmt.Errorf("duplicate app ID: %s", id)
		}
		appMap[id] = app
	}

	return &AppInstances{apps: appMap}, nil
}

// GetApp retrieves an application instance by ID
func (c *AppInstances) GetApp(id string) (App, bool) {
	app, exists := c.apps[id]
	return app, exists
}

// String returns a string representation of the app instances
func (c *AppInstances) String() string {
	if c == nil || len(c.apps) == 0 {
		return "AppInstances{empty}"
	}

	ids := make([]string, 0, len(c.apps))
	for id := range c.apps {
		ids = append(ids, id)
	}

	return fmt.Sprintf("AppInstances{apps: %v}", ids)
}
