package transaction

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
)

// Participant represents a single saga participant with its state machine.
// Each component that processes a configuration change is represented as a participant
// with its own lifecycle state tracking.
type Participant struct {
	Name      string
	fsm       finitestate.Machine
	logger    *slog.Logger
	timestamp time.Time
	err       error
}

// NewParticipant creates a new saga participant with its own state machine.
func NewParticipant(name string, handler slog.Handler) (*Participant, error) {
	fsm, err := finitestate.NewParticipantMachine(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to create participant state machine: %w", err)
	}

	logger := slog.New(handler).WithGroup("participant." + name)

	return &Participant{
		Name:      name,
		fsm:       fsm,
		logger:    logger,
		timestamp: time.Now(),
	}, nil
}

// GetState returns the current state of the participant.
func (p *Participant) GetState() string {
	return p.fsm.GetState()
}

// Execute transitions the participant to executing state.
// This happens when the participant starts processing the configuration.
func (p *Participant) Execute() error {
	err := p.fsm.Transition(finitestate.ParticipantExecuting)
	if err != nil {
		return err
	}

	p.timestamp = time.Now()
	p.logger.Debug("Participant executing")
	return nil
}

// MarkSucceeded transitions the participant to succeeded state.
// This happens when the participant has successfully processed the configuration
// and is ready for the commit phase.
func (p *Participant) MarkSucceeded() error {
	err := p.fsm.Transition(finitestate.ParticipantSucceeded)
	if err != nil {
		return err
	}

	p.timestamp = time.Now()
	p.logger.Debug("Participant succeeded")
	return nil
}

// MarkFailed transitions the participant to failed state.
// This happens when the participant encounters an error during configuration processing.
func (p *Participant) MarkFailed(err error) error {
	transErr := p.fsm.Transition(finitestate.ParticipantFailed)
	if transErr != nil {
		return transErr
	}

	p.timestamp = time.Now()
	p.err = err
	p.logger.Error("Participant failed", "error", err)
	return nil
}

// BeginCompensation transitions the participant to compensating state.
// Called when the saga needs to revert changes due to a failure.
// Only participants that succeeded are compensated.
func (p *Participant) BeginCompensation() error {
	// Only participants that succeeded can be compensated
	if p.GetState() != finitestate.ParticipantSucceeded {
		return nil // Nothing to compensate
	}

	err := p.fsm.Transition(finitestate.ParticipantCompensating)
	if err != nil {
		return err
	}

	p.timestamp = time.Now()
	p.logger.Debug("Participant compensating")
	return nil
}

// MarkCompensated transitions the participant to compensated state.
// This happens when the participant has successfully reverted its changes.
func (p *Participant) MarkCompensated() error {
	err := p.fsm.Transition(finitestate.ParticipantCompensated)
	if err != nil {
		return err
	}

	p.timestamp = time.Now()
	p.logger.Debug("Participant compensated")
	return nil
}

// ParticipantCollection manages a group of saga participants.
// It provides thread-safe access to participant states and coordinates
// compensation across all participants if needed.
type ParticipantCollection struct {
	participants map[string]*Participant
	logger       *slog.Logger
	handler      slog.Handler
	mu           sync.RWMutex
}

// NewParticipantCollection creates a new participant collection.
func NewParticipantCollection(handler slog.Handler) *ParticipantCollection {
	logger := slog.New(handler).WithGroup("participantCollection")

	return &ParticipantCollection{
		participants: make(map[string]*Participant),
		logger:       logger,
		handler:      handler,
	}
}

// GetOrCreate returns an existing participant or creates a new one.
// Used to get or initialize the state tracking for a component.
func (c *ParticipantCollection) GetOrCreate(name string) (*Participant, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, exists := c.participants[name]
	if !exists {
		var err error
		p, err = NewParticipant(name, c.handler)
		if err != nil {
			return nil, err
		}
		c.participants[name] = p
	}

	return p, nil
}

// AddParticipant adds a new participant to the collection.
// Used by the RegisterParticipant method in ConfigTransaction.
func (c *ParticipantCollection) AddParticipant(name string) error {
	_, err := c.GetOrCreate(name)
	return err
}

// AllParticipantsSucceeded returns true if all participants have succeeded.
// Used to determine if the saga can proceed to the commit phase.
func (c *ParticipantCollection) AllParticipantsSucceeded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.participants) == 0 {
		return true // Nothing to do means success
	}

	for _, p := range c.participants {
		if p.GetState() != finitestate.ParticipantSucceeded {
			return false
		}
	}

	return true
}

// BeginCompensation starts compensation for all succeeded participants.
// Called when a participant fails and we need to revert changes.
func (c *ParticipantCollection) BeginCompensation() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Beginning compensation for all succeeded participants")

	var errs []error
	for name, p := range c.participants {
		if err := p.BeginCompensation(); err != nil {
			c.logger.Error("Failed to begin compensation for participant",
				"participant", name,
				"error", err)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// GetParticipantStates returns a map of all participant states.
// Useful for diagnostics and monitoring.
func (c *ParticipantCollection) GetParticipantStates() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	states := make(map[string]string, len(c.participants))
	for name, p := range c.participants {
		states[name] = p.GetState()
	}

	return states
}

// GetParticipantErrors returns a map of participant names to their errors.
// Useful for diagnostics and reporting failures.
func (c *ParticipantCollection) GetParticipantErrors() map[string]error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	errs := make(map[string]error, len(c.participants))
	for name, p := range c.participants {
		if p.err != nil {
			errs[name] = p.err
		}
	}

	return errs
}
