package apps

import "github.com/atlanticdynamic/firelynx/internal/server/apps/echo"

// Instantiator is a function that creates an app instance from configuration
type Instantiator func(id string, config any) (App, error)

func createEchoApp(id string, _ any) (App, error) {
	// For echo apps, we don't need to validate the config structure
	// since the echo app doesn't use configuration data
	return echo.New(id), nil
}
