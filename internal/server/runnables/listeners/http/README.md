# HTTP Listener

The HTTP listener package implements HTTP protocol support as a saga participant in the configuration transaction system.

## Purpose

The HTTP listener provides:

1. HTTP server lifecycle management
2. Request routing to configured applications
3. Saga participant implementation for configuration updates
4. Dynamic route and listener management

## Components

- **runner.go**: Main HTTP listener runnable implementing supervisor interfaces
- **saga.go**: Implements StageConfig/CommitConfig for transaction participation
- **state.go**: Manages listener state and configuration
- **cfg/**: Configuration adapters for converting domain config to HTTP-specific structures

## Saga Participation

As a saga participant, the HTTP listener:

1. **StageConfig**: Validates and prepares new HTTP configuration without applying
2. **CommitConfig**: Applies staged configuration, restarting HTTP servers as needed
3. **CompensateConfig**: Rolls back to previous configuration if needed

Configuration updates require brief service interruption as HTTP servers restart with new bindings.

## Integration

The HTTP listener registers with the saga orchestrator and participates in configuration transactions alongside other system components.