//go:build integration

package mcp

import (
	"log/slog"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	configMCP "github.com/atlanticdynamic/firelynx/internal/config/apps/mcpserver"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mcpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBaseConfig returns a minimal valid config that can host MCP-server
// apps and routes. Tests populate cfg.Apps and rebind the route's App
// pointer afterwards (transaction layer treats route.App as the expanded
// instance with merged static data).
func buildBaseConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Version: config.VersionLatest,
		Listeners: []listeners.Listener{
			{
				ID:      "test-listener",
				Address: ":0",
				Type:    listeners.TypeHTTP,
				Options: options.NewHTTP(),
			},
		},
		Endpoints: endpoints.NewEndpointCollection(endpoints.Endpoint{
			ID:         "test-endpoint",
			ListenerID: "test-listener",
			Routes: routes.RouteCollection{
				routes.Route{
					AppID:     "mcp-server",
					Condition: conditions.NewHTTP("/mcp", "*"),
				},
			},
		}),
	}
}

func TestTransaction_MCPUnknownAppRefFailsValidation(t *testing.T) {
	cfg := buildBaseConfig(t)
	mcpApp := apps.App{
		ID: "mcp-server",
		Config: &configMCP.App{
			ID: "mcp-server",
			Tools: []configMCP.Tool{
				{AppID: "ghost-app"},
			},
		},
	}
	// route.App is the expanded instance with merged static data;
	// convertAndCreateApps uses route.App OR cfg.Apps, not both for the same ID.
	cfg.Apps = apps.NewAppCollection()
	cfg.Endpoints[0].Routes[0].App = &mcpApp

	tx, err := transaction.FromTest(t.Name(), cfg, slog.Default().Handler())
	require.NoError(t, err)

	err = tx.RunValidation()
	require.Error(t, err)
	require.ErrorIs(t, err, mcpserver.ErrUnknownAppRef)
	assert.Contains(t, err.Error(), "ghost-app")
}

func TestTransaction_MCPUnsupportedPromptFailsValidation(t *testing.T) {
	cfg := buildBaseConfig(t)
	echoApp := apps.App{
		ID:     "echo-app",
		Config: &configEcho.EchoApp{Response: "hi"},
	}
	mcpApp := apps.App{
		ID: "mcp-server",
		Config: &configMCP.App{
			ID: "mcp-server",
			Prompts: []configMCP.Prompt{
				{ID: "greeting", AppID: "echo-app"},
			},
		},
	}
	// echo-app has no route, so it lives in cfg.Apps; mcp-server is the route target.
	cfg.Apps = apps.NewAppCollection(echoApp)
	cfg.Endpoints[0].Routes[0].App = &mcpApp

	tx, err := transaction.FromTest(t.Name(), cfg, slog.Default().Handler())
	require.NoError(t, err)

	err = tx.RunValidation()
	require.Error(t, err)
	require.ErrorIs(t, err, mcpserver.ErrMCPPrimitiveNotSupported)
	assert.Contains(t, err.Error(), "prompt registration is not implemented")
}
