// Package apps provides interfaces and registry for application handlers
package apps

import "sync"

// SimpleRegistry is a basic implementation of the Registry interface
type SimpleRegistry struct {
	apps  map[string]App
	mutex sync.RWMutex
}

// NewSimpleRegistry creates a new SimpleRegistry
func NewSimpleRegistry() *SimpleRegistry {
	return &SimpleRegistry{
		apps: make(map[string]App),
	}
}

// GetApp retrieves an application by ID
func (r *SimpleRegistry) GetApp(id string) (App, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	app, ok := r.apps[id]
	return app, ok
}

// RegisterApp adds or replaces an application in the registry
func (r *SimpleRegistry) RegisterApp(app App) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.apps[app.ID()] = app
	return nil
}

// UnregisterApp removes an application from the registry
func (r *SimpleRegistry) UnregisterApp(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.apps, id)
	return nil
}
