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

func TestNewParticipantFSM(t *testing.T) {
	t.Parallel()

	t.Run("creates new participant machine with correct initial state", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)

		require.NoError(t, err)
		require.NotNil(t, machine)
		assert.Equal(t, ParticipantNotStarted, machine.GetState())
	})

	t.Run("handles nil handler gracefully", func(t *testing.T) {
		machine, err := NewParticipantFSM(nil)

		// Should either return an error or handle nil handler gracefully
		if err != nil {
			assert.Nil(t, machine)
		} else {
			require.NotNil(t, machine)
			assert.Equal(t, ParticipantNotStarted, machine.GetState())
		}
	})
}

func TestParticipantMachine(t *testing.T) {
	t.Parallel()

	setup := func() Machine {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
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

func TestParticipantFSM_GetStateChan(t *testing.T) {
	t.Parallel()

	t.Run("emits state changes through channel", func(t *testing.T) {
		ctx := t.Context()
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		stateChan := machine.GetStateChan(ctx)

		// Should receive initial state
		select {
		case state := <-stateChan:
			assert.Equal(t, ParticipantNotStarted, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for initial state")
		}

		// Transition and verify state change
		require.NoError(t, machine.Transition(ParticipantExecuting))

		select {
		case state := <-stateChan:
			assert.Equal(t, ParticipantExecuting, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for executing state")
		}
	})

	t.Run("closes channel on context cancellation", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		stateChan := machine.GetStateChan(ctx)

		// Get initial state
		select {
		case <-stateChan:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for initial state")
		}

		// Cancel context
		cancel()

		// Channel should be closed
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-stateChan:
				return !ok
			default:
				return false
			}
		}, 200*time.Millisecond, 10*time.Millisecond, "channel should be closed after context cancellation")
	})

	t.Run("multiple listeners receive state changes", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		ctx1 := t.Context()
		ctx2 := t.Context()

		stateChan1 := machine.GetStateChan(ctx1)
		stateChan2 := machine.GetStateChan(ctx2)

		// Both should receive initial state
		assert.Eventually(t, func() bool {
			select {
			case state := <-stateChan1:
				return state == ParticipantNotStarted
			default:
				return false
			}
		}, 100*time.Millisecond, 10*time.Millisecond)

		assert.Eventually(t, func() bool {
			select {
			case state := <-stateChan2:
				return state == ParticipantNotStarted
			default:
				return false
			}
		}, 100*time.Millisecond, 10*time.Millisecond)

		// Transition and verify both receive the change
		require.NoError(t, machine.Transition(ParticipantExecuting))

		assert.Eventually(t, func() bool {
			select {
			case state := <-stateChan1:
				return state == ParticipantExecuting
			default:
				return false
			}
		}, 100*time.Millisecond, 10*time.Millisecond)

		assert.Eventually(t, func() bool {
			select {
			case state := <-stateChan2:
				return state == ParticipantExecuting
			default:
				return false
			}
		}, 100*time.Millisecond, 10*time.Millisecond)
	})
}

func TestParticipantFSM_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles error state transitions", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		// Can transition to error from not_started
		require.NoError(t, machine.Transition(ParticipantError))
		assert.Equal(t, ParticipantError, machine.GetState())
	})

	t.Run("error state is terminal", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		// Get to error state
		require.NoError(t, machine.Transition(ParticipantError))

		// Cannot transition from error state
		err = machine.Transition(ParticipantExecuting)
		assert.Error(t, err)
		assert.Equal(t, ParticipantError, machine.GetState())
	})

	t.Run("can reach error from any non-terminal state", func(t *testing.T) {
		nonTerminalStates := []struct {
			name  string
			setup func(Machine)
		}{
			{
				name: "from executing",
				setup: func(m Machine) {
					require.NoError(t, m.Transition(ParticipantExecuting))
				},
			},
			{
				name: "from succeeded",
				setup: func(m Machine) {
					require.NoError(t, m.Transition(ParticipantExecuting))
					require.NoError(t, m.Transition(ParticipantSucceeded))
				},
			},
			{
				name: "from compensating",
				setup: func(m Machine) {
					require.NoError(t, m.Transition(ParticipantExecuting))
					require.NoError(t, m.Transition(ParticipantSucceeded))
					require.NoError(t, m.Transition(ParticipantCompensating))
				},
			},
		}

		for _, tc := range nonTerminalStates {
			t.Run(tc.name, func(t *testing.T) {
				handler := slog.NewTextHandler(os.Stdout, nil)
				machine, err := NewParticipantFSM(handler)
				require.NoError(t, err)

				tc.setup(machine)

				// Should be able to transition to error
				require.NoError(t, machine.Transition(ParticipantError))
				assert.Equal(t, ParticipantError, machine.GetState())
			})
		}
	})
}

func TestParticipantFSM_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("handles concurrent transitions safely", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		// Transition to executing
		require.NoError(t, machine.Transition(ParticipantExecuting))

		// Try concurrent transitions
		done := make(chan bool, 2)
		errors := make(chan error, 2)

		go func() {
			err := machine.Transition(ParticipantSucceeded)
			errors <- err
			done <- true
		}()

		go func() {
			err := machine.Transition(ParticipantFailed)
			errors <- err
			done <- true
		}()

		// Wait for both
		<-done
		<-done

		// One should succeed, one should fail
		err1 := <-errors
		err2 := <-errors

		// Exactly one should have succeeded
		assert.True(t, (err1 == nil) != (err2 == nil), "exactly one transition should succeed")

		// Final state should be either succeeded or failed
		finalState := machine.GetState()
		assert.True(t, finalState == ParticipantSucceeded || finalState == ParticipantFailed,
			"final state should be succeeded or failed, got %s", finalState)
	})

	t.Run("GetState is safe for concurrent access", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, nil)
		machine, err := NewParticipantFSM(handler)
		require.NoError(t, err)

		// Start multiple goroutines reading state
		done := make(chan bool, 10)
		for range 10 {
			go func() {
				for range 100 {
					_ = machine.GetState()
				}
				done <- true
			}()
		}

		// Perform transitions while reads are happening
		go func() {
			assert.NoError(t, machine.Transition(ParticipantExecuting))
			assert.NoError(t, machine.Transition(ParticipantSucceeded))
			assert.NoError(t, machine.Transition(ParticipantCompensating))
			assert.NoError(t, machine.Transition(ParticipantCompensated))
		}()

		// Wait for all readers
		for range 10 {
			<-done
		}
	})
}
