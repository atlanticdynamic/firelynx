package transaction

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configCalculation "github.com/atlanticdynamic/firelynx/internal/config/apps/calculation"
	configComposite "github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configFileRead "github.com/atlanticdynamic/firelynx/internal/config/apps/fileread"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcpserver"
	configScripts "github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/calculation"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/fileread"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcpserver"
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

	// Get the exec timeout deadline
	timeout := domainConfig.Evaluator.GetTimeout()

	return &script.Config{
		ID:                id,
		CompiledEvaluator: compiledEvaluator,
		StaticData:        staticData,
		Logger:            logger,
		ExecTimeout:       timeout,
	}, nil
}

// convertMCPConfig converts domain MCP config to mcpserver DTO. The returned
// Config preserves the per-primitive references (Tool/Prompt/Resource) so the
// runtime can register them with mcp-io without losing any user-supplied
// schema overrides or ID fields.
func convertMCPConfig(id string, domainConfig *configMCP.App) (*mcpserver.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("failed to convert MCP config: %w", ErrConfigNil)
	}

	cfg := &mcpserver.Config{ID: id}

	for _, t := range domainConfig.Tools {
		cfg.Tools = append(cfg.Tools, mcpserver.ToolRef{
			ID:           t.ID,
			AppID:        t.AppID,
			InputSchema:  t.Schema.Input,
			OutputSchema: t.Schema.Output,
		})
	}

	for _, p := range domainConfig.Prompts {
		cfg.Prompts = append(cfg.Prompts, mcpserver.PromptRef{
			ID:          p.ID,
			AppID:       p.AppID,
			InputSchema: p.Schema.Input,
		})
	}

	for _, r := range domainConfig.Resources {
		cfg.Resources = append(cfg.Resources, mcpserver.ResourceRef{
			ID:          r.ID,
			AppID:       r.AppID,
			URITemplate: r.URITemplate,
		})
	}

	return cfg, nil
}

// convertCalculationConfig converts domain calculation config to calculation DTO.
func convertCalculationConfig(
	id string,
	domainConfig *configCalculation.App,
) (*calculation.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("failed to convert calculation config: %w", ErrConfigNil)
	}

	return &calculation.Config{ID: id}, nil
}

// convertFileReadConfig converts domain fileread config to fileread DTO.
func convertFileReadConfig(
	id string,
	domainConfig *configFileRead.App,
) (*fileread.Config, error) {
	if domainConfig == nil {
		return nil, fmt.Errorf("failed to convert fileread config: %w", ErrConfigNil)
	}

	return &fileread.Config{
		ID:                    id,
		BaseDirectory:         domainConfig.BaseDirectory,
		AllowExternalSymlinks: domainConfig.AllowExternalSymlinks,
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

	instances, err := serverApps.NewAppInstances(appInstances)
	if err != nil {
		return nil, err
	}

	// Cross-component validation: every MCP server's Tool/Prompt/Resource
	// references must resolve to an app that implements the matching
	// provider interface. Wire each MCP server's AppLookup once it passes.
	if err := wireMCPServers(instances); err != nil {
		return nil, err
	}

	return instances, nil
}

// wireMCPServers validates each *mcpserver.App in the registry against the
// other apps and, on success, calls Build to install the AppLookup. Per
// transaction/CLAUDE.md, cross-component reference checks live here rather
// than in domain validation.
func wireMCPServers(instances *serverApps.AppInstances) error {
	lookup := mcpserver.AppLookup(instances.GetApp)

	var errs []error
	for app := range instances.All() {
		mcpApp, ok := app.(*mcpserver.App)
		if !ok {
			continue
		}
		if err := mcpApp.ValidateRefs(lookup); err != nil {
			errs = append(errs, fmt.Errorf("mcp server %q: %w", mcpApp.String(), err))
			continue
		}
		if err := mcpApp.Build(lookup); err != nil {
			errs = append(errs, fmt.Errorf("mcp server %q: %w", mcpApp.String(), err))
		}
	}
	return errors.Join(errs...)
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
		return mcpserver.New(dto), nil

	case *configCalculation.App:
		dto, err := convertCalculationConfig(id, appConfig)
		if err != nil {
			return nil, err
		}
		return calculation.New(dto), nil

	case *configFileRead.App:
		dto, err := convertFileReadConfig(id, appConfig)
		if err != nil {
			return nil, err
		}
		return fileread.New(dto), nil

	case *configComposite.CompositeScript:
		return nil, fmt.Errorf("failed to convert composite app %s: %w", id, ErrCompositeNotSupported)

	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownAppType, domainConfig)
	}
}
