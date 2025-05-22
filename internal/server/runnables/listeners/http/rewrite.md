# HTTP Listener Rewrite Plan

## Overview

This document outlines the plan for rewriting the HTTP listener to implement the Config Saga Pattern. The new implementation will make the HTTP listener a participant in the configuration saga, enabling atomic updates and rollbacks across system components.

## Requirements

1. HTTP Runner must implement `txmgr.SagaParticipant` interface (which includes `supervisor.Runnable` and `supervisor.Stateable`)
2. Implementation must NOT implement the `supervisor.Reloadable` interface, as we want the saga orchestrator to manage reloads
3. Must use `composite.Runner` from go-supervisor as a framework basis
4. Must support current and pending configurations for saga prepare/commit/rollback phases
5. Avoid import cycles between packages
6. Support creation without initial configuration (provided later via transactions)
7. Ensure child `HTTPServer` objects don't implement `composite.ReloadableWithConfig` to prevent the parent composite runner from reloading them directly

## Implementation Status

### Completed Components

1. ✅ **ConfigManager (cfg/manager.go)**: 
   - Thread-safe storage for current and pending configurations
   - Support for setting, committing, and rolling back pending configs
   - Unit tests for all methods
   
2. ✅ **Adapter (cfg/adapter.go)**: 
   - Extracts HTTP-specific configuration from transactions
   - Converts domain model endpoints and listeners to HTTP routes
   - Leverages domain model's GetStructuredHTTPRoutes for better integration
   - Unit tests for configuration extraction

3. ✅ **HTTPServer (httpserver/server.go)**:
   - Thread-safe wrapper around go-supervisor's httpserver.Runner
   - Deliberately does NOT implement ReloadableWithConfig or Reloadable
   - Implements serverImplementation interface for better testability
   - All state access properly protected with mutex locks
   - Unit tests for creation and state management

4. ✅ **Runner (runner.go)**:
   - Implements the SagaParticipant interface from txmgr package
   - Uses composite.Runner to manage multiple HTTP servers
   - Handles prepare, commit, and rollback phases properly
   - Base unit tests for creation and state management

### Pending Tasks

1. 🔄 **Integration Tests**:
   - Test HTTP runner with saga orchestrator
   - Test configuration transactions with realistic configurations
   - Test multiple HTTP listeners with different configurations
   - Test rollback scenarios

2. 🔄 **Eliminate RouteRegistry Dependency**:
   - Remove dependency on the separate routing registry component
   - Create routes as immutable data objects during configuration processing
   - Directly link routes to app instances in the HTTP adapter
   - Use the stdlib mux router instead of a custom implementation

3. 🔄 **RequestHandler Implementation**:
   - Replace dummy handler with proper request routing
   - Use the standard library mux router for path handling

4. 🔄 **Documentation**:
   - Add documentation for how to use the HTTP listener with the saga pattern
   - Include examples of HTTP configuration
   - Document the simplified routing approach

## Component Design

### Main Components

1. **HTTPRunner**: Implements `txmgr.SagaParticipant`, manages HTTP servers via the composite runner
2. **ConfigManager**: Thread-safe storage for current and pending configurations
3. **Adapter**: Extracts HTTP-specific configuration from transactions
4. **HTTPServer**: Wrapper around go-supervisor's httpserver.Runner (must not implement ReloadableWithConfig)

### Component Relationships

```
HTTPRunner (SagaParticipant)
  ├── ConfigManager
  │     ├── Current Adapter
  │     └── Pending Adapter
  └── composite.Runner[*HTTPServer]
        └── Multiple HTTPServer instances
```

## Key Implementation Improvements

1. **Clean Separation of Concerns**:
   - Configuration management in cfg package
   - HTTP server implementation in httpserver package
   - SagaParticipant implementation in main http package
   
2. **Domain Model Integration**:
   - Using GetStructuredHTTPRoutes from endpoints package
   - Direct access to GetEndpointsForListener for efficient route extraction

3. **Thread Safety**:
   - All shared state protected with appropriate mutex locks
   - Config updates properly managed through prepare/commit/rollback cycle
   
4. **Testability**:
   - Components designed with testing in mind
   - Mock interfaces for easier unit testing
   - Abstraction of server implementation for better test coverage

## Simplified Routing Approach

Instead of using a separate routing registry component with its own lifecycle and state management, we're simplifying the routing system with the following approach:

1. **Immutable Route Objects**:
   - Routes are created as immutable data objects during configuration processing
   - All route information, including path patterns and associated app instances, is determined at config time
   - No need for separate route resolution at request time

2. **Direct App Instance Linking**:
   - The HTTP adapter directly links routes to app instances during config extraction
   - This eliminates the need for a separate registry lookup at request time
   - App instances are available directly in the route objects

3. **Standard Library Mux Router**:
   - Leverages the Go standard library's HTTP router (mux) 
   - No need to implement a custom router with its own path matching logic
   - Simpler, more maintainable, and better tested implementation

4. **Thread-Safe Configuration Swapping**:
   - Route configuration is swapped atomically during transaction commit
   - No partial updates or inconsistent routing state
   - Fully thread-safe for concurrent request handling

This simplification removes an entire layer of indirection, making the system easier to understand and maintain, while also improving performance by eliminating registry lookups during request handling.

## Next Steps

1. Complete the integration tests
2. Implement simplified routing with stdlib mux and direct app instance linking
3. Document the new pattern for HTTP configuration
4. Integrate with the saga orchestrator in the main server

## Key Points

1. **No Reloadable Interface**: The runner does not implement supervisor.Reloadable to ensure only the saga transaction can trigger reloads
2. **No ReloadableWithConfig**: The HTTPServer does not implement ReloadableWithConfig to prevent direct reloads from the composite runner
3. **Composite Runner**: Uses composite.Runner from go-supervisor as a framework, but manages reload through the saga pattern
4. **Thread Safety**: All operations properly handle concurrency with mutex locks
5. **Clear Component Boundaries**: Each component has a specific responsibility
6. **FSM Integration**: Fully integrated with the saga state machine through the SagaParticipant interface
7. **Direct Route-to-App Mapping**: Routes are created with direct links to app instances, eliminating routing registry dependency