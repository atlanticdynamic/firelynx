# firelynx Technical Specification

This document provides a detailed technical specification for the firelynx application components, interfaces, and configuration.

## Configuration Format

firelynx uses a structured configuration format defined in Protocol Buffers. The configuration can be written in TOML format and is converted to Protocol Buffers internally.

### TOML Configuration

The TOML configuration format is designed to be human-readable and easy to edit. Here's an example TOML configuration:

```toml
# Server Configuration
version = "v1"

# Logging Configuration
[logging]
format = "json"  # "json" or "txt"
level = "info"   # "debug", "info", "warn", "error", or "fatal"

# Listener Configuration
[[listeners]]
id = "http_listener"
address = ":8080"

# HTTP Listener Options
# IMPORTANT: Use [listeners.http] not [listeners.protocol_options.http]
[listeners.http]
read_timeout = "30s"
write_timeout = "30s"
drain_timeout = "30s"

# gRPC Listener (Example)
[[listeners]]
id = "grpc_listener"
address = ":9090"

# gRPC Listener Options
# IMPORTANT: Use [listeners.grpc] not [listeners.protocol_options.grpc]
[listeners.grpc]
max_connection_idle = "5m"
max_connection_age = "30m"
max_concurrent_streams = 1000

# Endpoint Configuration
[[endpoints]]
id = "api_endpoint"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "sample_app"
http_path = "/api/v1"

# Application Configuration
[[apps]]
id = "sample_app"

[apps.script.risor]
code = '''
// Risor script code here
function handle(req) {
  return { status: 200, body: "Hello, World!" }
}
'''
timeout = "10s"
```

#### Important Note on Listener Protocol Options

While the Protocol Buffer definition uses a field named `protocol_options` that contains either `http` or `grpc` fields, in TOML configuration you should use:

- `[listeners.http]` for HTTP listener options (not `[listeners.protocol_options.http]`)
- `[listeners.grpc]` for gRPC listener options (not `[listeners.protocol_options.grpc]`)

This difference exists due to how the TOML-to-Protocol-Buffer conversion works internally.

### Protocol Buffer Schema

firelynx uses Protocol Buffers to define the configuration schema with these core components:

- **ServerConfig**: Root configuration containing listeners, endpoints, and applications
- **Listeners**: Protocol-specific entry points (HTTP, gRPC)
- **Endpoints**: Route mapping between listeners and applications
- **Applications**: Functional components (scripts, composite scripts)

## Application Types

firelynx supports the following application types:

### 1. Script App

A single script executed for each request with these characteristics:

- **Code**: Script content
- **Engine**: Script engine ("risor", "starlark", "extism", "native")
- **Entrypoint**: Function to call
- **Static Data**: Configuration data available to the script

The supported script engines are:

- **risor**: A Go-like dynamic scripting language optimized for embedding
- **starlark**: Python-like configuration language designed for sandboxed execution
- **extism (WASM)**: WebAssembly plugin system that supports multiple languages
- **native**: Built-in Go functions registered directly with the server

### 2. Composite Script App

A chain of scripts executed in sequence with:

- **Script App IDs**: List of script apps to run in sequence
- **Shared Data**: Data shared across all scripts in the chain
- **Execution Options**: Configuration for chain execution


## Configuration Translation

firelynx uses TOML as its human-readable configuration format. A marshaling layer translates between TOML and Protocol Buffers internally.

## Script Execution Environment

Scripts in firelynx have access to the following:

### 1. Context Object

A single `ctx` object containing:
- Request-specific data
- Static configuration data
- Helper functions and utilities

### 2. Script Return Format

Scripts must return structured data:

#### For Tool Scripts:
```javascript
{
  "isError": false,  // Boolean indicating success/failure
  "content": "...",  // String or structured data with result
  "metadata": {}     // Optional metadata
}
```

#### For Prompt Scripts:
```javascript
{
  "title": "...",    // Optional title for the prompt
  "content": "...",  // The formatted prompt text
  "metadata": {}     // Optional metadata
}
```

## Interfaces

firelynx defines several key interfaces for applications, listeners, endpoints, and state management. Core interfaces include Application for all app types, Listener for protocol handlers, and Endpoint for request routing.

## Library Integrations

firelynx integrates with several Go libraries:

- **go-supervisor**: Lifecycle management for concurrent components
- **go-fsm**: State machine implementation for transaction management
- **go-polyscript**: Multi-language script execution (Risor, Starlark, WebAssembly)


## Error Handling

See [ERROR_HANDLING.md](ERROR_HANDLING.md) for the detailed error handling strategy.

## Logging

firelynx uses structured logging with Go's slog package, supporting both JSON and text formats with configurable log levels.

## Configuration Client

The configuration client communicates with the server via gRPC using the ConfigService protocol to send configuration updates.