package transaction

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configComposite "github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcp"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/script"
)

var (
	ErrConfigNil             = errors.New("config cannot be nil")
	ErrEvaluatorNil          = errors.New("script app must have an evaluator")
	ErrCompiledEvaluatorNil  = errors.New("compiled evaluator is nil - domain validation may not have been run")
	ErrCompositeNotSupported = errors.New("composite apps are not yet supported in server layer")
	ErrDuplicateAppID        = errors.New("duplicate app ID")
	ErrUnknownAppType        = errors.New("unknown app type")
)

// convertEchoConfig converts domain echo config to echo DTO
func convertEchoConfig(id string, domainConfig *configEcho.EchoApp) (*echo.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("failed to convert echo config: %w", ErrConfigNil)
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
		return nil, fmt.Errorf("failed to convert script config: %w", ErrConfigNil)
	}

	// Check if evaluator exists
	if domainConfig.Evaluator == nil {
		return nil, fmt.Errorf("failed to convert script config: %w", ErrEvaluatorNil)
	}

	// Get compiled evaluator from domain config
	compiledEvaluator, err := domainConfig.Evaluator.GetCompiledEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to get compiled evaluator for app %s: %w", id, err)
	}
	if compiledEvaluator == nil {
		return nil, fmt.Errorf("failed to convert script app %s: %w", id, ErrCompiledEvaluatorNil)
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
		return nil, fmt.Errorf("failed to convert MCP config: %w", ErrConfigNil)
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

// convertAndCreateApps collects apps from domain config, converts them to DTOs, and creates instances
func convertAndCreateApps(cfg *config.Config) (*serverApps.AppInstances, error) {
	// First collect unique apps from routes (these have merged static data)
	uniqueApps := make(map[string]apps.App)

	// Collect expanded apps from routes
	for _, endpoint := range cfg.Endpoints {
		for _, route := range endpoint.Routes {
			if route.App != nil {
				// Use the expanded app instance which has merged static data
				if _, exists := uniqueApps[route.App.ID]; exists {
					return nil, fmt.Errorf("%w in routes: %s", ErrDuplicateAppID, route.App.ID)
				}
				uniqueApps[route.App.ID] = *route.App
			}
		}
	}

	// Add any apps that don't have routes (not expanded)
	if cfg.Apps != nil {
		for app := range cfg.Apps.All() {
			if _, exists := uniqueApps[app.ID]; exists {
				return nil, fmt.Errorf("%w: %s", ErrDuplicateAppID, app.ID)
			}
			uniqueApps[app.ID] = app
		}
	}

	// Convert domain apps to server apps using DTO pattern
	var appInstances []serverApps.App
	for _, domainApp := range uniqueApps {
		serverApp, err := convertDomainToServerApp(domainApp.ID, domainApp.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to convert app %s: %w", domainApp.ID, err)
		}
		appInstances = append(appInstances, serverApp)
	}

	return serverApps.NewAppInstances(appInstances)
}

// convertDomainToServerApp converts a domain app config to a server app instance
func convertDomainToServerApp(id string, domainConfig apps.AppConfig) (serverApps.App, error) {
	switch appConfig := domainConfig.(type) {
	case *configEcho.EchoApp:
		dto, err := convertEchoConfig(id, appConfig)
		if err != nil {
			return nil, err
		}
		return echo.New(dto), nil

	case *configScripts.AppScript:
		dto, err := convertScriptConfig(id, appConfig)
		if err != nil {
			return nil, err
		}
		return script.New(dto)

	case *configMCP.App:
		dto, err := convertMCPConfig(id, appConfig)
		if err != nil {
			return nil, err
		}
		return mcp.New(dto)

	case *configComposite.CompositeScript:
		return nil, fmt.Errorf("failed to convert composite app %s: %w", id, ErrCompositeNotSupported)

	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownAppType, domainConfig)
	}
}
