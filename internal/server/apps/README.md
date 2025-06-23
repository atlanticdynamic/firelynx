# Server Applications

The `apps` package manages application instances and provides the runtime interface between HTTP requests and configured applications.

## Purpose

This package handles:

1. Application instantiation from domain configurations
2. Runtime app registry for request routing
3. App lifecycle management
4. Request processing interface

## Components

- **factory.go**: Creates app instances from domain configurations
- **instances.go**: Registry for managing app instances by ID
- **app.go**: Common App interface definition
- **instantiators.go**: Type-specific app creation logic
- **echo/**: Echo app implementation

## App Types

Currently implemented:
- **EchoApp**: Returns request information for testing

Future implementations:
- **ScriptApp**: Executes scripts (Risor, Starlark, WebAssembly)
- **CompositeApp**: Chains multiple script apps

## Integration

The app registry is used by HTTP listeners to route requests to the appropriate app instance based on configured routes.