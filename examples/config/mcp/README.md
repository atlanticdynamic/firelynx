# MCP (Model Context Protocol) Examples

This directory contains example configurations demonstrating how to create MCP servers using firelynx with different scripting languages. Each example showcases different capabilities and use cases.

## Examples Overview

### 1. `mcp-risor-calculator.toml`
**Language**: Risor (Go-like syntax)  
**Port**: 8080  
**Focus**: Mathematical operations and string manipulation

**Tools**:
- `calculate` - Mathematical calculator with operator validation
- `format_text` - String formatting and text manipulation

**Key Features**:
- Static data configuration for limits and precision
- Error handling for invalid expressions
- Support for basic arithmetic operations
- Text processing with various transformations

### 2. `mcp-starlark-data-processor.toml`
**Language**: Starlark (Python-like syntax)  
**Port**: 8081  
**Focus**: JSON data processing and analysis

**Tools**:
- `analyze_json` - Structural analysis of JSON data
- `transform_data` - Data transformation operations

**Key Features**:
- Recursive data structure analysis
- Statistical analysis of data
- Data validation and transformation
- Support for complex nested objects

### 3. `mcp-wasm-text-analyzer.toml`
**Language**: WASM (WebAssembly with Extism)  
**Port**: 8082  
**Focus**: Performance-critical text analysis

**Tools**:
- `count_vowels` - High-performance vowel counting (uses existing WASM module)
- `text_statistics` - Comprehensive text analysis (hypothetical module)
- `generate_hash` - Cryptographic hash generation (hypothetical module)
- `detect_file_type` - File type detection from binary data (hypothetical module)

**Key Features**:
- Uses compiled WASM modules for performance
- Demonstrates how to reference existing WASM files
- Shows configuration for hypothetical WASM modules
- Timeout and memory configuration for WASM execution

### 4. `mcp-multi-language-toolkit.toml`
**Languages**: All three (Risor, Starlark, WASM)  
**Port**: 8083  
**Focus**: Comprehensive toolkit using the best language for each task

**Tools**:
- `unit_converter` (Risor) - Advanced unit conversion with multiple categories
- `validate_schema` (Starlark) - JSON schema validation with custom schemas
- `analyze_text_performance` (WASM) - Performance-critical text analysis
- `data_pipeline` (Starlark) - Multi-stage data processing coordination

**Key Features**:
- Demonstrates choosing the right language for each task
- Complex static data configurations
- Multi-stage data processing pipelines
- Performance optimization through language selection

## Running the Examples

### Prerequisites

1. **Build firelynx**: 
   ```bash
   make build
   ```

2. **For WASM examples**, build the vowel counter module:
   ```bash
   cd examples/wasm/rust/char_counter
   make build
   ```

### Starting an MCP Service

```bash
# Run any of the example configurations
./bin/firelynx server -c examples/config/mcp/mcp-risor-calculator.toml

# Or use a different example
./bin/firelynx server -c examples/config/mcp/mcp-starlark-data-processor.toml
```

### Testing MCP Tools

Once a service is running, you can test the MCP tools using an MCP client or HTTP requests to the configured endpoints.

#### Example HTTP Request Format

```bash
# Test the calculator (Risor example on port 8080)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "calculate",
      "arguments": {
        "expression": "15 + 25"
      }
    }
  }'

# Test JSON analysis (Starlark example on port 8081)
curl -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "analyze_json",
      "arguments": {
        "data": {"name": "John", "age": 30, "items": [1,2,3]},
        "type": "structure"
      }
    }
  }'
```

## Configuration Patterns

### Static Data Usage

All examples demonstrate different patterns for using static data:

```toml
[apps.mcp.tools.script.static_data]
max_value = 1000
precision = "high"
allowed_operations = ["add", "subtract"]
```

Access in scripts:
- **Risor**: `ctx.get("max_value", 100)`
- **Starlark**: `ctx.get("max_value", 100)`
- **WASM**: Data passed as structured input to WASM module

### Error Handling

Examples show different error handling patterns:

- **Risor**: Return `{"error": "message"}` objects
- **Starlark**: Return dictionaries with error fields  
- **WASM**: WASM modules handle errors internally and return status

### Performance Considerations

- **Risor**: Best for mathematical operations and string manipulation
- **Starlark**: Ideal for data processing and complex logic
- **WASM**: Optimal for CPU-intensive operations and when you need maximum performance

## Creating Custom Tools

### 1. Choose the Right Language

- **Use Risor** for: Math, string processing, simple logic
- **Use Starlark** for: Data transformation, JSON processing, complex workflows
- **Use WASM** for: Performance-critical operations, binary processing, existing native code

### 2. Tool Structure

Every MCP tool needs:
```toml
[[apps.mcp.tools]]
name = "tool_name"
description = "Tool description for MCP clients"

[apps.mcp.tools.script]
# Optional static configuration
[apps.mcp.tools.script.static_data]
config_key = "config_value"

# Script implementation (choose one)
[apps.mcp.tools.script.risor]
code = '''/* Risor code */'''

[apps.mcp.tools.script.starlark] 
code = '''# Starlark code'''

[apps.mcp.tools.script.extism]
wasm_file = "path/to/module.wasm"
entry_point = "function_name"
```

### 3. Script Interface

Scripts receive MCP arguments via `args.get("param_name", default_value)` and can access static data via `ctx.get("config_key", default_value)`.

Expected return formats:
- **Success**: `{"text": "result message"}` or `{"result": data, "text": "description"}`
- **Error**: `{"error": "error message"}`
- **Complex**: Any JSON object with appropriate fields

## Security Considerations

- **Input Validation**: Always validate MCP arguments in your scripts
- **Resource Limits**: Use static data to configure timeouts and size limits
- **WASM Sandboxing**: WASM modules run in a secure sandbox with configurable memory limits
- **Timeout Protection**: All script evaluators support configurable timeouts

## Development Tips

1. **Start Simple**: Begin with basic tools and add complexity gradually
2. **Test Incrementally**: Test each tool individually before combining
3. **Use Static Data**: Leverage static data for configuration rather than hardcoding values
4. **Handle Errors Gracefully**: Always provide meaningful error messages
5. **Consider Performance**: Choose the right language for each specific task
6. **Leverage Existing WASM**: Reuse existing WASM modules when possible

## Troubleshooting

### Common Issues

1. **Script Compilation Errors**: Check syntax and ensure all variables are defined
2. **Import Errors**: Verify that required packages are available in your script environment
3. **WASM Module Not Found**: Ensure WASM file paths are correct and modules are built
4. **Timeout Errors**: Increase timeout values for long-running operations
5. **JSON Parsing Errors**: Validate that your script returns properly formatted JSON

### Debug Mode

Run firelynx with verbose logging to debug script execution:
```bash
./bin/firelynx server -c config.toml --log-level debug
```

This will show detailed information about script compilation, execution, and any errors that occur.