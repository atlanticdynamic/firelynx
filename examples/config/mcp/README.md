# MCP (Model Context Protocol) Example

This directory contains a comprehensive MCP server configuration demonstrating multi-language scripting capabilities.

## Configuration

**`mcp-multi-language-toolkit.toml`** - Multi-language toolkit combining Risor and Starlark for different types of operations.

### Tools Provided

- **`unit_converter`** (Risor) - Convert between units (length, weight) with built-in conversion tables
- **`validate_schema`** (Starlark) - Validate JSON data against predefined schemas

## Quick Start

1. Build firelynx:
   ```bash
   make build
   ```

2. Run the example:
   ```bash
   ./bin/firelynx server -c examples/config/mcp/mcp-multi-language-toolkit.toml
   ```

3. Test with an MCP client at `http://localhost:8083/mcp`

## Language Strategy

- **Risor**: Mathematical operations, unit conversions, string manipulation
- **Starlark**: Data processing, schema validation, complex workflows

## Configuration Patterns

### Tool Structure
```toml
[[apps.mcp.tools]]
name = "tool_name"
description = "What this tool does"

[apps.mcp.tools.script.static_data]
config_key = "config_value"

[apps.mcp.tools.script.risor]  # or .starlark
code = '''/* script implementation */'''
```

### Script Interface
- **Input**: `args.get("param_name", default)`
- **Static Data**: `ctx.get("data", {}).get("config_key", default)`
- **Output**: `{"text": "result"}` or `{"error": "message"}`

### Error Handling
Scripts can return various types:
- **Map with error**: `{"error": "message"}` - Treated as tool error
- **Map with success**: `{"text": "result", "value": 42}` - Structured success response
- **String/bytes**: Returned as plain text content
- **Other types**: Converted to text representation

## Development Tips

1. Start with simple tools and add complexity gradually
2. Use static data for configuration instead of hardcoded values
3. Test each tool individually before combining
4. Choose the right language for each specific task
5. Always validate inputs and handle errors gracefully

For detailed implementation examples, see the header comments in each TOML file.