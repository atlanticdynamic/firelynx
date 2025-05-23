package transaction

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// convertToAppDefinitions converts config.Apps to server app definitions
// This adapter allows the server/apps package to work with config data
// without directly importing the config types
func convertToAppDefinitions(configApps apps.AppCollection) []serverApps.AppDefinition {
	definitions := make([]serverApps.AppDefinition, 0, len(configApps))

	for _, app := range configApps {
		definitions = append(definitions, serverApps.AppDefinition{
			ID:     app.ID,
			Config: app.Config, // app.Config already implements the Type() method we need
		})
	}

	return definitions
}
