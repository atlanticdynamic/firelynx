# HTTP Listeners Package

This package implements HTTP server functionality for Firelynx, providing a flexible system for creating, managing, and routing HTTP requests to application handlers.

## Architecture and Design Principles

1. **Lifecycle Management**: Components follow the standard Run/Stop/Reload lifecycle defined by the supervisor framework
2. **Lazy Configuration**: Configuration loading occurs during the Run phase, not during initialization
3. **Dynamic Reconfiguration**: Supports configuration changes without restarting the entire system
4. **Separation of Concerns**: Each component has a well-defined responsibility within the HTTP server lifecycle
5. **Registry Integration**: Application routing is provided through the registry in the configuration

## Components

### Runner (`runner.go`)

The core component that manages the lifecycle of multiple HTTP listeners. It:

- Implements the `supervisor.Runnable` and `supervisor.Reloadable` interfaces
- Creates and manages HTTP listeners based on configuration obtained during Run()
- Uses a composite runner pattern to manage multiple independent HTTP servers
- Handles validation and error reporting
- Manages reloads gracefully when configuration changes

### Listener (`listener.go`)

Represents a single HTTP server instance bound to a specific address. It:

- Wraps the standard library's `http.Server`
- Provides consistent lifecycle management (start, stop, graceful shutdown)
- Configures server timeouts based on configuration
- Implements graceful shutdown with configurable drain timeout
- Streamlined error handling during shutdown

### RouteMapper (`routermapper.go`)

Maps configuration endpoints to HTTP routes. It:

- Converts domain configuration objects into HTTP routes
- Filters routes based on listener ID
- Maps HTTP path conditions to application handlers
- Supports HTTP-specific routing conditions

### Config (`config.go`)

Defines the configuration structure for HTTP listeners with:

- Registry for application dispatch
- Listener configurations (address, timeouts)
- Route configurations (paths and application mappings)
- Default timeout values

## Route Handling

The system implements a path-based routing system where:
1. Routes are defined in the configuration
2. Each route is associated with an application ID and optional static data
3. Routes are mapped to specific listeners using listener IDs
4. Requests are forwarded to the appropriate application handler