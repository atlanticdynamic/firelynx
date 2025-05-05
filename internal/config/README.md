# Internal Config Package (`internal/config`)

This package defines the domain model for the Firelynx server configuration and provides utilities for loading, validating, and converting configuration data.

## Package Structure

The config package has been organized into modular components following a consistent pattern:

* `internal/config/`: Top-level domain types and orchestration
* `internal/config/apps/`: Application definition types and operations
* `internal/config/endpoints/`: Endpoint and route types and operations
* `internal/config/listeners/`: Network listener types and operations
* `internal/config/logs/`: Logging configuration types and operations
* `internal/config/loader/`: Configuration loading from different formats/sources
* `internal/config/errz/`: Common error types and handling
* Using `github.com/robbyt/protobaggins` for protocol buffer conversions

## Common File Patterns

Each sub-package follows a consistent file organization pattern:

1. **Core Type Definitions**:
   - `apps/apps.go`, `endpoints/endpoints.go`, `listeners/listeners.go`, `logs/logs.go`
   - Primary domain model types and interfaces
   - Type-specific operations and methods

2. **Protocol Buffer Conversion**:
   - `apps/proto.go`, `endpoints/proto.go`, `listeners/proto.go`
   - Bidirectional conversion between domain models and protobuf
   - Both standalone functions and receiver methods

3. **Validation**:
   - `apps/validate.go`, `endpoints/validate.go`, `listeners/validate.go`, `logs/validate.go`
   - Type-specific validation rules and logic
   - Implementation of the `Validate()` interface

4. **String/Tree Representation**:
   - `apps/string.go`, `endpoints/string.go`, `listeners/string.go`, `logs/string.go`
   - Human-readable string representations
   - Tree visualization for CLI and debugging output

5. **Error Definitions**:
   - `apps/errors.go`, `endpoints/errors.go`, `listeners/errors.go`
   - Package-specific error constants and types
   - Error handling utilities

6. **Tests**:
   - Unit tests for all functionality
   - Standard Go test naming (`*_test.go`)

## Core Files and Their Purposes

* **Domain Model (`config.go`)**: Defines the core configuration structures used throughout the application. Provides a type-safe way to work with configuration within the application code. Contains the main `Config` struct that orchestrates and references the domain models from sub-packages.

* **Initialization (`config.go`)**: Contains functions to create and initialize configuration objects from various sources (files, bytes, readers). Handles the initial conversion from protobuf to domain model via `NewFromProto`. Functions like `NewConfig`, `NewConfigFromBytes` and `NewConfigFromReader` orchestrate the full load-convert-validate workflow.

* **Proto Conversion (`proto.go`)**: Handles bidirectional conversion between domain models and protobuf representations for the top-level Config type. Each sub-package contains its own proto conversion for its specific types.

* **Validation (`validate.go`)**: Performs comprehensive validation of configurations, ensuring integrity and consistency across all components. Delegates to sub-package validation for type-specific validation.

* **Querying (`query.go`)**: Provides helper functions to extract specific subsets of configuration. Enables filtering by various criteria and retrieving related objects.

* **String Representation (`string.go`)**: Creates human-readable console output for configurations using the tree structure to visualize relationships between components.

* **Errors (`errors.go`)**: Defines sentinel errors used throughout the package. Standardizes error types for consistent handling.

## Core Concepts

* **Domain Model First**: The package is designed to work with Go domain models internally while using protobuf for serialization and transport.

* **Validation Chain**: Configuration goes through a multi-step process: loading → conversion → validation.

* **Protocol Buffers**: The canonical schema is defined using Protocol Buffers, allowing for language-agnostic definitions and interoperability.

* **Type Safety**: The domain model provides type safety features not available in the protobuf-generated code.

* **Modular Organization**: Each major configuration component has its own package with consistent file organization.

* **Hierarchical Design**: The configuration follows a clear hierarchy (Config → Listeners/Endpoints/Apps → Routes → Conditions) that is reflected in the query methods.

## Usage

### Configuration Relationships

The domain model contains several key relationships:

1. **Listeners and Endpoints**:
   - Endpoints reference one or more Listeners through the `ListenerIDs` field
   - Each Listener can be referenced by multiple Endpoints
   - This many-to-many relationship allows flexible protocol mappings

2. **Endpoints and Routes**:
   - Each Endpoint contains multiple Routes
   - Routes define conditions for matching requests (HTTP path, gRPC service)
   - Each Route contains a reference to an App via `AppID`

3. **Routes and Apps**:
   - Routes reference exactly one App through the `AppID` field
   - Each App can be referenced by multiple Routes
   - This allows the same App to be accessible through different routes

4. **CompositeScriptApp and ScriptApps**:
   - CompositeScriptApp references one or more ScriptApps through `ScriptAppIDs`
   - This allows composing multiple script apps into a single logical app

### Loading Configuration

The primary way to load and obtain a validated domain `Config` object is through the functions provided in `config.go`:

1. **Choose a source:**
   * From a file path: `config.NewConfig(filePath string)`
   * From raw bytes: `config.NewConfigFromBytes(data []byte)`
   * From an `io.Reader`: `config.NewConfigFromReader(reader io.Reader)`

2. **Process:** These functions perform the following steps internally:
   * Use the appropriate `loader` function to get a `loader.Loader`
   * Parse the source into a protobuf object
   * Convert the protobuf to the domain model using `NewFromProto`
   * Validate the domain config with `Validate()`

3. **Result:** If all steps succeed, you receive a valid `*Config` object. Otherwise, an error is returned.

```go
package main

import (
    "log"

    "github.com/atlanticdynamic/firelynx/internal/config"
)

func main() {
    filePath := "path/to/your/config.toml"
    cfg, err := config.NewConfig(filePath)
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Use the validated cfg object
    log.Printf("Loaded config version: %s", cfg.Version)
    
    // Access apps
    risorApps := cfg.Apps.FindByType("risor")
    for _, app := range risorApps {
        log.Printf("Found Risor app: %s", app.ID)
    }
    
    // Find app by ID
    app := cfg.Apps.FindByID("myapp")
    if app != nil {
        log.Printf("Found app: %s", app.ID)
    }
    
    // ... access other parts of cfg ...
}
```

### Converting Domain `Config` to Protobuf

If you have a domain `*Config` object and need its Protobuf representation, use the `ToProto()` method:

```go
// Assume 'cfg' is a valid *config.Config object
pbConfig := cfg.ToProto()

// Now pbConfig is a *pbSettings.ServerConfig object
// You can use it for serialization, sending over network, etc.
```

## Configuration Domain Model

```
Config
├── Logging (logs.Config)
├── Listeners (listeners.Listeners)
│   ├── ID
│   ├── Address
│   ├── Type (HTTP/gRPC)
│   └── Options (HTTPOptions/GRPCOptions)
├── Endpoints (endpoints.Endpoints)
│   ├── ID
│   ├── ListenerIDs
│   └── Routes
│       ├── AppID
│       ├── StaticData
│       └── Condition (HTTPPathCondition/GRPCServiceCondition)
└── Apps (apps.Apps)
    ├── ID
    └── Config
        ├── ScriptApp
        │   ├── StaticData
        │   └── Evaluator (Risor/Starlark/Extism)
        └── CompositeScriptApp
            ├── ScriptAppIDs
            └── StaticData
```

The configuration model follows a clear structure:
- Top level Config contains Logging, Listeners, Endpoints, and Apps collections
- Endpoints reference Listeners via ListenerIDs
- Routes within Endpoints reference Apps via AppID
- Routes use different Condition types to determine routing rules
- Apps can contain different configurations (ScriptApp, CompositeScriptApp)
- Apps with CompositeScriptApp configuration reference other apps via ScriptAppIDs

## Component-Specific Sub-Packages

### Apps Package (`internal/config/apps`)

The apps package encapsulates all application-related configuration:

* **App Types (`apps.go`)**: Defines app-specific types like `App`, `ScriptApp`, and various evaluator types (`RisorEvaluator`, `StarlarkEvaluator`, `ExtismEvaluator`).
* **Proto Conversion (`proto.go`)**: Converts between domain App types and protobuf representations.
* **Validation (`validate.go`)**: Contains app-specific validation logic, including reference validation between composite apps.
* **String Representation (`string.go`)**: Human-readable output and tree visualization.
* **Collection Operations**: Provides methods like `FindByID` and `FindByType` for working with collections of apps.
* **Evaluator Type Interface**: Defines a common interface for different script evaluation engines.

### Endpoints Package (`internal/config/endpoints`)

The endpoints package handles routing and protocol-independent endpoint configuration:

* **Endpoint Types (`endpoints.go`)**: Defines the `Endpoint` and `Route` types.
* **Proto Conversion (`proto.go`)**: Handles conversion to and from protobuf.
* **Validation (`validate.go`)**: Validates endpoint and route configuration.
* **String Representation (`string.go`)**: Creates human-readable output for endpoints.
* **Route Conditions**: Implements different matching conditions (`HTTPPathCondition`, `GRPCServiceCondition`).
* **Structured Routes**: Provides type-safe HTTP and gRPC route representations.

### Listeners Package (`internal/config/listeners`)

The listeners package manages network binding and protocol-specific options:

* **Listener Types (`listeners.go`)**: Defines the `Listener` type and protocol options.
* **Proto Conversion (`proto.go`)**: Converts between domain and protobuf representations.
* **Validation (`validate.go`)**: Ensures listener configuration is valid.
* **String Representation (`string.go`)**: Creates human-readable output for listeners.
* **Protocol Options**: Type-safe options for HTTP and gRPC protocols.

### Logs Package (`internal/config/logs`)

The logs package handles logging configuration:

* **Log Types (`logs.go`)**: Defines types for log format and level.
* **Proto Conversion (in `logs.go`)**: Converts between domain and protobuf.
* **Validation (`validate.go`)**: Validates logging configuration.
* **String Representation (`string.go`)**: Creates human-readable output.

### Common Patterns

Within each sub-package:

1. **Collection Types**: Each component has a plural collection type (`Apps`, `Endpoints`, `Listeners`) that provides collection-level operations.
2. **Domain Methods**: Domain objects implement methods for common operations like validation, conversion, and string representation.
3. **Type Safety**: Enums are implemented as typed constants with validation methods.
4. **Error Handling**: Each package defines its own error types and validation logic.
5. **Testing**: Comprehensive tests for all functionality.