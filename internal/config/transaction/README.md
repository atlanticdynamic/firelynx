# Configuration Transaction Layer

`internal/config/transaction` manages the lifecycle of a validated `config.Config` using a saga-style finite-state machine. It coordinates runtime components so a configuration change is either entirely applied or rolled back.

## Responsibilities

* Generate metadata (UUID, source, request ID, timestamps).
* Hold a domain `*config.Config` and derived app instances.
* Drive a finite-state machine (`finitestate.SagaMachine`).
* Register participants and track their individual states.
* Collect structured logs via `loglater.LogCollector`.
* Classify and aggregate errors (validation, terminal, accumulated).

## Out of Scope

* Parsing or validating configuration data — handled by `internal/config`.
* Starting or stopping runtime components — handled by server packages.

## State Machine Overview

```
created → validating ↔ invalid
           ↓
         validated → executing → succeeded → reloading → completed
                               ↘ failed → compensating → compensated
                                ↘ error
```

Valid transitions are declared in `finitestate.SagaTransitions`.

## Primary Errors Returned by the ConfigTransaction Methods

| Marker               | Meaning                                   |
|----------------------|-------------------------------------------|
| `ErrValidationFailed`| Configuration failed domain validation    |
| `ErrTerminalError`   | Unrecoverable infrastructure fault        |
| `ErrAccumulatedError`| Non-fatal issue collected during progress |

## Constructors

The `*config.Config` object must be be loaded before creating a transaction.
```go
transaction.FromFile(path string, cfg *config.Config, h slog.Handler)
transaction.FromAPI(requestID string, cfg *config.Config, h slog.Handler)
transaction.FromTest(testName string, cfg *config.Config, h slog.Handler)
```

## Quick Start

```go
cfg, _ := config.NewConfig("config.toml")

// nil handler uses a default slog text handler.
tx, err := transaction.FromFile("config.toml", cfg, nil)
if err != nil {
    log.Fatal(err)
}

if err := tx.RunValidation(); err != nil {
    log.Printf("invalid config: %v", err)
    return
}

log.Printf("transaction %s in state %s", tx.GetTransactionID(), tx.GetState())
```

There are many other phases to the transaction lifecycle, but they are handled by the `txmgr` Runnable.
