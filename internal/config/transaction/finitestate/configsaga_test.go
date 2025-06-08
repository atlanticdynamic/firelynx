// Configuration saga state machine tests.
package finitestate

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSagaFSM(t *testing.T) {
	t.Parallel()

	t.Run("creates new saga machine with correct initial state", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewSagaFSM(handler)

		require.NoError(t, err)
		assert.NotNil(t, machine)
		assert.Equal(t, StateCreated, machine.GetState())
	})
}

func TestSagaMachine(t *testing.T) {
	t.Parallel()

	// setup creates a new state machine for each test
	setup := func() Machine {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewSagaFSM(handler)
		require.NoError(t, err)
		return machine
	}

	t.Run("validates basic saga flow", func(t *testing.T) {
		machine := setup()

		// Initial state should be Created
		assert.Equal(t, StateCreated, machine.GetState())

		// Validate the happy path flow
		transitions := []string{
			StateValidating,
			StateValidated,
			StateExecuting,
			StateSucceeded,
			StateReloading, // Must go through reloading state before completed
			StateCompleted,
		}

		for _, state := range transitions {
			err := machine.Transition(state)
			require.NoError(t, err, "Failed to transition to %s", state)
			assert.Equal(t, state, machine.GetState())
		}
	})

	t.Run("validates compensation flow", func(t *testing.T) {
		machine := setup()

		// Initial state should be Created
		assert.Equal(t, StateCreated, machine.GetState())

		// Setup the precursor states
		err := machine.Transition(StateValidating)
		require.NoError(t, err)
		err = machine.Transition(StateValidated)
		require.NoError(t, err)
		err = machine.Transition(StateExecuting)
		require.NoError(t, err)

		// Now validate the failure path
		err = machine.Transition(StateFailed)
		require.NoError(t, err)
		assert.Equal(t, StateFailed, machine.GetState())

		// Compensation flow
		err = machine.Transition(StateCompensating)
		require.NoError(t, err)
		assert.Equal(t, StateCompensating, machine.GetState())

		err = machine.Transition(StateCompensated)
		require.NoError(t, err)
		assert.Equal(t, StateCompensated, machine.GetState())

		// Should be a terminal state - no further transitions
		err = machine.Transition(StateCreated)
		assert.Error(t, err)
		assert.Equal(t, StateCompensated, machine.GetState()) // State unchanged
	})

	t.Run("validates failure flows", func(t *testing.T) {
		// Different types of failures
		failure := func(transitions []string) func(t *testing.T) {
			return func(t *testing.T) {
				t.Helper()
				machine := setup()

				// Validate the transitions
				for _, state := range transitions {
					err := machine.Transition(state)
					require.NoError(t, err)
					assert.Equal(t, state, machine.GetState())
				}
			}
		}

		t.Run("failure during validation", failure([]string{StateValidating, StateInvalid}))
		t.Run("failure during execution", failure([]string{
			StateValidating,
			StateValidated,
			StateExecuting,
			StateFailed,
		}))
		t.Run("failure after success", failure([]string{
			StateValidating,
			StateValidated,
			StateExecuting,
			StateSucceeded,
			StateFailed,
		}))
	})

	t.Run("prevents invalid transitions", func(t *testing.T) {
		machine := setup()

		// Try to skip a state
		err := machine.Transition(StateValidated)
		assert.Error(t, err)
		assert.Equal(t, StateCreated, machine.GetState()) // State unchanged

		// Try to transition to a state that's not reachable from current state
		err = machine.Transition(StateCompensating)
		assert.Error(t, err)
		assert.Equal(t, StateCreated, machine.GetState()) // State unchanged

		// Setup valid state then try invalid transition
		err = machine.Transition(StateValidating)
		require.NoError(t, err)
		err = machine.Transition(StateCompensated)
		assert.Error(t, err)
		assert.Equal(t, StateValidating, machine.GetState()) // State unchanged
	})

	t.Run("GetStateChan provides state updates", func(t *testing.T) {
		machine := setup()
		ctx := t.Context()

		// Get the state channel
		stateChan := machine.GetStateChan(ctx)
		assert.NotNil(t, stateChan)

		// Make a state transition and check the channel
		err := machine.Transition(StateValidating)
		require.NoError(t, err)

		// Should receive the state change - including the initial state
		var receivedStates []string
		select {
		case state := <-stateChan:
			receivedStates = append(receivedStates, state)
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for state change notification")
		}

		// The behavior of the channel varies - it could send initial state or just the new state
		// Just check that we received at least one state update
		assert.NotEmpty(t, receivedStates)
	})
}

func TestSagaTransitions(t *testing.T) {
	t.Parallel()

	t.Run("verify that all states have defined transitions", func(t *testing.T) {
		allStates := []string{
			StateCreated, StateValidating, StateValidated, StateInvalid,
			StateExecuting, StateSucceeded, StateReloading, StateCompleted,
			StateFailed, StateCompensating, StateCompensated, StateError,
		}

		// Terminal states without transitions
		terminalStates := map[string]bool{
			StateInvalid:     true,
			StateCompleted:   true,
			StateCompensated: true,
			StateError:       true,
		}

		// Check each state except terminal states has defined transitions
		for _, state := range allStates {
			if terminalStates[state] {
				continue
			}

			transitions, exists := SagaTransitions[state]
			assert.True(t, exists, "State %s should have defined transitions", state)
			assert.NotEmpty(t, transitions, "State %s should have at least one transition", state)
		}
	})

	t.Run("verify terminal states have no transitions", func(t *testing.T) {
		terminalStates := []string{
			StateInvalid, StateCompleted, StateCompensated, StateError,
		}

		for _, state := range terminalStates {
			transitions := SagaTransitions[state]
			assert.Empty(t, transitions, "Terminal state %s should have no transitions", state)
		}
	})
}
