# gRPC Transport Layer (`server`)

This package provides a testable abstraction over gRPC server operations, enabling:

- Decoupling configuration logic from transport concerns
- Unit testing without network dependencies
- Clean separation between API and implementation

## Design

The package separates business logic (in `Runner`) from transport concerns (in `GRPCManager`). `GRPCManager` implements the `GRPCServer` interface that `Runner` depends on, allowing for easy testing with mock implementations.

## Usage

```go
// In production:
mgr, _ := server.NewGRPCManager(logger, ":7070", cfgService)
_ = mgr.Start(ctx)

// In tests:
mockServer := &mockGRPCServer{}
runner := cfgservice.NewRunner(...)
runner.SetGRPCServer(mockServer)
```

`GRPCManager` handles network concerns while `Runner` implements the business logic.