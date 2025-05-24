// Participant state machine implementation.
// Tracks the lifecycle of individual participants in a configuration saga.
package finitestate

import (
	"log/slog"

	"github.com/robbyt/go-fsm"
)

// Participant state constants
const (
	ParticipantNotStarted   = "not_started"
	ParticipantExecuting    = "executing"
	ParticipantSucceeded    = "succeeded"
	ParticipantFailed       = "failed"
	ParticipantCompensating = "compensating"
	ParticipantCompensated  = "compensated"
	ParticipantError        = "error"
)

// ParticipantTransitions defines valid state transitions for saga participants
var ParticipantTransitions = map[string][]string{
	ParticipantNotStarted:   {ParticipantExecuting, ParticipantError},
	ParticipantExecuting:    {ParticipantSucceeded, ParticipantFailed, ParticipantError},
	ParticipantSucceeded:    {ParticipantCompensating, ParticipantError},
	ParticipantFailed:       {ParticipantError},
	ParticipantCompensating: {ParticipantCompensated, ParticipantError},
	ParticipantCompensated:  {},
	ParticipantError:        {},
}

// ParticipantFactory creates participant state machines
type ParticipantFactory struct{}

// NewMachine creates a new participant state machine
func (f *ParticipantFactory) NewMachine(handler slog.Handler) (Machine, error) {
	return fsm.New(handler, ParticipantNotStarted, ParticipantTransitions)
}

// NewParticipantMachine creates a new participant state machine directly
func NewParticipantMachine(handler slog.Handler) (Machine, error) {
	factory := &ParticipantFactory{}
	return factory.NewMachine(handler)
}
