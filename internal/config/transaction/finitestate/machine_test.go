package finitestate

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates new machine with correct initial state", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := New(handler)

		require.NoError(t, err)
		require.NotNil(t, machine)
		assert.Equal(t, StateCreated, machine.GetState())
	})
}

func TestMachineInterface(t *testing.T) {
	t.Parallel()

	setup := func() Machine {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := New(handler)
		require.NoError(t, err)
		return machine
	}

	t.Run("validates basic transaction flow", func(t *testing.T) {
		machine := setup()

		// Initial state should be Created
		assert.Equal(t, StateCreated, machine.GetState())

		// Validate the happy path flow
		transitions := []string{
			StateValidating,
			StateValidated,
			StatePreparing,
			StatePrepared,
			StateCommitting,
			StateCommitted,
			StateCompleted,
		}

		for _, state := range transitions {
			err := machine.Transition(state)
			require.NoError(t, err, "Failed to transition to %s", state)
			assert.Equal(t, state, machine.GetState())
		}
	})

	t.Run("validates rollback flow", func(t *testing.T) {
		machine := setup()

		// Set up a transaction that has been validated and prepared
		require.NoError(t, machine.Transition(StateValidating))
		require.NoError(t, machine.Transition(StateValidated))
		require.NoError(t, machine.Transition(StatePreparing))
		require.NoError(t, machine.Transition(StatePrepared))

		// Now test rollback
		require.NoError(t, machine.Transition(StateRollingBack))
		assert.Equal(t, StateRollingBack, machine.GetState())

		require.NoError(t, machine.Transition(StateRolledBack))
		assert.Equal(t, StateRolledBack, machine.GetState())
	})

	t.Run("validates failure flows", func(t *testing.T) {
		testCases := []struct {
			name               string
			setupTransitions   []string
			failureTransition  string
			expectedFinalState string
		}{
			{
				name:               "failure during validation",
				setupTransitions:   []string{StateValidating},
				failureTransition:  StateFailed,
				expectedFinalState: StateFailed,
			},
			{
				name:               "failure during preparation",
				setupTransitions:   []string{StateValidating, StateValidated, StatePreparing},
				failureTransition:  StateFailed,
				expectedFinalState: StateFailed,
			},
			{
				name: "failure during commit",
				setupTransitions: []string{
					StateValidating,
					StateValidated,
					StatePreparing,
					StatePrepared,
					StateCommitting,
				},
				failureTransition:  StateFailed,
				expectedFinalState: StateFailed,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				machine := setup()

				// Set up the transition history
				for _, state := range tc.setupTransitions {
					require.NoError(t, machine.Transition(state))
				}

				// Now transition to failure
				require.NoError(t, machine.Transition(tc.failureTransition))
				assert.Equal(t, tc.expectedFinalState, machine.GetState())
			})
		}
	})

	t.Run("prevents invalid transitions", func(t *testing.T) {
		machine := setup()

		// Cannot go from Created to Committed directly
		err := machine.Transition(StateCommitted)
		require.Error(t, err)
		assert.Equal(t, StateCreated, machine.GetState())

		// Cannot go back to Created once validating
		require.NoError(t, machine.Transition(StateValidating))
		err = machine.Transition(StateCreated)
		require.Error(t, err)
		assert.Equal(t, StateValidating, machine.GetState())
	})

	t.Run("GetStateChan provides state updates", func(t *testing.T) {
		machine := setup()

		// First transition to validating
		err := machine.Transition(StateValidating)
		require.NoError(t, err)
		assert.Equal(t, StateValidating, machine.GetState())

		// Set up context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set up the channel to receive state updates
		stateChan := machine.GetStateChan(ctx)
		require.NotNil(t, stateChan)

		// Drain any initial state notification that may be present
		select {
		case <-stateChan:
			// Ignore initial state
		case <-time.After(100 * time.Millisecond):
			// No initial state was sent, that's fine
		}

		// Transition to validated
		err = machine.Transition(StateValidated)
		require.NoError(t, err)
		assert.Equal(t, StateValidated, machine.GetState())

		// Wait for the state change notification
		var receivedState string
		select {
		case receivedState = <-stateChan:
			assert.Equal(t, StateValidated, receivedState)
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for validated state notification")
		}

		// Test that the channel closes when context is canceled
		cancel()

		// Wait for channel to close
		select {
		case _, open := <-stateChan:
			if open {
				t.Fatal("Channel should be closed after context cancellation")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for channel to close")
		}
	})
}

func TestTransactionTransitions(t *testing.T) {
	t.Parallel()

	t.Run("verify that all states have defined transitions", func(t *testing.T) {
		// All states should either have valid transitions or be terminal
		states := []string{
			StateCreated,
			StateValidating,
			StateValidated,
			StateInvalid,
			StatePreparing,
			StatePrepared,
			StateCommitting,
			StateCommitted,
			StateCompleted,
			StateRollingBack,
			StateRolledBack,
			StateFailed,
		}

		for _, state := range states {
			_, exists := TransactionTransitions[state]
			assert.True(t, exists, "State %s is missing from TransactionTransitions", state)
		}
	})

	t.Run("verify terminal states have no transitions", func(t *testing.T) {
		terminalStates := []string{
			StateInvalid,
			StateCompleted,
			StateRolledBack,
			StateFailed,
		}

		for _, state := range terminalStates {
			transitions := TransactionTransitions[state]
			assert.Empty(t, transitions, "Terminal state %s should have no transitions", state)
		}
	})
}
