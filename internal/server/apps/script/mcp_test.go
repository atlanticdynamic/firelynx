package script

import (
	"errors"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRisorScriptApp compiles a Risor evaluator with the given code and
// static data, then returns a ScriptApp ready for testing.
func buildRisorScriptApp(t *testing.T, id, code string, staticDataMap map[string]any) *ScriptApp {
	t.Helper()

	risor := &evaluators.RisorEvaluator{Code: code}
	require.NoError(t, risor.Validate())

	domain := scripts.NewAppScript(id)
	domain.Evaluator = risor
	if staticDataMap != nil {
		domain.StaticData = &staticdata.StaticData{Data: staticDataMap}
	}

	cfg := createScriptConfig(t, id, domain)
	app, err := New(cfg)
	require.NoError(t, err)
	return app
}

func TestScriptApp_MCPToolName(t *testing.T) {
	app := buildRisorScriptApp(t, "unit-converter-app", `func tool() { return {"ok": true} }; tool()`, nil)
	assert.Equal(t, "unit-converter-app", app.MCPToolName())
}

func TestScriptApp_MCPToolDescription(t *testing.T) {
	app := buildRisorScriptApp(t, "tool-app", `func tool() { return {"ok": true} }; tool()`, nil)
	assert.Contains(t, app.MCPToolDescription(), "tool-app")
}

func TestScriptApp_MCPRawToolFunc_Success(t *testing.T) {
	const code = `
func tool() {
    args := ctx.get("args", {})
    name := args.get("name", "stranger")
    return {"greeting": "Hello, " + name}
}
tool()
`
	app := buildRisorScriptApp(t, "greeter", code, nil)
	fn := app.MCPRawToolFunc()
	require.NotNil(t, fn)

	out, err := fn(t.Context(), nil, []byte(`{"name":"world"}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"greeting":"Hello, world"}`, string(out))
}

func TestScriptApp_MCPRawToolFunc_StaticDataAvailable(t *testing.T) {
	const code = `
func tool() {
    factor := ctx.get("data", {}).get("factor", 0)
    args := ctx.get("args", {})
    n := args.get("n", 0)
    return {"result": n * factor}
}
tool()
`
	app := buildRisorScriptApp(t, "multiplier", code, map[string]any{"factor": 3})
	fn := app.MCPRawToolFunc()

	out, err := fn(t.Context(), nil, []byte(`{"n":7}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"result":21}`, string(out))
}

func TestScriptApp_MCPRawToolFunc_ScriptErrorBecomesValidationError(t *testing.T) {
	const code = `
func tool() {
    args := ctx.get("args", {})
    if args.get("name", "") == "" {
        return {"error": "name is required"}
    }
    return {"ok": true}
}
tool()
`
	app := buildRisorScriptApp(t, "tool-app", code, nil)
	fn := app.MCPRawToolFunc()

	_, err := fn(t.Context(), nil, []byte(`{}`))
	require.Error(t, err)

	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Contains(t, err.Error(), "name is required")
}

func TestScriptApp_MCPRawToolFunc_InvalidJSONInput(t *testing.T) {
	const code = `func tool() { return {"ok": true} }; tool()`
	app := buildRisorScriptApp(t, "tool-app", code, nil)
	fn := app.MCPRawToolFunc()

	_, err := fn(t.Context(), nil, []byte(`{not valid json`))
	require.Error(t, err)

	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Contains(t, err.Error(), "invalid tool input JSON")
}

func TestScriptApp_MCPRawToolFunc_EmptyInputAllowed(t *testing.T) {
	const code = `
func tool() {
    args := ctx.get("args", {})
    return {"got": args}
}
tool()
`
	app := buildRisorScriptApp(t, "tool-app", code, nil)
	fn := app.MCPRawToolFunc()

	out, err := fn(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"got":{}}`, string(out))
}

func TestScriptApp_MCPRawToolFunc_RuntimeError_ProcessingError(t *testing.T) {
	// Force a Risor runtime error: arithmetic on incompatible types. Risor
	// compiles successfully (types are dynamic) but evaluation fails when
	// the operation is attempted.
	const code = `
func tool() {
    args := ctx.get("args", {})
    s := args.get("s", "")
    return {"bad": s + 1}
}
tool()
`
	app := buildRisorScriptApp(t, "type-error-app", code, nil)
	fn := app.MCPRawToolFunc()

	_, err := fn(t.Context(), nil, []byte(`{"s":"hello"}`))
	require.Error(t, err)

	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te, "runtime errors should surface as mcpio ToolError (ProcessingError)")
	assert.Contains(t, err.Error(), "script execution failed")
}

// errors import is used by mcp_test.go via require.ErrorAs only; keep it
// pinned in case test edits drop the import.
var _ = errors.New
