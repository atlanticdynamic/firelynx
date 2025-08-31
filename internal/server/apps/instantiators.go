package apps

import (
	"fmt"

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
	if config == nil {
		return nil, ErrInvalidConfigType
	}

	// Type assert to echo domain config
	echoConfig, ok := config.(*configEcho.EchoApp)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	// Convert domain config to DTO
	dto, err := convertEchoConfig(id, echoConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigConversionFailed, err)
	}

	return echo.New(dto), nil
}

func createScriptApp(id string, config any) (App, error) {
	if config == nil {
		return nil, ErrInvalidConfigType
	}

	// Type assert to script domain config
	scriptConfig, ok := config.(*configScripts.AppScript)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	// Convert domain config to DTO
	dto, err := convertScriptConfig(id, scriptConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigConversionFailed, err)
	}

	return script.New(dto)
}

func createMCPApp(id string, config any) (App, error) {
	if config == nil {
		return nil, ErrInvalidConfigType
	}

	// Type assert to MCP domain config
	mcpConfig, ok := config.(*configMCP.App)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	// Convert domain config to DTO
	dto, err := convertMCPConfig(id, mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigConversionFailed, err)
	}

	return mcp.New(dto)
}
