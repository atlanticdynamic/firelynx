# HTTP Listener Status

**Last Updated:** May 2025

## Current State

The HTTP listener saga participant implementation is **complete and integrated**. It's registered in server.go and working in the system.

## What's Done

1. **Saga Participant Implementation** (`runner.go`)
   - StageConfig: Validates and stores pending config
   - CompensateConfig: Rolls back pending config
   - CommitConfig: Commits config and reloads servers
   
2. **Configuration Management** (`cfg/`)
   - ConfigManager: Thread-safe current/pending config storage
   - Adapter: Extracts HTTP config from transactions, creates routes with app instances

3. **Server Management** (`httpserver/server.go`)
   - Wraps go-supervisor's httpserver.Runner
   - No Reloadable/ReloadableWithConfig interfaces (as required)

4. **System Integration**
   - Registered in server.go with saga orchestrator
   - Part of the supervisor runnable set

## What Needs Work

The only remaining work is in `cfg/adapter.go` (see TODO at line 234):

```go
// TODO: This is where we'd implement a more complex handler that:
// - Handles route parameters (e.g., /api/users/{id})
// - Implements middleware chains
// - Handles content negotiation
// - Supports streaming responses
// - Implements WebSocket upgrades
// - Provides better error responses
```

Current implementation:
- Creates basic handlers that call app.HandleHTTP()
- Passes static data to apps
- Returns 500 on errors

## Testing

Integration tests exist and verify:
- Saga participant lifecycle
- Configuration updates
- Basic HTTP routing with mock apps

Need more tests for:
- Multiple listeners on different ports
- Rollback when another participant fails
- Concurrent requests during config updates

## No Routing Registry

The implementation successfully eliminated the routing registry:
- Routes created directly with app instances
- No registry lookups at request time
- App instances embedded in handler closures

## Critical Design Constraint

**HTTPServer must NOT implement `ReloadableWithConfig` or `Reloadable`**

The composite.Runner checks children during Reload():
1. If child implements `ReloadableWithConfig` → calls `ReloadWithConfig(config)`
2. If child implements `Reloadable` → calls `Reload()`

If HTTPServer implemented either interface, the composite runner would reload it directly, bypassing the saga transaction pattern. Configuration changes must ONLY flow through StageConfig → CommitConfig.

To verify this behavior, check:
- `go doc github.com/robbyt/go-supervisor/runnables/composite`
- Look at `composite.Runner.reloadConfig()` method
- See lines 109-118 in composite/reload.go

## Summary

The HTTP listener rewrite is **functionally complete**. The remaining TODO is for enhanced features (middleware, WebSockets, etc.) but the core saga participant implementation works correctly.