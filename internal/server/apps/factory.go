package apps

import (
	"errors"
	"fmt"
)

// AppFactory handles creation of app instances from configuration
type AppFactory struct {
	creators map[string]Instantiator
}

// NewAppFactory creates a new factory with standard app creators
func NewAppFactory() *AppFactory {
	return &AppFactory{
		creators: map[string]Instantiator{
			"echo":   createEchoApp,
			"script": createScriptApp,
			"mcp":    createMCPApp,
		},
	}
}

// AppDefinition is a minimal representation of an app configuration
// This matches the structure from internal/config/apps without importing it
type AppDefinition struct {
	ID     string
	Config AppConfigData
}

// AppConfigData is a minimal interface that matches internal/config/apps.AppConfig
// without importing it. This allows us to work with config data in a decoupled way.
type AppConfigData interface {
	Type() string
}

// CreateAppsFromDefinitions creates app instances from app definitions
func (f *AppFactory) CreateAppsFromDefinitions(
	definitions []AppDefinition,
) (*AppInstances, error) {
	if definitions == nil {
		return NewAppInstances([]App{})
	}

	// Build app instances from definitions
	instances := make([]App, 0, len(definitions))
	errs := []error{}

	for _, def := range definitions {
		app, err := f.createApp(def)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create app %s: %w", def.ID, err))
			continue
		}
		instances = append(instances, app)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// Create the instances collection with only the apps defined in config
	return NewAppInstances(instances)
}

// createApp creates a single app instance from a definition
func (f *AppFactory) createApp(def AppDefinition) (App, error) {
	// Validate app definition
	if def.ID == "" {
		return nil, errors.New("app ID cannot be empty")
	}

	if def.Config == nil {
		return nil, fmt.Errorf("no config specified for app %s", def.ID)
	}

	// Get app type from config
	appType := def.Config.Type()

	// Get creator for this type
	creator, ok := f.creators[appType]
	if !ok {
		return nil, fmt.Errorf("unknown app type %s for app %s", appType, def.ID)
	}

	// Create the app instance
	return creator(def.ID, def.Config)
}
