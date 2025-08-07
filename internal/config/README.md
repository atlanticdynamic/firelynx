# Configuration Domain Layer

`internal/config` defines the in-memory (domain) representation of the Firelynx server configuration.

## Responsibilities

* **Load & convert** – transform the protobuf `pb.ServerConfig` produced by the loader into Go structs (`config.Config`) and back.
* **Validate** – enforce cross-object rules that TOML or protobuf schemas cannot express (e.g. every `Endpoint.ListenerIDs` must reference an existing listener).
* **Query** – helper methods to quickly locate listeners, endpoints, routes, and apps.

## Out of Scope

The package does **not**:

* Start listeners, run apps, or keep runtime state.
* Pre-load configuration during component construction. Runnables fetch config via callbacks during `Run` or on `Reload`.

## Relationship to Protobuf

```text
TOML → loader → pb.ServerConfig ↔ internal/config ↔ server/core ↔ runtime components
```

Protobuf is the persistence/wire schema. Domain structs mirror the proto messages but add:

* idiomatic Go naming and zero-values,
* collection helpers (e.g. `FindByID`),
* validation and string/tree utilities.

Conversions are performed by `NewFromProto` and `ToProto`.

## Config Transactions

Validated configs are wrapped in a `transaction.ConfigTransaction` (`internal/config/transaction`). The transaction layer:

1. assigns a UUID and source to the change,
2. tracks progress with a state machine,
3. coordinates participants so all components apply or all roll back.

The saga logic lives entirely in the transaction package; the domain layer is only the payload.

## Quick Start

```go
cfg, _ := config.NewConfig("config.toml")
_ = cfg.Validate()

fmt.Println(cfg.Version)
listener := cfg.Listeners.FindByID("public-http")
```

The rest of the server interacts with configuration exclusively through this API, allowing the TOML and protobuf schemas to evolve without touching runtime code.

## Default Value Pattern

Configuration types should provide reasonable defaults when users omit optional fields:

1. **Define constants** with `Default` prefix for all configurable values
2. **Constructor** applies all defaults (e.g., `NewHTTP()`)
3. **FromProto conversion** always starts with defaults, only overrides when protobuf fields are provided
4. **Parent conversion** always calls FromProto, even when protobuf fields are nil

This ensures users get reasonable defaults even when configuration sections are completely omitted. See `internal/config/listeners/options/http.go` for the reference implementation.

**Note**: Protobuf Duration fields cannot have default values in the schema, so defaults must be implemented in the Go conversion layer.

## Validation Architecture

Configuration validation follows a strict two-phase architecture:

### Phase 1: Domain Creation
- **`NewFromProto()`** - Converts protobuf to domain objects
- **Applies defaults** - Sets reasonable defaults for omitted fields
- **NO validation** - Only data transformation, never validates business rules

### Phase 2: Validation
- **`Validate()`** - Validates business rules and cross-object constraints
- **Environment variable interpolation** - Expands `${VAR_NAME}` syntax during validation
- **Error accumulation** - Collects all validation errors using `errors.Join()`

### Timing
- **Interpolation happens during validation** - Not during conversion from protobuf
- **Validate interpolated values** - Business rules apply to expanded values, not templates
- **Explicit validation required** - Callers must call `.Validate()` explicitly

## Validation Flow

The configuration validation happens in the following order:

1. **Basic structure validation** - Validate IDs, required fields, and data types
2. **App expansion for routes** (`expandAppsForRoutes`) - Create route-specific app instances with merged static data
3. **Individual component validation** - Validate apps, listeners, endpoints individually
4. **Cross-component validation** - Validate references between components (routes to apps, endpoints to listeners)

## App Expansion

Apps are expanded during the validation phase to create route-specific instances. Each route that references an app gets its own instance with merged static data from both the app definition and route-specific configuration.

**Key Points:**
- Expansion happens in validation phase, not during protobuf conversion
- Each route gets a dedicated app instance with pre-merged static data
- Static data merging is completed before server instantiation
- Server components receive fully-prepared app instances

## Environment Variable Interpolation

Config fields support environment variable interpolation using `${VAR_NAME}` and `${VAR_NAME:default}` syntax.

### Implementation
- **Tag-based control**: Use `env_interpolation:"yes"/"no"` struct tags
- **Validation-time only**: Interpolation happens during `Validate()`, not conversion
- **Fields without tags**: Default to NOT being interpolated

### Supported Fields
- **Paths and URIs**: File paths, URLs, addresses
- **User content**: Messages, responses, configuration values
- **Network**: Host names, ports, endpoints

### Not Interpolated
- **Identifiers**: App IDs, listener IDs, middleware IDs
- **Code content**: Script source, function names, entrypoints
- **Structured data**: JSON, YAML, or other parseable content