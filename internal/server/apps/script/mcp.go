package script

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	mcpio "github.com/robbyt/mcp-io"
)

// MCPToolName returns the script app's ID as the default tool name. Users
// typically override this via [[apps.mcp.tools]].id when wiring the tool.
func (s *ScriptApp) MCPToolName() string {
	return s.id
}

// MCPToolDescription returns a generic description. Script apps are
// generic — the gateway has no language-level signal for a meaningful tool
// description. Set [[apps.mcp.tools]].id and document the tool in TOML
// comments instead.
func (s *ScriptApp) MCPToolDescription() string {
	return fmt.Sprintf("Script-backed MCP tool: %s", s.id)
}

// MCPRawToolFunc returns a raw tool function suitable for mcpio.WithRawTool.
//
// The function unmarshals the raw input JSON into map[string]any, runs the
// pre-compiled evaluator with the static config + runtime args namespaced
// per script/CLAUDE.md ({"data": {...}, "args": {...}}), and marshals the
// result back to JSON. Scripts that return {"error": "..."} surface as
// mcpio.ValidationError so MCP clients receive a structured tool error.
func (s *ScriptApp) MCPRawToolFunc() mcpio.RawToolFunc {
	return func(ctx context.Context, _ mcpio.RequestContext, input []byte) ([]byte, error) {
		var args map[string]any
		if len(input) > 0 {
			if err := json.Unmarshal(input, &args); err != nil {
				return nil, mcpio.ValidationError(fmt.Sprintf("invalid tool input JSON: %v", err))
			}
		}
		if args == nil {
			args = map[string]any{}
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, s.execTimeout)
		defer cancel()

		appStatic, err := s.appStaticProvider.GetData(timeoutCtx)
		if err != nil {
			return nil, fmt.Errorf("script app static data: %w", err)
		}

		scriptData := map[string]any{
			"data": maps.Clone(appStatic),
			"args": args,
		}

		contextProvider := data.NewContextProvider(constants.EvalData)
		enrichedCtx, err := contextProvider.AddDataToContext(timeoutCtx, scriptData)
		if err != nil {
			return nil, fmt.Errorf("script app context: %w", err)
		}

		result, err := s.evaluator.Eval(enrichedCtx)
		if err != nil {
			if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				return nil, mcpio.ProcessingError("script execution timeout")
			}
			return nil, mcpio.ProcessingError(fmt.Sprintf("script execution failed: %v", err))
		}

		out := result.Interface()

		// Surface script-side {"error": "..."} as a validation error.
		if m, ok := out.(map[string]any); ok {
			if msg, hasErr := m["error"].(string); hasErr && msg != "" {
				return nil, mcpio.ValidationError(msg)
			}
		}

		outputJSON, err := json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("script result marshal: %w", err)
		}
		return outputJSON, nil
	}
}
