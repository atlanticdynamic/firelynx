package apps

import (
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
)

// Instantiator is a function that creates an app instance from configuration
type Instantiator func(id string, config any) (App, error)

func createEchoApp(id string, config any) (App, error) {
	// Default response is the app ID
	response := id

	// If config is provided and has a response field, use it
	if echoConfig, ok := config.(*configEcho.EchoApp); ok {
		if echoConfig.Response != "" {
			response = echoConfig.Response
		}
	}

	return echo.New(id, response), nil
}
