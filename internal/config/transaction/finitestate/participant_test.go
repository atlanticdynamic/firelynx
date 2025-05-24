package finitestate

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParticipantMachine(t *testing.T) {
	t.Parallel()

	t.Run("creates new participant machine with correct initial state", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantMachine(handler)

		require.NoError(t, err)
		require.NotNil(t, machine)
		assert.Equal(t, ParticipantNotStarted, machine.GetState())
	})
}

func TestParticipantMachine(t *testing.T) {
	t.Parallel()

	setup := func() Machine {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantMachine(handler)
		require.NoError(t, err)
		return machine
	}

	t.Run("validates successful execution flow", func(t *testing.T) {
		machine := setup()

		// Initial state should be NotStarted
		assert.Equal(t, ParticipantNotStarted, machine.GetState())

		// Execute successfully
		require.NoError(t, machine.Transition(ParticipantExecuting))
		assert.Equal(t, ParticipantExecuting, machine.GetState())

		require.NoError(t, machine.Transition(ParticipantSucceeded))
		assert.Equal(t, ParticipantSucceeded, machine.GetState())
	})

	t.Run("validates failed execution flow", func(t *testing.T) {
		machine := setup()

		// Start executing
		require.NoError(t, machine.Transition(ParticipantExecuting))
		assert.Equal(t, ParticipantExecuting, machine.GetState())

		// Fail
		require.NoError(t, machine.Transition(ParticipantFailed))
		assert.Equal(t, ParticipantFailed, machine.GetState())
	})

	t.Run("validates compensation flow", func(t *testing.T) {
		machine := setup()

		// Get to succeeded state
		require.NoError(t, machine.Transition(ParticipantExecuting))
		require.NoError(t, machine.Transition(ParticipantSucceeded))

		// Begin compensation
		require.NoError(t, machine.Transition(ParticipantCompensating))
		assert.Equal(t, ParticipantCompensating, machine.GetState())

		// Complete compensation
		require.NoError(t, machine.Transition(ParticipantCompensated))
		assert.Equal(t, ParticipantCompensated, machine.GetState())
	})

	t.Run("prevents invalid transitions", func(t *testing.T) {
		machine := setup()

		// Cannot go from NotStarted to Succeeded directly
		err := machine.Transition(ParticipantSucceeded)
		require.Error(t, err)
		assert.Equal(t, ParticipantNotStarted, machine.GetState())

		// Cannot go from Failed to Compensating
		require.NoError(t, machine.Transition(ParticipantExecuting))
		require.NoError(t, machine.Transition(ParticipantFailed))
		err = machine.Transition(ParticipantCompensating)
		require.Error(t, err)
		assert.Equal(t, ParticipantFailed, machine.GetState())
	})
}

func TestParticipantTransitions(t *testing.T) {
	t.Parallel()

	t.Run("verify that all states have defined transitions", func(t *testing.T) {
		// All states should either have valid transitions or be terminal
		states := []string{
			ParticipantNotStarted,
			ParticipantExecuting,
			ParticipantSucceeded,
			ParticipantFailed,
			ParticipantCompensating,
			ParticipantCompensated,
			ParticipantError,
		}

		for _, state := range states {
			_, exists := ParticipantTransitions[state]
			assert.True(t, exists, "State %s is missing from ParticipantTransitions", state)
		}
	})

	t.Run("verify terminal states have no transitions", func(t *testing.T) {
		terminalStates := []string{
			ParticipantCompensated,
			ParticipantError,
		}

		for _, state := range terminalStates {
			transitions := ParticipantTransitions[state]
			assert.Empty(t, transitions, "Terminal state %s should have no transitions", state)
		}
	})
}
