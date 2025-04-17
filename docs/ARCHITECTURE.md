# firelynx Architecture

This document describes the architecture of the firelynx application, its components, and how they interact.

## Overview

firelynx (Model Context Protocol Application Layer) is a server application that implements the Model Context Protocol (MCP), enabling AI assistants to interact with custom tools, prompts, and resources. It is built with a modular design that emphasizes separation of concerns and enables hot-reloading of configurations.

## System Architecture

firelynx follows a client-server architecture where:

1. **firelynx Server**: Listens for connections from MCP clients and configuration updates
2. **MCP Clients**: AI applications (Claude, etc.) that connect to firelynx to access tools and resources
3. **Configuration Client**: Sends configuration updates to the firelynx server

```
┌────────────────┐    MCP Protocol    ┌───────────────┐
│                │◄──────────────────►│               │
│   MCP Client   │                    │  firelynx Server │
│  (e.g. Claude) │                    │               │
└────────────────┘                    │               │
                      gRPC Listener   │               │
┌────────────────┐   ( or protobuf )  │               │
│  Configuration │◄──────────────────►│               │
│     Client     │                    │               │
└────────────────┘                    └───────────────┘
```

## Core Components

The firelynx server consists of several core components:

### 1. Application Layers

firelynx follows a three-layer architecture:

```
┌─────────────────────────────────────────┐
│              Listeners                  │
│  (MCP, HTTP/REST, gRPC, Unix Socket)    │
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
│  (Script Apps, MCP Implementations)     │
└─────────────────────────────────────────┘
```

- **Listeners**: Protocol-specific server components that accept connections
- **Endpoints**: Map incoming requests to the appropriate application
- **Applications**: Implement functionality (scripts, MCP features)

### 2. State Management

firelynx uses the `go-fsm` library for state management and lifecycle control:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    FSM State    │    │ State Transition│    │  State Change   │
│    Machine      │───►│     Logic       │───►│  Notifications  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                                             │
        ▼                                             ▼
┌─────────────────┐                        ┌─────────────────┐
│ Component State │                        │ Client Response │
│  Management     │                        │    Handling     │
└─────────────────┘                        └─────────────────┘
```

The standard server states are:
- **New**: Initial state after creation
- **Booting**: During startup initialization
- **Running**: Normal operation
- **Reloading**: During configuration update
- **Stopping**: During graceful shutdown
- **Stopped**: After shutdown completion
- **Error**: Error condition
- **Unknown**: Unrecoverable state

### 3. Hot Reload System

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

The application uses `urfave/cli` for command-line parsing, `go-supervisor` for service lifecycle management, and `go-fsm` for state tracking:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   urfave/cli    │    │ Configuration   │    │ go-fsm          │
│   Command       │───►│   Loader        │───►│ State Machine   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                                                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  go-supervisor  │◄───│  Server Manager │◄───│  Component      │
│                 │    │                 │    │  Initializer    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

1. **CLI Command**: Parses command-line arguments using urfave/cli
2. **Configuration Loader**: Loads and validates TOML configuration 
3. **State Machine**: Initializes the FSM for state tracking
4. **Component Initializer**: Sets up listeners, endpoints, and applications
5. **Server Manager**: Coordinates server components
6. **Supervisor**: Manages lifecycle and handles signals

## Configuration Format

firelynx uses TOML for human-readable configuration, and Protocol Buffers for in-memory and over-the-wire representation:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│      TOML       │    │ Config Marshaler│    │ Protocol Buffer │
│  Config File    │───►│                 │───►│   Objects       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

The marshaler translates between the TOML format and protocol buffer objects.

## Script Application Types

firelynx supports different types of script applications:

1. **ScriptApp**: Single script executed for each request
2. **CompositeScriptApp**: Chain of scripts executed in sequence
3. **McpApp**: Specialized script apps implementing MCP features (tools, prompts, resources)

## MCP Implementation Details

### Tools Implementation

MCP tools are implemented as script applications that:

1. Receive parameters as input
2. Process the parameters using scripts
3. Return results in the MCP tool response format

Each tool provides:
- Name and description
- Parameter schema (JSON Schema)
- Script implementation
- Optional static data

### Prompts Implementation

MCP prompts are implemented as script applications that:

1. Receive argument values
2. Process the arguments to generate a prompt
3. Return the formatted prompt template

Each prompt provides:
- Name and description
- Argument definitions
- Script implementation
- Optional static data

## Client-Server Communication

### MCP Protocol Communication

- Client connects to firelynx server using WebSocket/HTTP
- Client sends MCP protocol requests (tool calls, prompt requests)
- Server processes requests through the appropriate script application
- Server returns responses formatted according to MCP specification

### Configuration Updates

- Configuration client connects to firelynx server via gRPC
- Client sends new configuration
- Server validates and applies the configuration
- Server responds with status of update

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