# Configuration Service (`cfgservice`)

`cfgservice` hosts the gRPC `ConfigService` API used by clients to read and update the running configuration.

## Responsibilities

* Provide two RPCs
  * `UpdateConfig` – accept a `pb.ServerConfig`, convert to domain config, create a `transaction.ConfigTransaction`, run `RunValidation`, and forward the transaction to the transaction-manager channel.
  * `GetConfig` – return a deep clone of the current active configuration from storage.
* Manage a `GRPCServer` instance and implement `supervisor.Runnable` for orderly startup and shutdown.
* Expose functional options (`WithLogger`, `WithGRPCServer`, `WithConfigTransactionStorage`, etc.) to aid testing and integration.

## Out of Scope

* Transaction orchestration and persistence – handled by `txmgr`.
* File-based configuration sources – handled by `cfgfileloader`.

## Key Types

```go
// Runner implements supervisor.Runnable and pb.ConfigServiceServer.
type Runner struct { /* see runner.go */ }

// Option configures a Runner.
type Option func(*Runner)
```

## Quick Start

```go
import (
    "github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
    "github.com/atlanticdynamic/firelynx/internal/config/transaction"
    "github.com/robbyt/go-supervisor/supervisor"
)

txCh := make(chan *transaction.ConfigTransaction, 1)
svc, _ := cfgservice.NewRunner("0.0.0.0:7070", txCh)

sup := supervisor.New("cfgservice", svc)
_ = sup.Run()
```

`UpdateConfig` calls are validated before being placed on `txCh`; `txmgr` consumes the channel to coordinate rollout.