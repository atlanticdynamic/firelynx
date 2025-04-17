# firelynx Project Design Summary

This document provides an overview of the firelynx project design and its documentation.

## Project Overview

firelynx (Model Context Protocol Application Layer) is a scriptable server implementation of the Model Context Protocol (MCP). It enables AI assistants like Claude to interact with custom tools, prompts, and resources powered by a flexible scripting environment.

## Core Features

- **MCP Protocol Support**: Implements the standardized Model Context Protocol
- **Scriptable Components**: Creates custom tools and prompt templates using scripts
- **Hot-Reloadable Configuration**: Updates configurations without downtime
- **Modular Architecture**: Separates listeners, endpoints, and applications
- **Multiple Script Engines**: Supports different scripting languages via go-polyscript
- **State Management**: Uses go-fsm for lifecycle and state tracking

## Documentation Structure

The firelynx project includes the following documentation:

1. **README.md**: Project overview, features, and quick start guide
2. **ARCHITECTURE.md**: Detailed architecture design and component interactions
3. **SPECIFICATION.md**: Technical specifications for interfaces and implementation details
4. **CLI.md**: Command-line interface documentation
5. **HOT_RELOAD.md**: Hot reload system design and implementation
6. **ERROR_HANDLING.md**: Error handling strategy and patterns

## Key Libraries

firelynx builds upon several foundational libraries:

1. **go-polyscript**: Provides the scripting engine abstraction layer
   - Supports multiple languages (Risor, Starlark, WebAssembly)
   - Handles script compilation, execution, and result processing
   - Manages static and dynamic data for scripts

2. **go-fsm**: Provides thread-safe state management
   - Manages component lifecycle states
   - Controls allowed state transitions
   - Enables state change notifications
   - Supports concurrent operations

3. **go-supervisor**: Handles service lifecycle management
   - Coordinates component startup and shutdown
   - Manages signal handling
   - Supports hot reloading
   - Enables component health monitoring

4. **mcp-go**: Implements the MCP protocol
   - Provides transport layer
   - Handles request/response formatting
   - Implements protocol-specific features

## Configuration System

firelynx uses a structured configuration with three core components:

1. **Listeners**: Protocol-specific entry points (MCP, HTTP, gRPC)
2. **Endpoints**: Route mapping between listeners and applications
3. **Applications**: Functional components (scripts, MCP implementations)

Configuration is stored in Protocol Buffers format, with TOML as the human-readable format.

## Hot Reload System

The hot reload system enables zero-downtime configuration updates by:

1. Using go-fsm to track system state during reloads
2. Validating new configurations before application
3. Transitioning components through safe state changes
4. Managing request flow during configuration transitions
5. Ensuring thread safety with atomic operations and locks
6. Properly cleaning up resources from previous configurations

## MCP Implementation

firelynx implements these MCP protocol features:

- **Tools**: Custom actions powered by scripts
- **Prompts**: Script-generated prompt templates
- **Resources**: Content access via scripts

## Command-Line Interface

firelynx provides a CLI with these main command groups:

- **server**: Start, stop, and manage the server
- **config**: Validate and update configurations
- **script**: Manage scripts
- **listeners/endpoints/apps**: Resource management

## Next Steps

1. **Implementation**: Start with the core server and configuration
2. **MCP Protocol Support**: Implement the MCP transport layer
3. **Script Integration**: Add script execution environment
4. **Hot Reload System**: Implement configuration update process
5. **CLI Development**: Build the command-line interface
6. **Testing and Documentation**: Add tests and enhance documentation