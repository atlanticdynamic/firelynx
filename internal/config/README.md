# Internal Config Package (`internal/config`)

This package defines the domain model for the Firelynx server configuration and provides utilities for loading, validating, and converting configuration data.

## Core Files and Their Purposes

* **Domain Model (`config.go`)**: Defines the core configuration structures used throughout the application. Provides a type-safe way to work with configuration within the application code. Contains the main `Config` struct and related types like `Listener`, `Endpoint`, and `Route`. App-related types have been moved to the dedicated `apps` package.

* **Initialization (`new.go`)**: Contains functions to create and initialize configuration objects from various sources (files, bytes, readers). Handles the initial conversion from protobuf to domain model via `NewFromProto`. Functions like `NewConfig`, `NewConfigFromBytes` and `NewConfigFromReader` orchestrate the full load-convert-validate workflow.

* **Proto Conversion (`proto.go`)**: Handles bidirectional conversion between domain models and protobuf representations. Contains the `ToProto` method for domain-to-proto conversion and the `appFromProto` function used during proto-to-domain conversion. These conversions enable configuration to be loaded from files and transmitted over network protocols. Updated to work with the dedicated apps package.

* **Validation (`validate.go`)**: Performs comprehensive validation of configurations, ensuring integrity and consistency. Checks for issues like duplicate IDs, invalid references between components, and route conflicts. The validation step is critical for preventing runtime errors due to misconfiguration. Updated to delegate app-specific validation to the apps package.

* **Querying (`query.go`)**: Provides helper functions to extract specific subsets of configuration. Enables filtering listeners by type, finding endpoints by listener ID, and retrieving apps by type. These helpers simplify common access patterns and reduce code duplication across the codebase. Reorganized to follow a hierarchical pattern with clear categorization of query methods.

* **String Representation (`string.go`)**: Creates human-readable console output for configurations. Uses the tree structure to visualize the relationships between configuration components. This is particularly useful for debugging and command-line output.

* **Errors (`errors.go`)**: Defines sentinel errors used throughout the package. Standardizes error types for consistent error handling when loading, converting, or validating configurations.

* **Logging (`logging.go`)**: Contains logging-specific configuration and conversion between domain and protobuf representations. Provides type-safe enums for log format and level settings.

* **Listener Options (`listener.go`)**: Implements protocol-specific configuration for different listener types (HTTP, gRPC). Contains helper methods to safely extract timeout and other settings with sensible defaults.

* **Endpoint (`endpoint.go`)**: Contains endpoint-related types and methods, including structured HTTP route representation.

* **Loading (`loader/`)**: Contains the loading infrastructure to read configuration from different sources (files, bytes, readers) and parse it into protobuf. Currently supports TOML format but is designed to allow other formats in the future.

## Core Concepts

* **Domain Model First**: The package is designed to work with Go domain models internally while using protobuf for serialization and transport.

* **Validation Chain**: Configuration goes through a multi-step process: loading → conversion → validation.

* **Protocol Buffers**: The canonical schema is defined using Protocol Buffers, allowing for language-agnostic definitions and interoperability.

* **Type Safety**: The domain model provides type safety features not available in the protobuf-generated code.

* **Modular Organization**: App-related functionality is now in a dedicated package to improve separation of concerns.

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

The primary way to load and obtain a validated domain `Config` object is through the functions provided in `new.go`:

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
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
)

func main() {
	filePath := "path/to/your/config.toml"
	cfg, err := config.NewConfig(filePath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Use the validated cfg object
	log.Printf("Loaded config version: %s", cfg.Version)
	
	// Access apps using the apps package methods
	risorApps := cfg.GetAppsByType("risor")
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

## Package Structure

The config package has been organized into more modular components:

* `internal/config/`: Core domain types and primary loading functions
* `internal/config/loader/`: Loading from different formats/sources
* `internal/config/apps/`: App-related types and validation logic

### Configuration Domain Model

```
Config
├── Listeners[]
│   ├── ID
│   ├── Address
│   ├── Type (HTTP/gRPC)
│   └── Options (HTTPListenerOptions/GRPCListenerOptions)
├── Endpoints[]
│   ├── ID
│   ├── ListenerIDs[]
│   └── Routes[]
│       ├── AppID
│       ├── StaticData
│       └── Condition (HTTPPathCondition/GRPCServiceCondition)
└── Apps[] (from apps package)
    ├── ID
    └── Config
        ├── ScriptApp
        │   ├── StaticData
        │   └── Evaluator (Risor/Starlark/Extism)
        └── CompositeScriptApp
            ├── ScriptAppIDs[]
            └── StaticData
```

The configuration model follows a clear structure:
- Top level Config contains Listeners, Endpoints, and Apps collections
- Endpoints reference Listeners via ListenerIDs
- Routes within Endpoints reference Apps via AppID
- Routes use different Condition types to determine routing rules
- Apps can contain different configurations (ScriptApp, CompositeScriptApp)
- Apps with CompositeScriptApp configuration reference other apps via ScriptAppIDs

### App Package (`internal/config/apps`)

The apps package contains:

* **App Types (`apps.go`)**: Defines app-specific types like `App`, `ScriptApp`, and various evaluator types (`RisorEvaluator`, `StarlarkEvaluator`, `ExtismEvaluator`). This separation allows for better organization of app-related code.
* **Validation (`validation.go`)**: Contains app-specific validation logic, including reference validation between composite apps.
* **Collection Operations**: Provides methods like `FindByID` for working with collections of apps.
* **Evaluator Type Interface**: Defines a common interface for different script evaluation engines.

### Hierarchical Query Design

The query methods in `query.go` have been redesigned to follow a clear hierarchical structure:

* **Top-down queries**: Start from top-level objects (Config) and navigate down through the hierarchy (Listeners → Endpoints → Routes → Apps)
* **Type-based queries**: Filter objects by their type (e.g., `GetListenersByType`, `GetAppsByType`)
* **Reverse lookups**: Find objects that reference a specific object (e.g., `GetEndpointsByListenerID`)

### Structured HTTP Routes

To improve HTTP route handling:
* Added `HTTPRoute` type for structured HTTP route representation
* Implemented `GetStructuredHTTPRoutes()` method to extract HTTP-specific route information
* Maintained backward compatibility with alias methods

### Route Conditions

The configuration system uses a flexible condition system for routes:

* `RouteCondition` interface defines the common contract (Type and Value methods)
* Implementation types:
  * `HTTPPathCondition`: Routes requests based on HTTP path patterns
  * `GRPCServiceCondition`: Routes requests based on gRPC service name
  * `MCPResourceCondition`: Routes requests based on MCP resource type

This pattern allows for:
* Type-safe condition handling in the domain model
* Common interface for condition validation
* Adding new condition types without changing existing code
* Clear serialization to and from protobuf

This modular structure improves maintainability and makes it easier to extend the configuration system as it evolves.