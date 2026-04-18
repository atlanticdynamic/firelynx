// Participant state machine implementation.
// Tracks the lifecycle of individual participants in a configuration saga.
package finitestate

import (
	"context"
	"log/slog"
	"time"

	"github.com/robbyt/go-fsm/v2"
	"github.com/robbyt/go-fsm/v2/hooks/broadcast"
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

type ParticipantFSM struct {
	*fsm.Machine
	stateManager *broadcast.Manager
}

func (p *ParticipantFSM) GetStateChan(ctx context.Context) <-chan string {
	return getStateChan(ctx, p.stateManager, p.GetState(), broadcast.WithTimeout(5*time.Second))
}

func NewParticipantFSM(handler slog.Handler) (*ParticipantFSM, error) {
	machine, stateManager, err := newMachine(handler, ParticipantNotStarted, ParticipantTransitions)
	if err != nil {
		return nil, err
	}
	return &ParticipantFSM{Machine: machine, stateManager: stateManager}, nil
}
