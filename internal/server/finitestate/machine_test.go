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
		assert.Equal(t, StatusNew, machine.GetState())
	})

	t.Run("uses provided handler", func(t *testing.T) {
		// Create a test handler
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		machine, err := New(handler)

		require.NoError(t, err)
		require.NotNil(t, machine)
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

	t.Run("Transition changes state", func(t *testing.T) {
		machine := setup()

		// Initial state should be New
		assert.Equal(t, StatusNew, machine.GetState())

		// Transition to Booting
		err := machine.Transition(StatusBooting)
		require.NoError(t, err)
		assert.Equal(t, StatusBooting, machine.GetState())

		// Transition to Running
		err = machine.Transition(StatusRunning)
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, machine.GetState())
	})

	t.Run("Transition returns error for invalid transition", func(t *testing.T) {
		machine := setup()

		// Try invalid transition (New -> Running)
		err := machine.Transition(StatusRunning)
		require.Error(t, err)
		assert.Equal(
			t,
			StatusNew,
			machine.GetState(),
			"State shouldn't change on failed transition",
		)
	})

	t.Run("TransitionBool returns success status", func(t *testing.T) {
		machine := setup()

		// Valid transition
		success := machine.TransitionBool(StatusBooting)
		assert.True(t, success)
		assert.Equal(t, StatusBooting, machine.GetState())

		// Invalid transition
		success = machine.TransitionBool(StatusNew)
		assert.False(t, success)
		assert.Equal(
			t,
			StatusBooting,
			machine.GetState(),
			"State shouldn't change on failed transition",
		)
	})

	t.Run("TransitionIfCurrentState changes state when condition met", func(t *testing.T) {
		machine := setup()

		// Initial state is New
		err := machine.TransitionIfCurrentState(StatusNew, StatusBooting)
		require.NoError(t, err)
		assert.Equal(t, StatusBooting, machine.GetState())

		// Current state is Booting, should not change to New
		err = machine.TransitionIfCurrentState(StatusNew, StatusNew)
		require.Error(t, err)
		assert.Equal(t, StatusBooting, machine.GetState())

		// Current state is Booting, should change to Running
		err = machine.TransitionIfCurrentState(StatusBooting, StatusRunning)
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, machine.GetState())
	})

	t.Run("SetState forces state change", func(t *testing.T) {
		machine := setup()

		// Set state directly to Running (bypassing normal transitions)
		err := machine.SetState(StatusRunning)
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, machine.GetState())

		// Set state to Error
		err = machine.SetState(StatusError)
		require.NoError(t, err)
		assert.Equal(t, StatusError, machine.GetState())
	})

	t.Run("GetStateChan provides state updates", func(t *testing.T) {
		machine := setup()

		// First transition to booting
		err := machine.Transition(StatusBooting)
		require.NoError(t, err)
		assert.Equal(t, StatusBooting, machine.GetState())

		// Set up context with timeout
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		// Set up the channel to receive state updates
		stateChan := machine.GetStateChan(ctx)
		require.NotNil(t, stateChan)

		// The current state should be delivered immediately on subscription
		select {
		case initialState := <-stateChan:
			assert.Equal(t, StatusBooting, initialState)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for initial state notification")
		}

		// Transition to Running
		err = machine.Transition(StatusRunning)
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, machine.GetState())

		// Wait for the state change notification
		var receivedState string
		select {
		case receivedState = <-stateChan:
			assert.Equal(t, StatusRunning, receivedState)
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for Running state notification")
		}

		// Context cancellation unsubscribes the channel. go-fsm v2 leaves
		// channel closure to the caller because the caller owns the channel.
		cancel()
	})
}

func TestTypicalTransitions(t *testing.T) {
	t.Parallel()

	t.Run("supports the standard lifecycle flow", func(t *testing.T) {
		machine, err := New(slog.NewTextHandler(os.Stdout, nil))
		require.NoError(t, err)

		require.NoError(t, machine.Transition(StatusBooting))
		require.NoError(t, machine.Transition(StatusRunning))
		require.NoError(t, machine.Transition(StatusStopping))
		require.NoError(t, machine.Transition(StatusStopped))
	})
}
