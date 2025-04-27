# Internal Config Package (`internal/config`)

This package defines the domain model for the Firelynx server configuration and provides utilities for loading, validating, and converting configuration data.

## Core Concepts

*   **Domain Model (`config.go`)**: Defines Go structs representing the configuration structure (e.g., `Config`, `Listener`, `Endpoint`, `App`, `Route`). This provides a type-safe way to work with configuration within the application code.
*   **Loading (`loader/`)**: Handles reading configuration data from various sources (files, bytes, readers) and parsing it into a Protobuf representation (`pbSettings.ServerConfig`). Currently supports TOML format.
*   **Protobuf (`gen/settings/v1alpha1`)**: The canonical schema for configuration is defined using Protocol Buffers. This allows for language-agnostic configuration definitions and potential interoperability.
*   **Conversion (`conversion.go`, `proto.go`)**: Provides functions to convert between the Protobuf representation (`pbSettings.ServerConfig`) and the Go domain model (`Config`).
    *   `FromProto`: Converts Protobuf -> Domain Model.
    *   `ToProto`: Converts Domain Model -> Protobuf.
*   **Validation (`validate.go`)**: Contains logic to validate the integrity and consistency of the loaded domain `Config` object (e.g., checking for duplicate IDs, valid references, route conflicts).
*   **Utilities (`util.go`, `query.go`)**: Helper functions for tasks like converting Protobuf value types and querying specific parts of the configuration.

## Usage

### Loading Configuration

The primary way to load and obtain a validated domain `Config` object is through the functions provided in `new.go`:

1.  **Choose a source:**
    *   From a file path: `config.NewConfig(filePath string)`
    *   From raw bytes: `config.NewConfigFromBytes(data []byte)`
    *   From an `io.Reader`: `config.NewConfigFromReader(reader io.Reader)`

2.  **Process:** These functions perform the following steps internally:
    *   Use the appropriate `loader` function (e.g., `loader.NewLoaderFromFilePath`) to get a `loader.Loader`.
    *   Call `loader.LoadProto()` to parse the source (e.g., TOML file) into a `*pbSettings.ServerConfig` Protobuf object.
    *   Call `config.FromProto()` (defined in `conversion.go`) to convert the `*pbSettings.ServerConfig` into the domain `*Config` object.
    *   Call `config.Validate()` to perform validation checks on the domain `*Config`.

3.  **Result:** If all steps succeed, you receive a valid `*Config` object. Otherwise, an error detailing the failure (loading, conversion, or validation) is returned.

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

If you have a domain `*Config` object (perhaps constructed or modified programmatically) and need its Protobuf representation, use the `ToProto()` method:

```go
// Assume 'cfg' is a valid *config.Config object
pbConfig := cfg.ToProto()

// Now pbConfig is a *pbSettings.ServerConfig object
// You can use it for serialization, sending over network, etc.
```

## Package Structure Suggestion

While currently flat, consider organizing the `internal/config` package further if complexity grows:

*   `internal/config/`: Contains the core domain types (`config.go`) and primary loading functions (`new.go`).
*   `internal/config/loader/`: (Already exists) Handles loading from different formats/sources.
*   `internal/config/validation/`: Contains validation logic (`validate.go`).
*   `internal/config/convert/`: Contains proto/domain conversion logic (`conversion.go`, `proto.go`, `util.go`).
*   `internal/config/query/`: Contains query helpers (`query.go`).

This separation can improve modularity and maintainability as the configuration system evolves.
