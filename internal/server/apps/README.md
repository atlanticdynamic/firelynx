# App Runtime Implementation

This package contains the runtime implementations of applications in the firelynx server.

## Architecture

This package follows the clean architecture pattern:

1. The `interfaces.go` file defines the core interfaces that applications must implement
2. The `implementations.go` file serves as the registry of all available app implementations
3. Individual app implementations are in subdirectories (e.g., `echo/`, `script/`)

## Built-in Apps

The server includes certain built-in apps that are always available:

- `echo`: A simple echo app that responds with information about the received request

These built-in apps are registered by the core adapter in `internal/server/core/built_in_apps.go`. 
This follows the proper architectural boundary where:

- Domain config layer (`internal/config/apps`) only validates configurations
- Core adapter layer (`internal/server/core`) instantiates apps
- Runtime layer (`internal/server/apps`) provides app implementations

## App Creation

Apps should be created using the factory methods in `implementations.go`. This provides a single
point of control for creating app instances of different types:

```go
// Example of creating an app using the factory
creator, exists := apps.AvailableAppImplementations["echo"]
if exists {
    app, err := creator(id, config)
    // Use the app
}
```

## Validation

The `GetAvailableAppTypes()` and `GetBuiltInAppIDs()` functions provide information about
available app types and built-in app IDs, which is used by the configuration layer for validation.
This allows validation without creating circular dependencies between packages.