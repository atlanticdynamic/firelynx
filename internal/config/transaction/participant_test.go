package transaction

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupParticipantTest(t *testing.T) (slog.Handler, *Participant) {
	t.Helper()
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	participant, err := NewParticipant("test-participant", handler)
	require.NoError(t, err)
	require.NotNil(t, participant)

	return handler, participant
}

func TestNewParticipant(t *testing.T) {
	t.Parallel()

	t.Run("creates new participant with correct initial state", func(t *testing.T) {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

		participant, err := NewParticipant("test-participant", handler)

		require.NoError(t, err)
		assert.Equal(t, "test-participant", participant.Name)
		assert.Equal(t, finitestate.ParticipantNotStarted, participant.GetState())
		assert.NotNil(t, participant.logger)
		assert.NotNil(t, participant.fsm)
		assert.NotZero(t, participant.timestamp)
		require.NoError(t, participant.err)
	})
}

func TestParticipantStateMachine(t *testing.T) {
	t.Parallel()

	t.Run("executes full participant lifecycle", func(t *testing.T) {
		_, p := setupParticipantTest(t)

		// Initial state
		assert.Equal(t, finitestate.ParticipantNotStarted, p.GetState())

		// Execute
		require.NoError(t, p.Execute())
		assert.Equal(t, finitestate.ParticipantExecuting, p.GetState())

		// Mark succeeded
		require.NoError(t, p.MarkSucceeded())
		assert.Equal(t, finitestate.ParticipantSucceeded, p.GetState())

		// Begin compensation
		require.NoError(t, p.BeginCompensation())
		assert.Equal(t, finitestate.ParticipantCompensating, p.GetState())

		// Mark compensated
		require.NoError(t, p.MarkCompensated())
		assert.Equal(t, finitestate.ParticipantCompensated, p.GetState())
	})

	t.Run("handles failure path", func(t *testing.T) {
		_, p := setupParticipantTest(t)

		// Execute
		require.NoError(t, p.Execute())

		// Mark failed
		testErr := errors.New("test failure")
		require.NoError(t, p.MarkFailed(testErr))
		assert.Equal(t, finitestate.ParticipantFailed, p.GetState())
		assert.Equal(t, testErr, p.err)

		// Compensation should not work for failed participants
		require.NoError(t, p.BeginCompensation())
		// State should still be failed
		assert.Equal(t, finitestate.ParticipantFailed, p.GetState())
	})

	t.Run("prevents invalid state transitions", func(t *testing.T) {
		_, p := setupParticipantTest(t)

		// Try to compensate before executing (should be a no-op because BeginCompensation
		// only compensates participants in succeeded state)
		require.NoError(t, p.BeginCompensation())
		assert.Equal(t, finitestate.ParticipantNotStarted, p.GetState())

		// Skip directly to mark compensated without proper flow
		// This should fail because the FSM can't transition from not_started to compensated
		err := p.MarkCompensated()
		require.Error(t, err)

		// We're testing that an error occurs, but not specifically checking the error message
		// since that's an implementation detail of the fsm package
		require.Error(t, err)
		t.Logf("Got expected error: %v", err)
	})
}

func TestParticipantCollection(t *testing.T) {
	t.Parallel()

	setupCollection := func(t *testing.T) *ParticipantCollection {
		t.Helper()
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		return NewParticipantCollection(handler)
	}

	t.Run("creates empty collection", func(t *testing.T) {
		collection := setupCollection(t)

		assert.NotNil(t, collection)
		assert.Empty(t, collection.participants)
		assert.NotNil(t, collection.logger)
		assert.NotNil(t, collection.handler)
	})

	t.Run("adds new participant", func(t *testing.T) {
		collection := setupCollection(t)

		err := collection.AddParticipant("component1")
		require.NoError(t, err)

		assert.Len(t, collection.participants, 1)
		assert.Contains(t, collection.participants, "component1")
		assert.Equal(
			t,
			finitestate.ParticipantNotStarted,
			collection.participants["component1"].GetState(),
		)
	})

	t.Run("prevents duplicate participant addition", func(t *testing.T) {
		collection := setupCollection(t)

		err := collection.AddParticipant("component1")
		require.NoError(t, err)

		// Try to add the same participant again
		err = collection.AddParticipant("component1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("gets or creates participant", func(t *testing.T) {
		collection := setupCollection(t)

		// First call should create
		p1, err := collection.GetOrCreate("component1")
		require.NoError(t, err)
		assert.NotNil(t, p1)

		// Second call should return existing
		p2, err := collection.GetOrCreate("component1")
		require.NoError(t, err)
		assert.Same(t, p1, p2) // Should be the same instance
	})

	t.Run("checks if all participants succeeded", func(t *testing.T) {
		collection := setupCollection(t)

		// Empty collection should return true
		assert.True(t, collection.AllParticipantsSucceeded())

		// Add participants
		p1, err := collection.GetOrCreate("component1")
		require.NoError(t, err)

		p2, err := collection.GetOrCreate("component2")
		require.NoError(t, err)

		// Not all succeeded yet
		assert.False(t, collection.AllParticipantsSucceeded())

		// Mark p1 as succeeded
		require.NoError(t, p1.Execute())
		require.NoError(t, p1.MarkSucceeded())

		// Still not all succeeded
		assert.False(t, collection.AllParticipantsSucceeded())

		// Mark p2 as succeeded
		require.NoError(t, p2.Execute())
		require.NoError(t, p2.MarkSucceeded())

		// Now all should be succeeded
		assert.True(t, collection.AllParticipantsSucceeded())
	})

	t.Run("begins compensation for all participants", func(t *testing.T) {
		collection := setupCollection(t)

		// Add participants and make them succeed
		p1, err := collection.GetOrCreate("component1")
		require.NoError(t, err)
		require.NoError(t, p1.Execute())
		require.NoError(t, p1.MarkSucceeded())

		p2, err := collection.GetOrCreate("component2")
		require.NoError(t, err)
		require.NoError(t, p2.Execute())
		require.NoError(t, p2.MarkSucceeded())

		// Begin compensation for all
		err = collection.BeginCompensation()
		require.NoError(t, err)

		// Both should now be compensating
		assert.Equal(t, finitestate.ParticipantCompensating, p1.GetState())
		assert.Equal(t, finitestate.ParticipantCompensating, p2.GetState())
	})

	t.Run("gets participant states", func(t *testing.T) {
		collection := setupCollection(t)

		// Add participants in different states
		p1, err := collection.GetOrCreate("component1")
		require.NoError(t, err)
		require.NoError(t, p1.Execute())

		p2, err := collection.GetOrCreate("component2")
		require.NoError(t, err)
		require.NoError(t, p2.Execute())
		require.NoError(t, p2.MarkSucceeded())

		// Get states
		states := collection.GetParticipantStates()

		assert.Len(t, states, 2)
		assert.Equal(t, finitestate.ParticipantExecuting, states["component1"])
		assert.Equal(t, finitestate.ParticipantSucceeded, states["component2"])
	})

	t.Run("gets participant errors", func(t *testing.T) {
		collection := setupCollection(t)

		// Add participants with one having an error
		p1, err := collection.GetOrCreate("component1")
		require.NoError(t, err)
		require.NoError(t, p1.Execute())
		testErr := errors.New("test failure")
		require.NoError(t, p1.MarkFailed(testErr))

		p2, err := collection.GetOrCreate("component2")
		require.NoError(t, err)
		require.NoError(t, p2.Execute())
		require.NoError(t, p2.MarkSucceeded())

		// Get errors
		errs := collection.GetParticipantErrors()

		assert.Len(t, errs, 1)
		assert.Contains(t, errs, "component1")
		assert.Equal(t, testErr, errs["component1"])
		assert.NotContains(t, errs, "component2")
	})
}
