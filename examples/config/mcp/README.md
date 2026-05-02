# MCP (Model Context Protocol) Examples

This directory contains MCP server configurations that expose firelynx apps —
both multi-language script apps and typed built-in apps — as MCP tools.
firelynx's MCP support uses a **gateway model**: apps are defined as ordinary
firelynx apps, and the MCP server references them by `app_id` to expose
supported providers via [mcp-io](https://github.com/robbyt/mcp-io) +
[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk).

## Configuration

**`mcp-multi-language-toolkit.toml`** - Risor + Starlark script apps fronted by
a single MCP server (port 8083). Provides:

- **`unit_converter`** (Risor) - Convert between units (length, weight) with
  built-in conversion tables.
- **`validate_schema`** (Starlark) - Validate JSON data against predefined
  schemas.

**`mcp-typed-builtins.toml`** - First-class typed built-in apps fronted by an
MCP server (port 8084). No scripts required — schemas are derived from each
app's Go input/output types. Provides:

- **`calculate`** (calculation app) - Apply `+`, `-`, `*`, or `/` to two
  numeric operands.
- **`fileread`** (fileread app) - Read a file from a sandboxed base directory.

Built-in typed app tools such as `echo`, `calculation`, and `fileread` can be
referenced by an MCP server in either configuration. Their schemas are
generated from Go input/output types, so `input_schema` can be omitted.

## Quick Start

1. Build firelynx:
   ```bash
   make build
   ```

2. Run an example:
   ```bash
   # Script-backed tools (port 8083)
   ./bin/firelynx server -c examples/config/mcp/mcp-multi-language-toolkit.toml

   # Typed built-in tools (port 8084)
   ./bin/firelynx server -c examples/config/mcp/mcp-typed-builtins.toml
   ```

3. Connect any MCP client to `http://localhost:8083/mcp` or
   `http://localhost:8084/mcp` (e.g. via `mcp-cli`, the
   `modelcontextprotocol/go-sdk` client, or Claude Desktop).

## Configuration Pattern

The gateway model separates the *tool implementation* (a script app) from the
*MCP server* that exposes it. A script app is a normal firelynx app and is
defined exactly the same way you would for HTTP serving:

```toml
[[apps]]
id = "unit-converter-app"
type = "script"

[apps.script.static_data]
# any static config the script can read via ctx.get("data", {})

[apps.script.risor]
code = '''
func convert() {
    args := ctx.get("args", {})    # runtime input from the MCP client
    static := ctx.get("data", {})  # static_data above
    # ... return a map; {"error": "..."} surfaces as a tool error
}
convert()
'''
```

Then the MCP server app references it by `app_id`:

```toml
[[apps]]
id = "multi-toolkit"
type = "mcp"

[[apps.mcp.tools]]
id = "unit_converter"               # MCP tool name shown to clients
app_id = "unit-converter-app"       # firelynx app that backs it
input_schema = '''
{ "type": "object", "properties": { ... }, "required": [...] }
'''
```

Notes:

- `id` is optional. When empty, the gateway falls back to the backing app's
  `MCPToolName()` (which for script apps is just the `app_id`).
- `input_schema` is **required** for script-backed tools because mcp-io's raw
  registration path needs an explicit schema. Typed provider apps such as
  `echo`, `calculation`, and `fileread` let mcp-io derive schemas
  automatically, so those can omit `input_schema`/`output_schema`.
- `input_schema` is rejected for typed-only providers because mcp-io derives
  typed tool schemas from Go input/output structs.
- `output_schema` is accepted and JSON-validated for future compatibility, but
  it is not currently forwarded to MCP clients.
- Prompt and resource config fields exist in the schema, but runtime support is
  intentionally tool-only today. Configuring prompts or resources fails
  validation with an unsupported-primitive error.

### Typed Built-In Tool Examples

```toml
[[apps]]
id = "calculator"
type = "calculation"
[apps.calculation]

[[apps]]
id = "workspace-reader"
type = "fileread"
[apps.fileread]
base_directory = "${WORKSPACE_DIR}"

[[apps]]
id = "tools"
type = "mcp"

[[apps.mcp.tools]]
app_id = "calculator"

[[apps.mcp.tools]]
app_id = "workspace-reader"
```

### Script ↔ MCP Contract

Inside a script app exposed as an MCP tool:

- **Input** is namespaced under `ctx.get("args", {})`.
- **Static data** is namespaced under `ctx.get("data", {})`.
- **Output** is whatever the script returns:
  - A map containing `{"error": "..."}` is surfaced as a structured tool error
    (`mcpio.ValidationError`).
  - Any other JSON-serializable value becomes the tool's structured result.

This matches the namespacing in `internal/server/apps/script/CLAUDE.md`.

## Development Tips

1. Iterate on the script app first using the HTTP path; once it works there it
   will work as an MCP tool.
2. Keep configuration in `static_data` instead of hardcoding it inside the
   script.
3. Set `id` on each `[[apps.mcp.tools]]` block to give clients a friendly tool
   name independent of the underlying app id.
4. Use Risor for math/string-heavy work and Starlark for structured data
   validation — the toolkit example demonstrates both.
