# Configuration Loader Package

This package provides utilities for loading, validating, and working with server configurations from different sources, primarily TOML files.

## Overview

The loader package follows a modular design with a standardized interface pattern:

```go
type Loader interface {
    LoadProto() (*pbSettings.ServerConfig, error)
    GetProtoConfig() *pbSettings.ServerConfig
}
```

Different format implementations (like TOML) implement this interface, allowing for a consistent API regardless of the underlying source format.

## Package Structure

- `loader.go`: Defines the Loader interface and core functionality
- `errors.go`: Error types and formatting helpers
- `toml/`: TOML-specific implementation of the Loader interface

## Usage Examples

### Loading from a File

```go
// Load config from a TOML file
loader, err := loader.NewLoaderFromFilePath("/path/to/config.toml")
if err != nil {
    return fmt.Errorf("failed to create loader: %w", err)
}

// Parse and validate the configuration
config, err := loader.LoadProto()
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Use the config
fmt.Println("Config version:", config.GetVersion())
```

### Loading from Bytes

```go
// Load from raw TOML bytes
data := []byte(`
version = "v1"
[logging]
format = "json"
level = "info"
`)

loader, err := loader.NewLoaderFromBytes(data, func(data []byte) loader.Loader {
    return toml.NewTomlLoader(data)
})
if err != nil {
    return err
}

config, err := loader.LoadProto()
if err != nil {
    return err
}
```

## Configuration Flow

1. **Configuration Source**: TOML file, bytes or reader
2. **Loading**: TOML → intermediate representation → Protocol Buffers
3. **Post-processing**: Handle enums, special cases, and nested structures
4. **Validation**: Ensure the configuration is complete and valid
5. **Usage**: The validated Protocol Buffer config is ready for use by the server

## TOML Implementation

The TOML loader supports the full configuration schema with:

- **Listeners**: HTTP and gRPC server configurations
- **Endpoints**: Define routing between listeners and applications
- **Routes**: Rules for routing based on request properties
- **Apps**: Application definitions (echo, script, composite)
- **Logging**: Format and level configurations

## Error Handling

Errors are structured for clear debugging:

- Layered hierarchy (base errors and context-specific errors)
- Error wrapping with `fmt.Errorf("%w", err)` for clean error chains
- Aggregation of multiple errors via `errors.Join()`
- Rich context in error messages (file paths, indexes, IDs)

## Notes for Developers

- When working with Protocol Buffers and TOML, note that field names use camelCase in Proto but snake_case in TOML.
- Protocol Buffer oneofs are represented as nested objects in TOML.
- All validation happens after loading, so partial/invalid configs will fail with clear error messages.
- Error messages include detailed context to help identify issues in large configurations.