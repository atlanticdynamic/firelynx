# Script Applications

The `scripts` package defines configuration for script-based applications that execute user-provided code.

## Script Engines

Script apps support multiple engines via go-polyscript:

- **Risor**: Go-like scripting language
- **Starlark**: Python-like configuration language  
- **Extism**: WebAssembly plugins

## Configuration Structure

Each script app contains:
- Script code
- Engine type
- Static data passed to script
- Timeout settings
- Entry point function

## Script Execution Context

Scripts receive a context object containing:
- Request data
- Static configuration data
- Helper functions

Scripts return structured responses following conventions for tool results or prompt generation.