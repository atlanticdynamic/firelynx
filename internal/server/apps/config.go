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

// convertEchoConfig converts domain echo config to echo DTO
func convertEchoConfig(id string, domainConfig *configEcho.EchoApp) (*echo.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("echo config cannot be nil")
	}

	// Extract response, default to app ID if empty
	response := domainConfig.Response
	if response == "" {
		response = id
	}

	return &echo.Config{
		ID:       id,
		Response: response,
	}, nil
}

// convertScriptConfig converts domain script config to script DTO
func convertScriptConfig(id string, domainConfig *configScripts.AppScript) (*script.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("script config cannot be nil")
	}

	// Check if evaluator exists
	if domainConfig.Evaluator == nil {
		return nil, fmt.Errorf("script app must have an evaluator")
	}

	// Get compiled evaluator from domain config
	compiledEvaluator, err := domainConfig.Evaluator.GetCompiledEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to get compiled evaluator for app %s: %w", id, err)
	}
	if compiledEvaluator == nil {
		return nil, fmt.Errorf("compiled evaluator is nil for app %s - domain validation may not have been run", id)
	}

	// Extract static data
	var staticData map[string]any
	if domainConfig.StaticData != nil {
		staticData = domainConfig.StaticData.Data
	}

	// Create logger for this app instance
	logger := slog.Default().With("app_type", "script", "app_id", id)

	// Get timeout from domain config
	timeout := domainConfig.Evaluator.GetTimeout()

	return &script.Config{
		ID:                id,
		CompiledEvaluator: compiledEvaluator,
		StaticData:        staticData,
		Logger:            logger,
		Timeout:           timeout,
	}, nil
}

// convertMCPConfig converts domain MCP config to MCP DTO
func convertMCPConfig(id string, domainConfig *configMCP.App) (*mcp.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("MCP config cannot be nil")
	}

	// Get compiled server from domain config
	compiledServer := domainConfig.GetCompiledServer()
	if compiledServer == nil {
		return nil, fmt.Errorf("%w for app %s", mcp.ErrServerNotCompiled, id)
	}

	return &mcp.Config{
		ID:             id,
		CompiledServer: compiledServer,
	}, nil
}
