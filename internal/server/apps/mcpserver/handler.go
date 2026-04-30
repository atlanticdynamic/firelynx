package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// BuildHandler constructs an mcp-io HTTP handler from the supplied Config.
//
// For each Tool ref it consults the resolved app:
//   - if the app implements MCPTypedToolProvider and the user did not supply
//     an input_schema override, the typed path is used (mcp-io auto-generates
//     the schema from Go types).
//   - if the app implements MCPRawToolProvider, the raw path is used. A user
//     input_schema is required because mcp-io's WithRawTool refuses nil
//     schemas.
//   - if both interfaces are implemented and an input_schema override is
//     present, the raw path is preferred so the override takes effect.
//   - typed-only providers reject input_schema overrides because mcp-io
//     derives schemas from Go types.
//
// Cross-reference and provider-conformance must already be validated via
// App.ValidateRefs before calling this — BuildHandler returns an error if a
// ref cannot be resolved or wired.
func BuildHandler(cfg *Config, lookup AppLookup, serverName string) (http.Handler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if lookup == nil {
		return nil, fmt.Errorf("app lookup must not be nil")
	}

	// Stateless + JSON response: simpler to host behind a generic HTTP router
	// like firelynx's, where SSE streaming session tracking adds complexity
	// without value for the MCP request/response use case.
	opts := []mcpio.Option{
		mcpio.WithName(serverName),
		mcpio.WithHTTPTransport(
			mcpwrapper.WithStateless(),
			mcpwrapper.WithJSONResponse(),
		),
	}

	for i, ref := range cfg.Tools {
		opt, err := buildToolOption(ref, lookup)
		if err != nil {
			return nil, fmt.Errorf("tool[%d] (app_id=%q): %w", i, ref.AppID, err)
		}
		opts = append(opts, opt)
	}

	if len(cfg.Prompts) > 0 {
		return nil, fmt.Errorf("prompt registration is not yet wired: %d prompt refs configured", len(cfg.Prompts))
	}
	if len(cfg.Resources) > 0 {
		return nil, fmt.Errorf("resource registration is not yet wired: %d resource refs configured", len(cfg.Resources))
	}

	return mcpio.NewHandler(opts...)
}

// buildToolOption resolves a single ToolRef to an mcp-io Option, picking the
// typed or raw registration path based on what the backing app implements
// and whether the user supplied a schema override.
func buildToolOption(ref ToolRef, lookup AppLookup) (mcpio.Option, error) {
	app, ok := lookup(ref.AppID)
	if !ok {
		return nil, fmt.Errorf("%w", ErrUnknownAppRef)
	}

	typed, hasTyped := app.(MCPTypedToolProvider)
	raw, hasRaw := app.(MCPRawToolProvider)
	if !hasTyped && !hasRaw {
		return nil, fmt.Errorf("%w", ErrAppNotMCPProvider)
	}

	name := ref.ID
	if name == "" {
		switch {
		case hasTyped:
			name = typed.MCPToolName()
		case hasRaw:
			name = raw.MCPToolName()
		}
	}

	if hasRaw && (ref.InputSchema != "" || !hasTyped) {
		if ref.InputSchema == "" {
			return nil, fmt.Errorf(
				"raw tool provider %q requires an input_schema in TOML; mcp-io.WithRawTool cannot derive one",
				ref.AppID,
			)
		}

		var schema any
		if err := json.Unmarshal([]byte(ref.InputSchema), &schema); err != nil {
			return nil, fmt.Errorf("invalid input_schema JSON: %w", err)
		}

		return mcpio.WithRawTool(name, raw.MCPToolDescription(), schema, raw.MCPRawToolFunc()), nil
	}

	if ref.InputSchema != "" {
		return nil, fmt.Errorf(
			"typed tool provider %q does not support input_schema overrides (mcp-io WithTool auto-derives the schema from Go types)",
			ref.AppID,
		)
	}

	return typed.MCPToolOption(name), nil
}
