package cfgservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/finitestate"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_GetState(t *testing.T) {
	t.Run("initial state is New", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		state := r.GetState()
		assert.Equal(t, finitestate.StatusNew, state)
	})

	t.Run("state after manual transition", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		err := r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)

		state := r.GetState()
		assert.Equal(t, finitestate.StatusBooting, state)
	})

	t.Run("state during run lifecycle", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		// Start runner in background
		runErrCh := make(chan error, 1)
		go func() {
			runErrCh <- r.Run(h.ctx)
		}()

		// Check initial state is New
		assert.Equal(t, finitestate.StatusNew, r.GetState())

		// Wait for state to become Running
		require.Eventually(t, func() bool {
			return r.GetState() == finitestate.StatusRunning
		}, 1*time.Second, 10*time.Millisecond, "Runner should reach Running state")

		// Stop and wait for state changes
		r.Stop()

		// Wait for Run to complete
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Run did not complete in time")
		}

		// Final state should be Stopped
		assert.Equal(t, finitestate.StatusStopped, r.GetState())
	})
}

func TestRunner_IsRunning(t *testing.T) {
	t.Run("not running initially", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		assert.False(t, r.IsRunning())
	})

	t.Run("not running in booting state", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		err := r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)

		assert.False(t, r.IsRunning())
	})

	t.Run("running after transition to running state", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()
		assert.True(t, r.IsRunning())
	})

	t.Run("not running in stopping state", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()
		require.True(t, r.IsRunning())

		err := r.fsm.Transition(finitestate.StatusStopping)
		require.NoError(t, err)

		assert.False(t, r.IsRunning())
	})

	t.Run("not running in stopped state", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		h.transitionToRunning()
		err := r.fsm.Transition(finitestate.StatusStopping)
		require.NoError(t, err)
		err = r.fsm.Transition(finitestate.StatusStopped)
		require.NoError(t, err)

		assert.False(t, r.IsRunning())
	})

	t.Run("running state during full lifecycle", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		// Initially not running
		assert.False(t, r.IsRunning())

		// Start runner in background
		runErrCh := make(chan error, 1)
		go func() {
			runErrCh <- r.Run(h.ctx)
		}()

		// Wait for runner to become running
		require.Eventually(t, func() bool {
			return r.IsRunning()
		}, 1*time.Second, 10*time.Millisecond, "Runner should be running")

		// Stop the runner
		r.Stop()

		// Wait for runner to stop being running
		require.Eventually(t, func() bool {
			return !r.IsRunning()
		}, 200*time.Millisecond, 10*time.Millisecond, "Runner should stop running")

		// Wait for Run to complete
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Run did not complete in time")
		}

		// Finally not running
		assert.False(t, r.IsRunning())
	})
}

func TestRunner_GetStateChan(t *testing.T) {
	t.Run("channel receives state changes", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()
		stateChan := r.GetStateChan(stateCtx)

		// Channel should initially have the current state
		select {
		case state := <-stateChan:
			assert.Equal(t, finitestate.StatusNew, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive initial state")
		}

		// Transition to booting and check channel
		err := r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)

		select {
		case state := <-stateChan:
			assert.Equal(t, finitestate.StatusBooting, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive booting state")
		}

		// Transition to running and check channel
		err = r.fsm.Transition(finitestate.StatusRunning)
		require.NoError(t, err)

		select {
		case state := <-stateChan:
			assert.Equal(t, finitestate.StatusRunning, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive running state")
		}
	})

	t.Run("channel closed when context cancelled", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		ctx, cancel := context.WithCancel(t.Context())
		stateChan := r.GetStateChan(ctx)

		// Read initial state
		select {
		case state := <-stateChan:
			assert.Equal(t, finitestate.StatusNew, state)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive initial state")
		}

		// Cancel context
		cancel()

		// Channel should be closed
		require.Eventually(t, func() bool {
			select {
			case _, ok := <-stateChan:
				return !ok // Channel should be closed
			case <-time.After(10 * time.Millisecond):
				return false
			}
		}, 100*time.Millisecond, 10*time.Millisecond, "Channel should be closed")
	})

	t.Run("multiple channels receive same state changes", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		ctx := t.Context()
		stateChan1 := r.GetStateChan(ctx)
		stateChan2 := r.GetStateChan(ctx)

		// Both channels should receive initial state
		for i, ch := range []<-chan string{stateChan1, stateChan2} {
			select {
			case state := <-ch:
				assert.Equal(
					t,
					finitestate.StatusNew,
					state,
					"Channel %d should receive initial state",
					i+1,
				)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Channel %d should receive initial state", i+1)
			}
		}

		// Transition and check both channels
		err := r.fsm.Transition(finitestate.StatusBooting)
		require.NoError(t, err)

		for i, ch := range []<-chan string{stateChan1, stateChan2} {
			select {
			case state := <-ch:
				assert.Equal(
					t,
					finitestate.StatusBooting,
					state,
					"Channel %d should receive booting state",
					i+1,
				)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Channel %d should receive booting state", i+1)
			}
		}
	})

	t.Run("state changes during full run lifecycle", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner
		defer h.cancel()

		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()
		stateChan := r.GetStateChan(stateCtx)

		// Collect states in background
		var stateHistory []string
		var stateHistoryMutex sync.Mutex
		go func() {
			for state := range stateChan {
				stateHistoryMutex.Lock()
				stateHistory = append(stateHistory, state)
				stateHistoryMutex.Unlock()
			}
		}()

		// Start runner
		runErrCh := make(chan error, 1)
		go func() {
			runErrCh <- r.Run(h.ctx)
		}()

		// Wait for runner to start
		require.Eventually(t, func() bool {
			return r.IsRunning()
		}, 1*time.Second, 10*time.Millisecond, "Runner should be running")

		// Stop the runner
		r.Stop()

		// Wait for Run to complete
		select {
		case err := <-runErrCh:
			require.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Run did not complete in time")
		}

		// Wait for all 5 expected states to be collected
		require.Eventually(t, func() bool {
			stateHistoryMutex.Lock()
			defer stateHistoryMutex.Unlock()
			return len(stateHistory) == 5
		}, 1*time.Second, 10*time.Millisecond, "Should collect all 5 states")

		// Cancel state context to close the channel
		stateCancel()

		// Check exact state sequence
		stateHistoryMutex.Lock()
		defer stateHistoryMutex.Unlock()
		expected := []string{
			finitestate.StatusNew,
			finitestate.StatusBooting,
			finitestate.StatusRunning,
			finitestate.StatusStopping,
			finitestate.StatusStopped,
		}
		require.Equal(t, expected, stateHistory)
	})
}

func TestRunner_StateConsistency(t *testing.T) {
	t.Run("GetState and IsRunning consistency", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		// Test all states for consistency
		states := []string{
			finitestate.StatusNew,
			finitestate.StatusBooting,
			finitestate.StatusRunning,
			finitestate.StatusStopping,
			finitestate.StatusStopped,
			finitestate.StatusError,
		}

		for _, expectedState := range states {
			err := r.fsm.SetState(expectedState)
			require.NoError(t, err)

			actualState := r.GetState()
			assert.Equal(t, expectedState, actualState)

			expectedRunning := (expectedState == finitestate.StatusRunning)
			actualRunning := r.IsRunning()
			assert.Equal(t, expectedRunning, actualRunning,
				"IsRunning() should return %t when state is %s", expectedRunning, expectedState)
		}
	})

	t.Run("GetStateChan and GetState consistency", func(t *testing.T) {
		h := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
		r := h.runner

		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()
		stateChan := r.GetStateChan(stateCtx)

		// Read initial state from channel
		select {
		case chanState := <-stateChan:
			directState := r.GetState()
			assert.Equal(t, directState, chanState, "Channel state should match GetState()")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should receive initial state from channel")
		}

		// Test consistency after state transitions
		states := []string{
			finitestate.StatusBooting,
			finitestate.StatusRunning,
			finitestate.StatusStopping,
		}
		for _, expectedState := range states {
			err := r.fsm.Transition(expectedState)
			require.NoError(t, err)

			select {
			case chanState := <-stateChan:
				directState := r.GetState()
				assert.Equal(t, directState, chanState,
					"Channel state should match GetState() after transition to %s", expectedState)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Should receive state %s from channel", expectedState)
			}
		}
	})
}

// stateTestHarness provides utilities for testing state transitions with real lifecycle
type stateTestHarness struct {
	*runnerTestHarness
	runErrCh chan error
}

func newStateTestHarness(t *testing.T) *stateTestHarness {
	t.Helper()
	base := newRunnerTestHarness(t, testutil.GetRandomListeningPort(t))
	return &stateTestHarness{
		runnerTestHarness: base,
		runErrCh:          make(chan error, 1),
	}
}

func (h *stateTestHarness) startRunner() {
	go func() {
		h.runErrCh <- h.runner.Run(h.ctx)
	}()
}

func (h *stateTestHarness) waitForRunningState() {
	require.Eventually(h.t, func() bool {
		return h.runner.IsRunning()
	}, 1*time.Second, 10*time.Millisecond, "Runner should reach Running state")
}

func (h *stateTestHarness) stopAndWaitForCompletion() {
	h.runner.Stop()
	select {
	case err := <-h.runErrCh:
		require.NoError(h.t, err)
	case <-time.After(200 * time.Millisecond):
		h.t.Fatal("Run did not complete in time")
	}
}

func TestRunner_StateIntegration(t *testing.T) {
	t.Run("complete lifecycle with state monitoring", func(t *testing.T) {
		h := newStateTestHarness(t)
		defer h.cancel()

		r := h.runner
		stateCtx, stateCancel := context.WithCancel(t.Context())
		defer stateCancel()

		// Monitor state changes
		stateChan := r.GetStateChan(stateCtx)
		stateLog := make([]string, 0)
		var stateLogMutex sync.Mutex

		go func() {
			for state := range stateChan {
				stateLogMutex.Lock()
				stateLog = append(stateLog, state)
				stateLogMutex.Unlock()
			}
		}()

		// Initial state verification
		assert.Equal(t, finitestate.StatusNew, r.GetState())
		assert.False(t, r.IsRunning())

		// Start the runner
		h.startRunner()

		// Wait for running state
		h.waitForRunningState()
		assert.Equal(t, finitestate.StatusRunning, r.GetState())
		assert.True(t, r.IsRunning())

		// Stop and wait for completion
		h.stopAndWaitForCompletion()

		// Final state verification
		assert.Equal(t, finitestate.StatusStopped, r.GetState())
		assert.False(t, r.IsRunning())

		// Wait for all 5 expected states to be collected
		require.Eventually(t, func() bool {
			stateLogMutex.Lock()
			defer stateLogMutex.Unlock()
			return len(stateLog) == 5
		}, 1*time.Second, 10*time.Millisecond, "Should collect all 5 states")

		// Cancel state context to close the channel
		stateCancel()

		// Verify exact state sequence
		stateLogMutex.Lock()
		defer stateLogMutex.Unlock()
		expected := []string{
			finitestate.StatusNew,
			finitestate.StatusBooting,
			finitestate.StatusRunning,
			finitestate.StatusStopping,
			finitestate.StatusStopped,
		}
		require.Equal(t, expected, stateLog)
	})
}
