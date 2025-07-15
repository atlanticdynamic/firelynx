# firelynx - Model Context Protocol Server

[![Go Reference](https://pkg.go.dev/badge/github.com/atlanticdynamic/firelynx.svg)](https://pkg.go.dev/github.com/atlanticdynamic/firelynx)
[![Go Report Card](https://goreportcard.com/badge/github.com/atlanticdynamic/firelynx)](https://goreportcard.com/report/github.com/atlanticdynamic/firelynx)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=atlanticdynamic_firelynx&metric=coverage)](https://sonarcloud.io/summary/new_code?id=atlanticdynamic_firelynx)
[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

firelynx is a scriptable implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server. It enables AI assistants like Claude to interact with custom tools, prompts, and resources powered by a scripting environment.

## Features

- **MCP Protocol Support**: Implements the standardized [Model Context Protocol](https://modelcontextprotocol.io/)
- **Scriptable Tools and Prompts**: Create custom tools and prompt templates using multiple scripting languages
- **Hot-Reloadable Configuration**: Update server configuration via gRPC or file reload without stopping the server
- **Modular Architecture**: Separation between listeners, endpoints, and applications
- **Multiple Script Engines**: Powered by [go-polyscript](https://github.com/robbyt/go-polyscript)
- **Lifecycle Management**: Handled by [go-supervisor](https://github.com/robbyt/go-supervisor)

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
# Start with a configuration file (gRPC config API services disabled)
firelynx server --config /path/to/config.toml

# Start with an empty configuration (enable gRPC services on port 8765)
firelynx server --listen :8765

# Start with an initial config AND enable the gRPC listener for updates
firelynx server --config /path/to/config.toml --listen :8765

# Use the client CLI to interact with the server
firelynx client --server localhost:8765
```

### Configuration

firelynx uses TOML configuration files with the following structure:

```toml
# firelynx Server Configuration
# TBD...
```

## Architecture

firelynx follows a three-layer architecture:

1. **Listeners**: Protocol-specific entry points (MCP, HTTP, gRPC)
2. **Endpoints**: Connection mapping between listeners and applications
3. **Applications**: Functional components including script apps and MCP implementations

## Development

Requires Go 1.24 or later to compile.

```bash
# Clone the repository
git clone https://github.com/atlanticdynamic/firelynx.git
cd firelynx

# Generate protobuf code
make protogen

# Run tests
make test
make test-all

# Compile the binary
make build

# Run the compiled server/client binary
./bin/firelynx --help
```

## Documentation

Documentation is located near the code in README files throughout the codebase:

- [Configuration Domain Model](internal/config/README.md): Configuration validation and domain model
- [Configuration Transactions](internal/config/transaction/README.md): Saga pattern for configuration updates
- [Transaction Manager](internal/server/runnables/txmgr/README.md): Configuration transaction coordination
- [CLI Usage](cmd/firelynx/README.md): Command-line interface and commands

## License

GPL v3 - See [LICENSE](LICENSE) for details.