// Package finitestate provides state machine capabilities for transactions.
package finitestate

import "log/slog"

// Legacy state constants for backward compatibility with existing code
const (
	StatePreparing   = StateExecuting
	StatePrepared    = StateSucceeded
	StateCommitting  = StateExecuting
	StateCommitted   = StateSucceeded
	StateRollingBack = StateCompensating
	StateRolledBack  = StateCompensated
)

// TransactionTransitions is the legacy transitions map for backward compatibility
var TransactionTransitions = SagaTransitions

// New is a legacy function that creates a saga state machine for backward compatibility
func New(handler slog.Handler) (Machine, error) {
	return NewSagaMachine(handler)
}
