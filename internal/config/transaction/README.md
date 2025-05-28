# Config Transaction Package

The `transaction` package uses a saga pattern for configuration changes. It handles validated configuration transactions that can be applied or rolled back.

## Package Contents

A configuration transaction contains:

- **Domain Config**: Configuration from `internal/config`
- **Participant Tracking**: State management for participating components
- **State Machine**: Transaction lifecycle state tracking
- **Transaction ID**: UUID for correlation
- **Source Information**: Origin (file, API, test)
- **Log Collection**: Transaction logs

## Implementation

### State Machines

- **Transaction FSM**: Tracks lifecycle states
- **Participant FSM**: Each component has its own state machine
- **Coordinated Changes**: Changes apply to all components or none

Transaction states: created → validated → executing → succeeded/failed → reloading/compensating → completed/compensated

### Error Handling

The transaction system categorizes errors into three types using sentinel error wrapping:

#### Error Types

1. **Validation Errors** (`ErrValidationFailed`): Configuration validation failures
   - Triggered by `MarkInvalid(err)`
   - Result in `StateInvalid` (terminal state)
   - Example: Invalid configuration syntax, missing required fields

2. **Terminal Errors** (`ErrTerminalError`): Unrecoverable system errors
   - Triggered by `MarkError(err)` or `MarkFailed(err)`
   - Result in `StateError` or `StateFailed`
   - Example: Database connection failure, filesystem errors

3. **Accumulated Errors** (`ErrAccumulatedError`): Non-fatal errors collected before state transitions
   - Added via `AddError(err)`
   - Do not trigger state transitions
   - Example: Warning conditions, recoverable failures

#### Error Collection

All errors are stored in a unified slice and retrieved via `GetErrors()`. Each error is wrapped with its type using `fmt.Errorf("%w: %w", errorType, originalErr)`. Use `errors.Is()` to check error types.

### Constructors

- `FromFile(path, config, logger)`: File-based transactions
- `FromAPI(requestID, config, logger)`: API-based transactions
- `FromTest(testName, config, logger)`: Test transactions
