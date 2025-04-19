# firelynx - Model Context Protocol Server

[![Go Reference](https://pkg.go.dev/badge/github.com/atlanticdynamic/firelynx.svg)](https://pkg.go.dev/github.com/atlanticdynamic/firelynx)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

firelynx is a scriptable implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server. It enables AI assistants like Claude to interact with custom tools, prompts, and resources powered by a flexible scripting environment.

## Features

- **MCP Protocol Support**: Implements the standardized [Model Context Protocol](https://modelcontextprotocol.io/)
- **Scriptable Tools and Prompts**: Create custom tools and prompt templates using multiple scripting languages
- **Hot-Reloadable Configuration**: Update server configuration without downtime
- **Modular Architecture**: Clear separation between listeners, endpoints, and applications
- **Multiple Script Engines**: Powered by [go-polyscript](https://github.com/robbyt/go-polyscript) for language flexibility
- **Graceful Lifecycle Management**: Handled by [go-supervisor](https://github.com/robbyt/go-supervisor)

## Quick Start

### Installation

```bash
# Install from source
go install github.com/atlanticdynamic/firelynx/cmd/firelynx@latest

# Or build from source
git clone https://github.com/atlanticdynamic/firelynx.git
cd firelynx
make install
```

### Running the Server

```bash
# Start the server with empty configuration (awaiting client to apply config)
firelynx server

# Start with custom configuration file
firelynx server --config /path/to/config.toml

# Start with specific listen address
firelynx server --listen :8765
```

### Configuration

firelynx uses TOML configuration files with the following structure:

```toml
# firelynx Server Configuration
[[listeners]]
id = "mcp_listener"
protocol = "mcp"
address = "localhost:8765"

[listeners.protocol_options.mcp]
connection_timeout = "60s"
max_connections = 100

[[endpoints]]
id = "tools_endpoint"
listener_id = "mcp_listener"
app_id = "sample_tools"

[endpoints.route]
mcp_resource = "tools/call"

[[apps]]
id = "sample_tools"

[apps.app_type.mcp]
name = "Sample Tools"
description = "Example tools for demonstration"

[apps.app_type.mcp.mcp_implementation.tool]
script = '''
// Tool implementation in Risor
result := ctx.get("input", "") + " processed"
return {
  "isError": false,
  "content": result
}
'''
engine = "risor"
parameter_schema = '''
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input to process"
    }
  },
  "required": ["input"]
}
'''
```

## Architecture

firelynx follows a three-layer architecture:

1. **Listeners**: Protocol-specific entry points (MCP, HTTP, gRPC)
2. **Endpoints**: Connection mapping between listeners and applications
3. **Applications**: Functional components including script apps and MCP implementations

For detailed architecture documentation, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## MCP Protocol Support

firelynx implements the following MCP protocol features:

- **Tools**: Create custom tools that Claude can use to perform actions
- **Prompts**: Define prompt templates with arguments
- **Resources**: Access and retrieve content from various sources

For more information on the MCP protocol, visit the [official documentation](https://modelcontextprotocol.io/).

## Development

### Prerequisites

- Go 1.20 or later
- Protocol Buffer compiler and tools (`buf`)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/atlanticdynamic/firelynx.git
cd firelynx

# Generate protobuf code
make protogen

# Build the binary
make build

# Run tests
make test
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md): Overall system design
- [Specification](docs/SPECIFICATION.md): Technical specifications
- [CLI](docs/CLI.md): Command-line interface documentation
- [Hot Reload](docs/HOT_RELOAD.md): Hot reload system design
- [Error Handling](docs/ERROR_HANDLING.md): Error handling strategy

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.