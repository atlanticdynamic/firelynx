package apps

import (
	"fmt"
	"log/slog"

	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/script"
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

func createScriptApp(id string, config any) (App, error) {
	scriptConfig, ok := config.(*configScripts.AppScript)
	if !ok {
		return nil, fmt.Errorf("invalid config type for script app: %T", config)
	}

	logger := slog.Default().With("app_type", "script", "app_id", id)

	return script.New(id, scriptConfig, logger)
}

func createMCPApp(id string, config any) (App, error) {
	mcpConfig, ok := config.(*configMCP.App)
	if !ok {
		return nil, fmt.Errorf("invalid config type for MCP app: %T", config)
	}

	return mcp.New(id, mcpConfig)
}
