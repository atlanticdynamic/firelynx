# Server Applications

The `apps` package manages application instances and provides the runtime interface between HTTP requests and configured applications.

## Purpose

This package handles:

1. Application instantiation from domain configurations
2. Runtime app registry for request routing
3. App lifecycle management
4. Request processing interface

## Components

- **instances.go**: Registry for managing app instances by ID
- **app.go**: Common App interface definition

## App Interface

Apps implement a simple interface:
- `String()` - Returns the app ID for registry lookup
- `HandleHTTP(ctx, ResponseWriter, *Request)` - Processes HTTP requests

Static data is embedded during app creation, not passed at runtime.

## App Types

Currently implemented:
- **EchoApp**: Returns request information for testing
- **ScriptApp**: Executes scripts (Risor, Starlark, WebAssembly)
- **MCPApp**: Executes Model Context Protocol tools

Future implementations:
- **CompositeApp**: Chains multiple script apps

## Integration

The app registry routes HTTP requests to configured application instances based on path mappings.

## Architecture

- **DTO Pattern**: Domain configs converted to standalone Config structs via transaction layer
- **Direct Instantiation**: Clean instantiator functions create app instances from DTOs
- **Server Registry**: Simple map-based registry (`AppInstances`) for runtime lookup
- **Iterator Support**: Both domain and server layers use Go 1.23 `All()` methods for clean iteration