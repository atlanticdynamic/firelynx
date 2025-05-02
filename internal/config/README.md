# Internal Config Package (`internal/config`)

This package defines the domain model for the Firelynx server configuration and provides utilities for loading, validating, and converting configuration data.

## Core Files and Their Purposes

* **Domain Model (`config.go`)**: Defines the core configuration structures used throughout the application. Provides a type-safe way to work with configuration within the application code. Contains the main `Config` struct and related types like `Listener`, `Endpoint`, `App`, and `Route`.

* **Initialization (`new.go`)**: Contains functions to create and initialize configuration objects from various sources (files, bytes, readers). Handles the initial conversion from protobuf to domain model via `NewFromProto`. Functions like `NewConfig`, `NewConfigFromBytes` and `NewConfigFromReader` orchestrate the full load-convert-validate workflow.

* **Proto Conversion (`proto.go`)**: Handles bidirectional conversion between domain models and protobuf representations. Contains the `ToProto` method for domain-to-proto conversion and the `appFromProto` function used during proto-to-domain conversion. These conversions enable configuration to be loaded from files and transmitted over network protocols.

* **Validation (`validate.go`)**: Performs comprehensive validation of configurations, ensuring integrity and consistency. Checks for issues like duplicate IDs, invalid references between components, and route conflicts. The validation step is critical for preventing runtime errors due to misconfiguration.

* **Querying (`query.go`)**: Provides helper functions to extract specific subsets of configuration. Enables filtering listeners by type, finding endpoints by listener ID, and retrieving apps by type. These helpers simplify common access patterns and reduce code duplication across the codebase.

* **String Representation (`string.go`)**: Creates human-readable console output for configurations. Uses the tree structure to visualize the relationships between configuration components. This is particularly useful for debugging and command-line output.

* **Errors (`errors.go`)**: Defines sentinel errors used throughout the package. Standardizes error types for consistent error handling when loading, converting, or validating configurations.

* **Logging (`logging.go`)**: Contains logging-specific configuration and conversion between domain and protobuf representations. Provides type-safe enums for log format and level settings.

* **Listener Options (`listener.go`)**: Implements protocol-specific configuration for different listener types (HTTP, gRPC). Contains helper methods to safely extract timeout and other settings with sensible defaults.

* **Loading (`loader/`)**: Contains the loading infrastructure to read configuration from different sources (files, bytes, readers) and parse it into protobuf. Currently supports TOML format but is designed to allow other formats in the future.

## Core Concepts

* **Domain Model First**: The package is designed to work with Go domain models internally while using protobuf for serialization and transport.

* **Validation Chain**: Configuration goes through a multi-step process: loading → conversion → validation.

* **Protocol Buffers**: The canonical schema is defined using Protocol Buffers, allowing for language-agnostic definitions and interoperability.

* **Type Safety**: The domain model provides type safety features not available in the protobuf-generated code.

## Usage

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
)

func main() {
	filePath := "path/to/your/config.toml"
	cfg, err := config.NewConfig(filePath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Use the validated cfg object
	log.Printf("Loaded config version: %s", cfg.Version)
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

## Future Package Structure Considerations

As complexity grows, consider further organizing the package into more modular components:

* `internal/config/`: Core domain types and primary loading functions
* `internal/config/loader/`: (Already exists) Loading from different formats/sources
* `internal/config/validation/`: Validation logic
* `internal/config/convert/`: Proto/domain conversion logic
* `internal/config/query/`: Query helpers

This separation could improve modularity and maintainability as the configuration system evolves.