package apps

import (
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/script"
)

// CreateEchoApp creates an echo app from DTO configuration
func CreateEchoApp(config *echo.Config) (App, error) {
	if config == nil {
		return nil, ErrInvalidConfigType
	}
	return echo.New(config), nil
}

// CreateScriptApp creates a script app from DTO configuration
func CreateScriptApp(config *script.Config) (App, error) {
	if config == nil {
		return nil, ErrInvalidConfigType
	}
	return script.New(config)
}

// CreateMCPApp creates an MCP app from DTO configuration
func CreateMCPApp(config *mcp.Config) (App, error) {
	if config == nil {
		return nil, ErrInvalidConfigType
	}
	return mcp.New(config)
}
