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

### 2. Lifecycle Management

firelynx uses the `go-supervisor` package for component lifecycle management and coordination. While `go-fsm` is used internally by `go-supervisor`, the focus is on the supervisor pattern for managing components.

Components in the system implement standard lifecycle interfaces:

1. **Runnable**: Components with a `Run(ctx)` method that runs until the context is canceled
2. **Reloadable**: Components with a `Reload()` method that can update their configuration
3. **Named**: Components with a `String()` method that provides a unique identifier

The standard component lifecycle states are:
- **New**: Initial state after creation
- **Running**: Normal operation
- **Reloading**: During configuration update
- **Stopping**: During graceful shutdown
- **Stopped**: After shutdown completion
- **Error**: Error condition

### 3. Architectural Layers

Firelynx follows a strict separation of concerns across three distinct layers to ensure maintainability and flexibility:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐      ┌─────────────────┐
│                 │     │                 │     │                 │      │                 │
│     Protobuf    │◄───►│   Domain Config │◄───►│   Core Adapter  │◄────►│ Runtime         │
│     Schema      │     │      Layer      │     │      Layer      │      │ Components      │
│                 │     │                 │     │                 │      │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘      └─────────────────┘
                      internal/config       internal/server/core      internal/server/*
```

#### Domain Config Layer (`internal/config/*`)

The domain config layer has three specific responsibilities:

1. **Proto Conversion**: Transform serialized protocol buffer data into strongly-typed Go domain models
2. **Semantic Validation**: Verify relationships between resources, check valid app names and ensure referential integrity
3. **Proto Serialization**: Transform domain models back to protocol buffers

**Important Boundaries**: This layer does NOT handle:
- Instantiation of runtime components or app instances
- Execution of any business logic
- Runtime request routing or handling

#### Core Adapter Layer (`internal/server/core/*`)

This layer serves as the only bridge between domain config and runtime components:

1. **Domain Config Access**: The only component with direct imports from `internal/config`
2. **Type Conversion**: Converts domain config types to package-specific configs
3. **Configuration Callbacks**: Provides configuration through callbacks to runtime components

#### Runtime Components (`internal/server/*` except `core`)

These implement the actual server functionality with these characteristics:

1. **Package Independence**: Each component defines its own config types with no dependency on domain config
2. **Callback-Based Configuration**: Receives configuration via callbacks, not direct dependencies
3. **Lazy Configuration**: Loads configuration during Run(), not during initialization
4. **Lifecycle Adherence**: Follows the standard supervisor lifecycle (Run, Stop, Reload)

### 4. Hot Reload System

The hot reload system enables configuration updates without server downtime:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Configuration  │    │   Validation    │    │   Component     │
│    Receiver     │───►│     Engine      │───►│   Orchestrator  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                                                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Traffic     │◄───│    Component    │◄───│    Component    │
│    Switcher     │    │     Registry    │    │     Factory     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

1. **Configuration Receiver**: gRPC endpoint for receiving configuration updates via protobufs objects
2. **Validation Engine**: Validates new configurations (each app is loaded, and it's validation is called)
3. **Component Orchestrator**: Manages the transition to the new configuration (using go-supervisor reload)
4. **Component Factory**: Creates listener, endpoint, and application instances (listener changes require traffic interuption, but endpoint or app changes can be "hot")
5. **Component Registry**: Maintains references to active components
6. **Traffic Switcher**: Controls request flow during reloading

For more details on the hot reload system, see [HOT_RELOAD.md](HOT_RELOAD.md).

### 4. Scripting System

The scripting system is powered by go-polyscript:

```
┌─────────────────┐                           ┌─────────────────┐
│  Script Compile │                           │  Runtime Data   │
└─────────────────┘                           └─────────────────┘
        │                                              │
        ▼                                              ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Script         │    │  go-polyscript  │    │     Script      │
│  ExecutableUnit │───►│    Evaluator    │───►│     Runner      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        ▲                                              │
        │                                              ▼
┌─────────────────┐                           ┌─────────────────┐
│  Static Data    │                           │  Result Handler │
└─────────────────┘                           └─────────────────┘
```

1. **go-polyscript Compiler**: go-polyscript loads/compiles the script into runnable bytecode, produces an ExecutableUnit
2. **go-polyscript ExecutableUnit**: go-polyscript runnable object, used for creating the Evaluator
3. **Static Data**: Compile-time static data bundled into the ExecutableUnit, such as config
4. **go-polyscript Evaluator**: Executes scripts using appropriate engine (Risor, Starlark, Extism (WASM), or native)
5. **firelynx Script Runner**: Manages script execution lifecycle
6. **Runtime Data**: Each invocation can supply additional runtime data
7. **Result Handler**: Processes and transforms script results

The supported engines include:
- **Risor**: Go-like scripting language for embedded scripts
- **Starlark**: Python-like configuration language 
- **Extism (WASM)**: WebAssembly plugin system supporting multiple languages
- **native**: Direct Go function registration for built-in functionality

### 5. MCP Implementation

The MCP protocol implementation provides the standardized interface:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  MCP Transport  │    │   MCP Request   │    │  MCP Response   │
│     Layer       │───►│    Handler      │───►│    Formatter    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  MCP Feature    │
                      │  Implementations│
                      └─────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  Script-based   │
                      │  MCP Handlers   │
                      └─────────────────┘
```

The MCP implementation is based on the [mcp-go](https://github.com/mark3labs/mcp-go) library, which provides the core MCP protocol support.

## Application Startup Flow

The application uses `urfave/cli` for command-line parsing and context-based coordination for component lifecycle management:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   urfave/cli    │    │ Configuration   │    │   Context &     │
│   Command       │───►│   Manager       │───►│   Goroutines    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                                                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Signal         │────│  Component      │◄───│  Reload         │
│  Handling       │    │  Coordination   │    │  Channels       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

1. **CLI Command**: Parses command-line arguments using urfave/cli
2. **Configuration Manager**: Loads and manages TOML configuration through gRPC
3. **Context & Goroutines**: Uses context cancellation for coordinating component lifecycle
4. **Reload Channels**: Components communicate via channels for configuration updates
5. **Component Coordination**: Coordinates the initialization and shutdown of components
6. **Signal Handling**: Captures system signals and initiates graceful shutdown

## Configuration System

### Configuration Format and Flow

firelynx uses TOML for human-readable configuration and Protocol Buffers for in-memory storage and network transmission. The configuration system follows a domain-driven design approach with several key flows:

#### Initial Loading Flow

When the server starts and loads configuration from a file:

1. **TOML → Protocol Buffers**:
   - `loader.NewLoaderFromFilePath()` gets a loader for the TOML file
   - `loader.LoadProto()` parses TOML and converts to Protocol Buffer objects

2. **Protocol Buffers → Domain Model (for validation)**:
   - `config.NewFromProto()` converts Protocol Buffers to Domain Model
   - `config.Validate()` validates the domain model

3. **Domain Model → Protocol Buffers (for storage)**:
   - After validation, `domainConfig.ToProto()` converts back to Protocol Buffers
   - The Protocol Buffer version is stored internally: `r.config = protoConfig`

#### Access for Processing

When components need configuration for processing:

1. **Protocol Buffers → Domain Model**:
   - `GetPbConfigClone()` gets a copy of the stored Protocol Buffer
   - `config.NewFromProto()` converts to domain model for processing

2. **Domain Model → Service-Specific Format**:
   - For HTTP components, `GetHTTPConfigCallback()` transforms domain model to HTTP config
   - Each service gets a configuration format tailored to its needs

#### Configuration Update Flow via gRPC

When a client updates configuration via gRPC:

1. **Protocol Buffers (wire) → Domain Model (validation)**:
   - Client sends Protocol Buffer objects over gRPC
   - Server converts to domain model: `config.NewFromProto(req.Config)`
   - Validates with `domainConfig.Validate()`

2. **Protocol Buffers (storage)**:
   - Original Protocol Buffer is stored: `r.config = req.Config`
   - Reload notification is sent to components

This architecture provides:
1. **Type Safety**: Strong typing for configuration elements
2. **Validation**: Domain-specific validation rules
3. **Maintainability**: Changes to the Protocol Buffer schema don't affect application code
4. **Idiomatic Go**: The domain model follows Go best practices

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

The configuration client communicates with the server via gRPC:

1. Client loads TOML configuration from disk
2. Client converts TOML to Protocol Buffer format (loader.LoadProto)
3. Client connects to server via gRPC
4. Client sends Protocol Buffer configuration in update request
5. Server validates configuration (by converting to domain model)
6. Server stores the validated Protocol Buffer configuration
7. Server sends reload notification to components via channels
8. Components reload with the new configuration

This approach allows for:
- Clear separation between client and server
- Strong validation before configuration is applied
- Asynchronous notification of configuration changes
- Graceful handling of configuration updates

## Logging Architecture

firelynx uses Go's `slog` package for structured logging:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Root Logger   │───►│ Component Logger│───►│   Log Handler   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

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