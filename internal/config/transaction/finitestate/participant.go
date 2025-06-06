// Participant state machine implementation.
// Tracks the lifecycle of individual participants in a configuration saga.
package finitestate

import (
	"context"
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

type ParticipantFSM struct {
	*fsm.Machine
}

func (p *ParticipantFSM) GetStateChan(ctx context.Context) <-chan string {
	return p.GetStateChanWithOptions(ctx, fsm.WithSyncBroadcast())
}

func NewParticipantFSM(handler slog.Handler) (*ParticipantFSM, error) {
	machine, err := fsm.New(handler, ParticipantNotStarted, ParticipantTransitions)
	if err != nil {
		return nil, err
	}
	return &ParticipantFSM{Machine: machine}, nil
}
