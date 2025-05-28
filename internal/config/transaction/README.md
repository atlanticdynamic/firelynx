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

1. **Validation**: Pre-execution configuration validation
2. **Failure Detection**: Component-level failure tracking
3. **Compensation**: Rolling back successful changes when failures occur
4. **Logging**: Transaction-scoped logging
5. **Error Context**: Error wrapping for context preservation

### Constructors

- `FromFile(path, config, logger)`: File-based transactions
- `FromAPI(requestID, config, logger)`: API-based transactions
- `FromTest(testName, config, logger)`: Test transactions
