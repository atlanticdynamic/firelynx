# Internal Config Package (`internal/config`)

This package defines the domain model for the Firelynx server configuration and provides utilities for loading, validating, and converting configuration data.

## Core Purpose

The domain configuration layer serves as the bridge between the protobuf layer and the rest of the application. It provides three functions:

1. **Convert from proto to domain config**: Transform serialized protocol buffer data into strongly-typed Go domain models
2. **Semantic validation**: Verify relationships beyond TOML syntax - confirming apps exist, listener IDs are mapped to endpoints, route conditions are valid, and maintaining referential integrity across the configuration graph
3. **Convert back to proto**: Transform domain models back to protocol buffers for serialization

The semantic validation ensures configuration consistency that cannot be enforced at the TOML or protobuf schema level, such as verifying that all referenced app IDs actually exist in the apps collection.

### Important Boundaries

The domain config layer does **not** handle:
- Instantiation of runtime components or app instances
- Execution of any business logic
- Runtime request routing or handling

This clear separation ensures the domain config remains a pure data model with validation. Runtime execution is the responsibility of the `internal/server` packages.

## Architectural Patterns

### Clean Separation via Adapters

The Firelynx architecture maintains strict separation between layers using an adapter pattern:

```
protobuf ↔ domain config ↔ core adapter ↔ package-specific configs
```

1. **Domain Config Layer** (`internal/config/*`):
   - Converts proto ↔ domain model
   - Validates domain model
   - Has no knowledge of runtime components

2. **Core Adapter Layer** (`internal/server/core/*`):
   - Only place that accesses domain config types
   - Converts domain config to package-specific configs
   - Provides configuration callbacks to runnables

3. **Runtime Components** (`internal/server/*` except `core`):
   - Define their own package-specific configs
   - Have no direct dependencies on domain config
   - Receive config through callbacks

This design ensures that if domain config structure changes, only the core adapter layer needs updating, not all runtime components.

### Callback Pattern for Configuration

Runtime components receive configuration through callbacks rather than direct dependencies:

```go
// Package-specific config callback (no domain config knowledge)
type ConfigCallback func() (*MyPackageConfig, error)

// Runtime component using the callback
func NewComponent(configCallback ConfigCallback) *Component {
    return &Component{configCallback: configCallback}
}

// Called during Run() or Reload(), not during initialization
func (c *Component) Run(ctx context.Context) error {
    config, err := c.configCallback()
    if err != nil {
        return err
    }
    // Use config...
}
```

This pattern:
- Avoids premature configuration loading
- Follows the supervisor lifecycle (Run, Stop, Reload)
- Maintains clean separation between configuration and runtime

## Package Structure

The config package has been organized into modular components following a consistent pattern:

* `internal/config/`: Top-level domain types and orchestration
* `internal/config/apps/`: Application definition types and operations
  * `internal/config/apps/echo/`: Echo app configuration
  * `internal/config/apps/scripts/`: Script app configuration
    * `internal/config/apps/scripts/evaluators/`: Script evaluator implementations (Risor, Starlark, Extism)
  * `internal/config/apps/composite/`: Composite script app configuration that combines multiple script apps
* `internal/config/endpoints/`: Endpoint and route types and operations
  * `internal/config/endpoints/routes/`: Route definitions and conditions
    * `internal/config/endpoints/routes/conditions/`: Route matching conditions (HTTP, gRPC)
* `internal/config/listeners/`: Network listener types and operations
  * `internal/config/listeners/options/`: Protocol-specific listener options
* `internal/config/logs/`: Logging configuration types and operations
* `internal/config/loader/`: Configuration loading from different formats/sources
* `internal/config/staticdata/`: Shared static data types and operations
* `internal/config/errz/`: Common error types and handling
* `internal/config/styles/`: Formatting and display styles for components

## Common File Patterns

Each sub-package follows a consistent file organization pattern:

1. **Core Type Definitions**:
   - `apps/apps.go`, `endpoints/endpoints.go`, `listeners/listeners.go`, `logs/logs.go`
   - Primary domain model types and interfaces
   - Type-specific operations and methods
   - Collection types follow the singular noun + "Collection" convention (e.g., `AppCollection`, `EndpointCollection`)

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

## Hierarchical Operation Patterns

The config package uses a hierarchical design where parent objects call their children's methods to perform operations. This pattern is consistently applied for:

### 1. Validation Chain

Validation follows a top-down pattern where each component:
- Validates its own properties first
- Calls `Validate()` on all child components
- Aggregates and returns all errors using `errors.Join()`

Example flow:
```
Config.Validate()
  ├── Logging.Validate()
  ├── Listeners.Validate()
  │     └── For each Listener: Listener.Validate()
  ├── Endpoints.Validate()
  │     └── For each Endpoint: Endpoint.Validate()
  │           └── Routes.Validate()
  │                 └── For each Route: Route.Validate()
  │                       └── Condition.Validate()
  └── Apps.Validate()
        └── For each App: App.Validate()
              └── AppConfig.Validate() (Script/CompositeScript/Echo)
                    └── Evaluator.Validate() (for script apps)
                    └── StaticData.Validate() (if present)
```

### 2. Protocol Buffer Conversion

Conversion to/from protocol buffers follows a similar hierarchy:
- Parent objects call `ToProto()` on all child objects 
- Objects are responsible for converting their own fields
- Child objects are assembled into the parent's protocol buffer structure

Example flow:
```
Config.ToProto()
  ├── Logging.ToProto()
  ├── Listeners.ToProto()
  │     └── For each Listener: Listener.ToProto()
  │           └── Options.ToProto() (HTTP/gRPC specific)
  ├── Endpoints.ToProto()
  │     └── For each Endpoint: Endpoint.ToProto()
  │           └── Routes.ToProto()
  │                 └── For each Route: Route.ToProto()
  │                       └── Condition.ToProto()
  │                       └── StaticData.ToProto() (if present)
  └── Apps.ToProto()
        └── For each App: App.ToProto()
              └── AppConfig.ToProto() (Script/CompositeScript/Echo)
                    └── Evaluator.ToProto() (for script apps)
                    └── StaticData.ToProto() (if present)
```

### 3. Tree Generation for Visualization

Tree generation for visualization follows a similar pattern:
- Parent objects call `ToTree()` on all child objects
- Each object is responsible for formatting its own properties
- Child trees are added as branches to the parent tree

Example flow:
```
Config.String() (calls ConfigTree)
  ├── Logging.ToTree()
  ├── Listeners.ToTree()
  │     └── For each Listener: Listener.ToTree()
  ├── Endpoints.ToTree()
  │     └── For each Endpoint: Endpoint.ToTree()
  │           └── Routes.ToTree()
  └── Apps.ToTree()
        └── For each App: App.ToTree()
              └── AppConfig.ToTree() (Script/CompositeScript/Echo)
```

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

4. **App Types**:
   - `App` is the container for any app configuration
   - Three main app types implement the `AppConfig` interface:
     - `Echo`: Simple app that echoes back information
     - `AppScript`: Script app using a specific evaluator (Risor, Starlark, Extism)
     - `CompositeScript`: References multiple script apps via `ScriptAppIDs`

5. **Static Data**:
   - Both Routes and Apps can have associated static data 
   - Static data provides configuration values to apps at runtime
   - The `StaticData` type includes both data and merge strategy

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
    for _, app := range cfg.Apps {
        log.Printf("Found app: %s (type: %s)", app.ID, app.Config.Type())
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

// Now pbConfig is a *pb.ServerConfig object
// You can use it for serialization, sending over network, etc.
```

## Configuration Domain Model

```
Config
├── Logging (logs.Config)
├── Listeners (listeners.ListenerCollection)
│   ├── ID
│   ├── Address
│   ├── Type (HTTP/gRPC)
│   └── Options (HTTPOptions/GRPCOptions)
├── Endpoints (endpoints.EndpointCollection)
│   ├── ID
│   ├── ListenerIDs
│   └── Routes (routes.RouteCollection)
│       ├── AppID
│       ├── StaticData
│       └── Condition (HTTPPathCondition/GRPCServiceCondition)
└── Apps (apps.AppCollection)
    ├── ID
    └── Config (AppConfig interface)
        ├── Echo
        │   └── Response
        ├── AppScript
        │   ├── StaticData
        │   └── Evaluator (Risor/Starlark/Extism)
        └── CompositeScript
            ├── ScriptAppIDs
            └── StaticData
```

The configuration model follows a clear structure:
- Top level Config contains Logging, Listeners, Endpoints, and Apps collections
- Endpoints reference Listeners via ListenerIDs
- Routes within Endpoints reference Apps via AppID
- Routes use different Condition types to determine routing rules
- Apps can contain different configurations implementing the AppConfig interface:
  - Echo: Simple response app
  - AppScript: Script with an evaluator (Risor, Starlark, Extism)
  - CompositeScript: Collection of script apps

## Component-Specific Sub-Packages

### Apps Package (`internal/config/apps`)

The apps package encapsulates all application-related configuration:

* **App Types (`apps.go`, `types.go`)**: Defines the main `App` struct, `AppCollection` type, and the `AppConfig` interface implemented by all app types.
* **Proto Conversion (`proto.go`)**: Converts between domain App types and protobuf representations.
* **Validation (`validate.go`)**: Contains app-specific validation logic, including reference validation between composite apps.
* **String Representation (`string.go`)**: Human-readable output and tree visualization.
* **Collection Operations**: Provides methods like `FindByID` for working with collections of apps.
* **App Implementations**: Several subpackages implement specific app types:
  * `echo`: Simple response-based apps
  * `scripts`: Script-based apps with different evaluators
  * `composite`: Apps that combine multiple script apps

### Echo Apps (`internal/config/apps/echo`)

The echo package provides a simple app type that echoes back request information:

* **Types (`echo.go`)**: Defines the `Echo` struct and methods.
* **Proto Conversion (`proto.go`)**: Converts between domain Echo and protobuf.
* **Validation**: Basic validation to ensure the Echo app has a response.

### Script Apps (`internal/config/apps/scripts`)

The scripts package provides script-based app configurations:

* **Types (`types.go`)**: Defines the `AppScript` struct.
* **Evaluators (`evaluators/`)**: Contains different script evaluation engines:
  * `RisorEvaluator`: For evaluating Risor scripts
  * `StarlarkEvaluator`: For evaluating Starlark scripts
  * `ExtismEvaluator`: For evaluating WebAssembly via Extism
* **Proto Conversion (`proto.go`)**: Converts between domain AppScript and protobuf.
* **Validation (`validate.go`)**: Validates script app configurations.

### Composite Script Apps (`internal/config/apps/composite`)

The composite package provides a way to combine multiple script apps:

* **Types (`types.go`)**: Defines the `CompositeScript` struct.
* **Proto Conversion (`proto.go`)**: Converts between domain CompositeScript and protobuf.
* **Validation (`validate.go`)**: Validates composite script app configurations.

### Static Data (`internal/config/staticdata`)

The staticdata package provides types for passing configuration data:

* **Types (`types.go`)**: Defines the `StaticData` struct with data map and merge mode.
* **Proto Conversion (`proto.go`)**: Converts between domain StaticData and protobuf.
* **Validation (`validate.go`)**: Validates static data configurations.

### Endpoints Package (`internal/config/endpoints`)

The endpoints package handles routing and protocol-independent endpoint configuration:

* **Endpoint Types (`endpoints.go`)**: Defines the `Endpoint` and `EndpointCollection` types.
* **Proto Conversion (`proto.go`)**: Handles conversion to and from protobuf.
* **Validation (`validate.go`)**: Validates endpoint configuration.
* **String Representation (`string.go`)**: Creates human-readable output for endpoints.
* **Routes (`routes/`)**: Contains route definitions and conditions.
* **Structured Routes**: Provides type-safe HTTP and gRPC route representations.

### Listeners Package (`internal/config/listeners`)

The listeners package manages network binding and protocol-specific options:

* **Listener Types (`listeners.go`)**: Defines the `Listener` type and protocol options.
* **Proto Conversion (`proto.go`)**: Converts between domain and protobuf representations.
* **Validation (`validate.go`)**: Ensures listener configuration is valid.
* **String Representation (`string.go`)**: Creates human-readable output for listeners.
* **Protocol Options (`options/`)**: Type-safe options for HTTP and gRPC protocols.

### Logs Package (`internal/config/logs`)

The logs package handles logging configuration:

* **Log Types (`logs.go`)**: Defines types for log format and level.
* **Proto Conversion (`proto.go`)**: Converts between domain and protobuf.
* **Validation (`validate.go`)**: Validates logging configuration.
* **String Representation (`string.go`)**: Creates human-readable output.

### Common Patterns

Within each sub-package:

1. **Collection Types**: Each component has a collection type (`AppCollection`, `EndpointCollection`, `ListenerCollection`, `RouteCollection`) that provides collection-level operations.
2. **Domain Methods**: Domain objects implement methods for common operations like validation, conversion, and string representation.
3. **Type Safety**: Enums are implemented as typed constants with validation methods.
4. **Error Handling**: Each package defines its own error types and validation logic.
5. **Testing**: Comprehensive tests for all functionality.