# Server Applications

The `apps` package contains app implementations that process HTTP requests.

## Purpose

This package provides:

1. App implementations (Echo, Calculation, FileRead, Script, MCP gateway)
2. App interface definition
3. Map-based app storage by ID

## Key Files

- **app.go**: App interface definition
- **instances.go**: Map wrapper for storing app instances by ID
- **{type}/config.go**: Configuration structs for each app type
- **{type}/{type}.go**: App implementation

## App Interface

All apps implement two methods:
- `String()` - Returns app ID
- `HandleHTTP()` - Processes HTTP requests

**Data Flow**: Static data is embedded during app creation, not passed at runtime.

## App Types

**Currently implemented:**
- **Echo**: Returns request information for testing and debugging
- **Calculation**: Applies `+`, `-`, `*`, or `/` to `left` and `right` numeric inputs
- **FileRead**: Reads safe relative file paths from a configured base directory
- **Script**: Executes scripts using Risor, Starlark, or WebAssembly engines
- **MCP gateway**: Exposes app-backed tool providers over the Model Context Protocol

**Planned:**
- **Composite**: Chains multiple script apps in sequence

## Usage

Apps are created during configuration validation and stored in an `AppInstances` map. The HTTP layer looks up apps by ID and calls their `HandleHTTP()` method to process requests.
