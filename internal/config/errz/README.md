# Config Error Handling

The `errz` package provides standardized error types for configuration validation and processing. It implements structured error types that provide context and enable proper error handling throughout the config layer.

## Error Types

- **ConfigError**: General configuration validation errors with path context
- **ReferenceError**: Errors for invalid references between configuration components
- **ValidationError**: Type-specific validation failures

## Error Patterns

All config errors implement error wrapping and provide structured information for debugging and user feedback. Errors include the configuration path where they occurred and detailed context about what validation failed.

## Usage

Config validation functions return structured errors that can be unwrapped to access the underlying cause while preserving the configuration context where the error occurred.