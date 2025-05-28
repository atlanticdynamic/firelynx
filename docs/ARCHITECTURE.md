# firelynx Architecture

This document describes the architecture of the firelynx application, its components, and how they interact.

## Overview

firelynx is a server application that provides configurable application routing and scripting capabilities. It is built with a modular design that emphasizes separation of concerns and enables hot-reloading of configurations.

## System Architecture

firelynx follows a client-server architecture where:

1. **firelynx Server**: Listens for connections from MCP clients and configuration updates
2. **MCP Clients**: AI applications (Claude, etc.) that connect to firelynx to access tools and resources
3. **Configuration Client**: Sends configuration updates to the firelynx server

```
┌────────────────┐    MCP Protocol    ┌─────────────────┐
│                │◄──────────────────►│                 │
│   MCP Client   │                    │                 │
│  (e.g. Claude) │                    │                 │
└────────────────┘                    │ firelynx Server │
                      gRPC Listener   │                 │
┌────────────────┐   ( or protobuf )  │                 │
│  Configuration │◄──────────────────►│                 │
│     Client     │                    │                 │
└────────────────┘                    └────────────────┘
```

## Core Components

The firelynx server consists of several core components:

### 1. Application Layers

firelynx follows a three-layer architecture:

```
┌─────────────────────────────────────────┐
│              Listeners                  │
│  (HTTP, gRPC, Unix Socket)              │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│              Endpoints                  │
│  (Request Routing and Mapping)          │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│             Applications                │
│  (Echo App, Script Apps, Composite)     │
└─────────────────────────────────────────┘
```

- **Listeners**: Protocol-specific server components that accept connections (HTTP, gRPC)
- **Endpoints**: Map incoming requests to the appropriate application
- **Applications**: Implement functionality (echo, scripts, composite scripts)

### 2. Configuration Transaction System

firelynx implements a saga pattern for atomic configuration updates with the following components:

- **cfgfileloader**: Watches TOML files and creates validated configuration transactions
- **cfgservice**: Provides gRPC interface for configuration updates, creates validated transactions
- **txmgr**: Manages transaction lifecycle and coordinates the saga orchestrator
- **Saga Orchestrator**: Implements two-phase commit across all registered participants
- **Participants**: Components (like HTTP Listener) that implement StageConfig/CommitConfig

### 3. Lifecycle Management

firelynx uses the `go-supervisor` package for component lifecycle management and coordination. While `go-fsm` is used internally by `go-supervisor`, the focus is on the supervisor pattern for managing components. "Runnable" components in this system implement several interfaces from `go-supervisor`. 

### 4. Component Architecture

firelynx components are organized with clear separation of concerns:

#### Domain Config Layer (`internal/config/*`)

The domain config layer provides:

1. **Proto Conversion**: Transform serialized protocol buffer data into strongly-typed Go domain models
2. **Semantic Validation**: Verify relationships between resources and ensure referential integrity  
3. **Proto Serialization**: Transform domain models back to protocol buffers

#### Runtime Components (`internal/server/runnables/*`)

Runtime components implement the actual server functionality:

1. **Package Independence**: Each component defines its own config adapters
2. **Transaction Participation**: Components implement saga participant interfaces
3. **Lifecycle Management**: Components follow the go-supervisor lifecycle patterns

### 5. Hot Reload System

The hot reload system enables configuration updates with minimal service interruption through the transaction saga pattern:

1. **Configuration Sources**: File watcher and gRPC service create configuration transactions
2. **Transaction Validation**: Configurations are validated before processing
3. **Saga Orchestration**: Two-phase commit ensures atomic updates across all participants
4. **Component Coordination**: Participants stage and commit configuration changes atomically

Configuration updates require brief service interruption as HTTP listeners need to restart with new configurations.

For more details on the hot reload system, see [HOT_RELOAD.md](HOT_RELOAD.md).

### 6. Scripting System

The scripting system is powered by go-polyscript and supports multiple script engines:

1. **Script Compilation**: Scripts are compiled into executable units
2. **Static Data**: Configuration data is bundled with compiled scripts
3. **Runtime Execution**: Scripts execute with request-specific data
4. **Result Processing**: Script outputs are transformed into appropriate response formats

The supported engines include:
- **Risor**: Go-like scripting language for embedded scripts
- **Starlark**: Python-like configuration language 
- **Extism (WASM)**: WebAssembly plugin system supporting multiple languages
- **native**: Direct Go function registration for built-in functionality


## Application Startup Flow

The application uses `urfave/cli` for command-line parsing and go-supervisor for component lifecycle management:

1. **CLI Command**: Parses command-line arguments using urfave/cli
2. **Component Initialization**: Creates cfgfileloader, cfgservice, and txmgr components
3. **Supervisor**: Uses go-supervisor to manage component lifecycle
4. **Transaction Flow**: Components communicate via channels for configuration transactions
5. **Signal Handling**: Captures system signals and initiates graceful shutdown

## Configuration System

### Configuration Transaction Flow

firelynx uses a transaction-based configuration system that ensures atomic updates:

#### Configuration Sources

Configuration transactions are created from two sources:

1. **File-based Configuration (cfgfileloader)**:
   - Converts TOML to domain model and validates
   - Creates ConfigTransaction objects for valid configurations
   - Sends transactions to txmgr via channels

2. **gRPC Configuration Updates (cfgservice)**:
   - Receives Protocol Buffer configurations via gRPC
   - Converts to domain model and validates
   - Creates ConfigTransaction objects for valid configurations
   - Sends transactions to txmgr via channels

#### Transaction Processing

When a configuration transaction is received:

1. **Transaction Creation**: Source creates a ConfigTransaction with metadata (source, timestamp, ID)
2. **Validation**: Configuration is validated using domain model rules
3. **Transaction Management**: txmgr receives the validated transaction
4. **Saga Orchestration**: Saga orchestrator coordinates two-phase commit across participants
5. **Stage Phase**: Each participant (e.g., HTTP listener) stages the configuration
6. **Commit Phase**: If all participants succeed, changes are committed atomically

This architecture provides:
1. **Atomicity**: All configuration changes succeed or fail as a unit
2. **Consistency**: Participants never see partial configuration updates
3. **Auditability**: Complete transaction history with metadata
4. **Rollback**: Failed transactions can be compensated

### Domain Configuration Model

The domain model follows a consistent pattern for all configuration components:

1. **Collection Types**: Each component uses a collection type (e.g., `AppCollection`, `EndpointCollection`) that follows the "singular noun + Collection" naming convention.

2. **Interface-Based Design**: App configurations implement a common `AppConfig` interface, allowing different app types (Echo, Script, Composite) to be handled polymorphically.

3. **Package Structure**: Each major component has its own package with consistent file organization:
   - Core types and collections
   - Protocol buffer conversion
   - Validation logic
   - String/tree representation
   - Error handling

This design provides:
1. **Type Safety**: Strong typing for configuration elements
2. **Validation**: Domain-specific validation rules
3. **Maintainability**: Changes to the Protocol Buffer schema don't affect application code
4. **Idiomatic Go**: The domain model follows Go best practices
5. **Separation of Concerns**: Serialization logic is kept separate from business logic

## App Types

firelynx supports different types of applications:

1. **EchoApp**: Simple app that echoes back request information (currently implemented)
2. **ScriptApp**: App powered by script in one of several languages (structure implemented, echo handler used as placeholder)
   - **RisorScript**: Go-like scripting language
   - **StarlarkScript**: Python-like configuration language
   - **ExtismScript**: WebAssembly-based scripts
3. **CompositeScriptApp**: Chain of script apps executed in sequence (structure implemented, echo handler used as placeholder)

## App Registry and Instantiation

The app registry system manages the creation and lookup of app instances:

1. **Apps Factory**: Converts app configurations to running instances via `AppsToInstances`
2. **App Registry**: Stores app instances by ID for runtime access
3. **Route-to-App Mapping**: Maps HTTP routes to the appropriate app instances

This process ensures that:
- App configurations are validated before instantiation
- Apps are properly initialized with their configuration
- Routing can find the correct app instance for each request
- Different app types can be used interchangeably

## HTTP Server Implementation

The HTTP server implementation uses a composite runner pattern:

1. **HTTP Runner**: Manages the lifecycle of multiple HTTP listeners
2. **Composite Runner**: Dynamically adds and removes HTTP servers based on configuration
3. **HTTP Server**: Handles HTTP requests and routes them to the appropriate app
4. **RouteMapper**: Maps HTTP paths to application handlers

This architecture allows:
- Multiple HTTP listeners with different configurations
- Dynamic addition and removal of listeners at runtime
- Thread-safe configuration updates
- Clean lifecycle management with context cancellation

### Client-Server Communication

The configuration client communicates with the server via gRPC using the transaction system:

1. Client loads TOML configuration from disk
2. Client converts TOML to Protocol Buffer format
3. Client connects to server via gRPC
4. Client sends Protocol Buffer configuration in update request
5. Server creates ConfigTransaction and validates configuration
6. Server processes transaction through saga orchestrator
7. Server responds with transaction success/failure
8. Components are updated atomically through two-phase commit

This approach provides:
- Atomic configuration updates with rollback capability
- Strong validation before any changes are applied
- Complete audit trail of configuration changes
- Graceful error handling and recovery

## Logging Architecture

firelynx uses Go's `slog` package for structured logging with the following characteristics:

- Each component receives an slog.Handler for logging
- Multiple log streams are possible
- Log levels are configurable per component

## Error Handling

The error handling strategy is described in detail in [ERROR_HANDLING.md](ERROR_HANDLING.md).

Key aspects:

1. Script syntax errors are caught during validation
2. Runtime errors are handled appropriately
3. Protocol-specific error responses follow MCP format
4. Detailed error logging for diagnosis

## Extension Points

firelynx is designed to be extensible in several ways:

1. **New Script Engines**: Via go-polyscript
2. **New Application Types**: Beyond the built-in types
3. **New MCP Features**: As the MCP protocol evolves
4. **Custom Listeners**: Additional protocol support