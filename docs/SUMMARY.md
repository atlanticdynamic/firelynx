# firelynx Project Design Summary

This document provides an overview of the firelynx project design and its documentation.

## Project Overview

firelynx is a scriptable application server providing HTTP and gRPC protocol support with dynamic script execution capabilities. It supports multiple scripting languages and uses a transaction-based configuration update system.

## Core Features

- **Multi-Protocol Support**: HTTP and gRPC listeners for diverse client connections
- **Scriptable Applications**: Dynamic script execution using Risor, Starlark, and WebAssembly
- **Configuration Transactions**: Saga-based configuration updates with two-phase commit
- **Modular Architecture**: Separates listeners, endpoints, and applications
- **Multiple Script Engines**: Supports different scripting languages via go-polyscript
- **State Management**: Uses go-fsm for lifecycle and transaction coordination

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


## Configuration System

firelynx uses a structured configuration with three core components:

1. **Listeners**: Protocol-specific entry points (HTTP, gRPC)
2. **Endpoints**: Route mapping between listeners and applications
3. **Applications**: Functional components (scripts, composite scripts)

Configuration is stored in Protocol Buffers format, with TOML as the human-readable format.

## Configuration Transaction System

Configuration updates use a saga pattern with two-phase commit:

1. **cfgfileloader**: Watches TOML files and creates validated transactions
2. **cfgservice**: gRPC service that creates validated transactions from client requests
3. **txmgr**: Transaction manager orchestrating saga participants
4. **Participants**: Components implementing StageConfig/CommitConfig for atomic updates

This ensures configuration changes are validated before application and can be rolled back if any participant fails during the commit phase.

## Hot Reload System

The configuration transaction system manages updates with minimal service interruption by:

1. Using go-fsm to track transaction state during configuration changes
2. Validating new configurations in isolated transactions
3. Coordinating two-phase commit across all participants
4. Managing request flow during brief configuration transitions
5. Ensuring thread safety with atomic operations and channels
6. Properly cleaning up resources when transactions complete or abort


## Command-Line Interface

firelynx provides a CLI with these main command groups:

- **server**: Start, stop, and manage the server
- **config**: Validate and update configurations
- **script**: Manage scripts
- **listeners/endpoints/apps**: Resource management

## Implementation Status

The core system is implemented with:

1. **Server Infrastructure**: HTTP and gRPC listeners with script execution
2. **Configuration Transaction System**: Saga-based updates with two-phase commit
3. **Script Integration**: Multi-language support via go-polyscript
4. **State Management**: go-fsm integration for transaction coordination
5. **CLI Interface**: Command-line tools for server and configuration management
6. **Testing**: Integration tests covering transaction scenarios