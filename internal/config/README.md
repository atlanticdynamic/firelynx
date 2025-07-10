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

## Environment Variable Interpolation

Config fields can support environment variable interpolation using `${VAR_NAME}` syntax. To add this to new fields:

1. **Protobuf**: Add field comment documenting interpolation support
2. **Domain conversion**: Use `interpolation.ExpandEnvVars()` when converting from protobuf
3. **Validation**: Validate the expanded value, not the raw template