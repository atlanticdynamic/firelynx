package registry

import (
	"maps"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// Simple is a basic implementation of the Registry interface
type Simple struct {
	apps      map[string]apps.App
	deletions int
	mutex     sync.RWMutex
}

// New creates a new SimpleRegistry
func New() *Simple {
	return &Simple{
		apps: make(map[string]apps.App),
	}
}

// GetApp retrieves an application by ID
func (r *Simple) GetApp(id string) (apps.App, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	app, ok := r.apps[id]
	return app, ok
}

// RegisterApp adds or replaces an application in the registry
func (r *Simple) RegisterApp(app apps.App) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.apps[app.ID()] = app
	return nil
}

// UnregisterApp removes an application from the registry
func (r *Simple) UnregisterApp(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.apps, id)
	r.deletions++
	if r.deletions > 100 {
		// rebuild the registry map to free up memory
		newApps := maps.Clone(r.apps)
		r.apps = newApps
		r.deletions = 0
	}
	return nil
}
