package transaction

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForCompletion(t *testing.T) {
	t.Run("returns immediately when already in terminal state", func(t *testing.T) {
		terminalStates := []string{
			finitestate.StateCompleted,
			finitestate.StateCompensated,
			finitestate.StateError,
			finitestate.StateInvalid,
		}

		for _, terminalState := range terminalStates {
			t.Run(terminalState, func(t *testing.T) {
				tx, _ := setupTest(t)
				ctx := t.Context()

				// Force transaction into terminal state
				switch terminalState {
				case finitestate.StateCompleted:
					require.NoError(t, tx.BeginValidation())
					tx.IsValid.Store(true)
					require.NoError(t, tx.MarkValidated())
					require.NoError(t, tx.BeginExecution())
					require.NoError(t, tx.MarkSucceeded())
					require.NoError(t, tx.BeginReload())
					require.NoError(t, tx.MarkCompleted())
				case finitestate.StateCompensated:
					require.NoError(t, tx.BeginValidation())
					tx.IsValid.Store(true)
					require.NoError(t, tx.MarkValidated())
					require.NoError(t, tx.BeginExecution())
					require.NoError(t, tx.MarkFailed(ctx, errors.New("test error")))
					require.NoError(t, tx.BeginCompensation())
					require.NoError(t, tx.MarkCompensated())
				case finitestate.StateError:
					require.NoError(t, tx.MarkError(errors.New("test error")))
				case finitestate.StateInvalid:
					require.NoError(t, tx.BeginValidation())
					require.NoError(t, tx.MarkInvalid(errors.New("validation failed")))
				}

				// WaitForCompletion should return immediately
				start := time.Now()
				err := tx.WaitForCompletion(ctx)
				duration := time.Since(start)

				assert.NoError(t, err)
				assert.Less(
					t,
					duration,
					10*time.Millisecond,
					"Should return immediately for terminal state",
				)
				assert.Equal(t, terminalState, tx.GetState())
			})
		}
	})

	t.Run("waits for transaction to reach terminal state", func(t *testing.T) {
		tx, _ := setupTest(t)
		ctx := t.Context()

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Start WaitForCompletion in goroutine
		waitDone := make(chan error, 1)
		waitStarted := make(chan struct{})

		go func() {
			close(waitStarted)
			waitDone <- tx.WaitForCompletion(ctx)
		}()

		// Wait for goroutine to start
		<-waitStarted

		// Verify it's waiting (not completed yet)
		select {
		case err := <-waitDone:
			t.Fatalf("WaitForCompletion returned too early: %v", err)
		case <-time.After(50 * time.Millisecond):
			// Good, it's waiting
		}

		// Complete the transaction
		require.NoError(t, tx.MarkSucceeded())
		require.NoError(t, tx.BeginReload())
		require.NoError(t, tx.MarkCompleted())

		// Now WaitForCompletion should return
		assert.Eventually(t, func() bool {
			select {
			case err := <-waitDone:
				assert.NoError(t, err)
				return true
			default:
				return false
			}
		}, 1*time.Second, 10*time.Millisecond, "WaitForCompletion should return after transaction completed")

		assert.Equal(t, finitestate.StateCompleted, tx.GetState())
	})

	t.Run("returns context error when context is cancelled", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Create cancellable context
		ctx, cancel := context.WithCancel(t.Context())

		// Start WaitForCompletion in goroutine
		waitDone := make(chan error, 1)
		waitStarted := make(chan struct{})

		go func() {
			close(waitStarted)
			waitDone <- tx.WaitForCompletion(ctx)
		}()

		// Wait for goroutine to start
		<-waitStarted

		// Verify it's waiting
		assert.Never(t, func() bool {
			select {
			case <-waitDone:
				return true
			default:
				return false
			}
		}, 50*time.Millisecond, 10*time.Millisecond, "Should be waiting")

		// Cancel context
		cancel()

		// WaitForCompletion should return with context.Canceled
		assert.Eventually(t, func() bool {
			select {
			case err := <-waitDone:
				assert.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
				return true
			default:
				return false
			}
		}, 1*time.Second, 10*time.Millisecond, "WaitForCompletion should return after context cancellation")
	})

	t.Run("returns context error when context times out", func(t *testing.T) {
		tx, _ := setupTest(t)

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// WaitForCompletion should return with context.DeadlineExceeded
		err := tx.WaitForCompletion(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("handles multiple concurrent waiters", func(t *testing.T) {
		tx, _ := setupTest(t)
		ctx := t.Context()

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Start multiple waiters
		const numWaiters = 5
		waitDone := make(chan error, numWaiters)
		var wg sync.WaitGroup
		waitersStarted := make(chan struct{})

		for i := 0; i < numWaiters; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Signal when all waiters have started
				if i == numWaiters-1 {
					close(waitersStarted)
				}
				waitDone <- tx.WaitForCompletion(ctx)
			}()
		}

		// Wait for all waiters to start
		<-waitersStarted

		// Ensure waiters are waiting
		assert.Never(t, func() bool {
			select {
			case <-waitDone:
				return true
			default:
				return false
			}
		}, 50*time.Millisecond, 10*time.Millisecond, "Waiters should be waiting")

		// Complete the transaction
		require.NoError(t, tx.MarkSucceeded())
		require.NoError(t, tx.BeginReload())
		require.NoError(t, tx.MarkCompleted())

		// Wait for all waiters to complete
		wg.Wait()

		// Verify all waiters completed successfully
		assert.Eventually(t, func() bool {
			for i := 0; i < numWaiters; i++ {
				select {
				case err := <-waitDone:
					assert.NoError(t, err, "Waiter %d should have completed successfully", i)
				default:
					return false
				}
			}
			return true
		}, 1*time.Second, 10*time.Millisecond, "All waiters should complete")
	})

	t.Run("works with different terminal state transitions", func(t *testing.T) {
		testCases := []struct {
			name          string
			setupStates   func(*ConfigTransaction) error
			expectedState string
		}{
			{
				name: "compensation path",
				setupStates: func(tx *ConfigTransaction) error {
					ctx := t.Context()
					if err := tx.BeginValidation(); err != nil {
						return err
					}
					tx.IsValid.Store(true)
					if err := tx.MarkValidated(); err != nil {
						return err
					}
					if err := tx.BeginExecution(); err != nil {
						return err
					}
					if err := tx.MarkFailed(ctx, errors.New("test failure")); err != nil {
						return err
					}
					if err := tx.BeginCompensation(); err != nil {
						return err
					}
					return tx.MarkCompensated()
				},
				expectedState: finitestate.StateCompensated,
			},
			{
				name: "error path",
				setupStates: func(tx *ConfigTransaction) error {
					return tx.MarkError(errors.New("unrecoverable error"))
				},
				expectedState: finitestate.StateError,
			},
			{
				name: "invalid path",
				setupStates: func(tx *ConfigTransaction) error {
					if err := tx.BeginValidation(); err != nil {
						return err
					}
					return tx.MarkInvalid(errors.New("validation failed"))
				},
				expectedState: finitestate.StateInvalid,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tx, _ := setupTest(t)
				ctx := t.Context()

				// Start WaitForCompletion (transaction starts in "created" state)

				// Start WaitForCompletion
				waitDone := make(chan error, 1)
				waitStarted := make(chan struct{})

				go func() {
					close(waitStarted)
					waitDone <- tx.WaitForCompletion(ctx)
				}()

				// Wait for goroutine to start
				<-waitStarted

				// Let waiter establish itself
				assert.Never(t, func() bool {
					select {
					case <-waitDone:
						return true
					default:
						return false
					}
				}, 20*time.Millisecond, 5*time.Millisecond, "Should be waiting")

				// Transition to terminal state
				require.NoError(t, tc.setupStates(tx))

				// Verify WaitForCompletion returns
				assert.Eventually(t, func() bool {
					select {
					case err := <-waitDone:
						assert.NoError(t, err)
						return true
					default:
						return false
					}
				}, 1*time.Second, 10*time.Millisecond, "WaitForCompletion should return after reaching terminal state")

				assert.Equal(t, tc.expectedState, tx.GetState())
			})
		}
	})

	t.Run("handles closed state channel", func(t *testing.T) {
		tx, _ := setupTest(t)
		ctx := t.Context()

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Create a context that we'll cancel to simulate channel closure
		waitCtx, cancel := context.WithCancel(ctx)

		// Start WaitForCompletion
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- tx.WaitForCompletion(waitCtx)
		}()

		// Cancel context after a short delay to close the state channel
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// WaitForCompletion should handle the closed channel and return
		assert.Eventually(t, func() bool {
			select {
			case err := <-waitDone:
				// Should return context.Canceled error since we canceled the context
				assert.ErrorIs(t, err, context.Canceled)
				return true
			default:
				return false
			}
		}, 1*time.Second, 10*time.Millisecond, "WaitForCompletion should handle closed channel")
	})

	t.Run("concurrent state transitions and waiters", func(t *testing.T) {
		tx, _ := setupTest(t)
		ctx := t.Context()

		// Start transaction in non-terminal state
		require.NoError(t, tx.BeginValidation())
		tx.IsValid.Store(true)
		require.NoError(t, tx.MarkValidated())
		require.NoError(t, tx.BeginExecution())

		// Start multiple waiters and a state transition concurrently
		const numWaiters = 3
		waitResults := make([]error, numWaiters)
		var wg sync.WaitGroup

		// Start waiters
		for i := 0; i < numWaiters; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				waitResults[idx] = tx.WaitForCompletion(ctx)
			}(i)
		}

		// Start state transition after a short delay
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Let waiters start
			time.Sleep(20 * time.Millisecond)

			// Complete the transaction
			assert.NoError(t, tx.MarkSucceeded())
			assert.NoError(t, tx.BeginReload())
			assert.NoError(t, tx.MarkCompleted())
		}()

		// Wait for all operations to complete
		wg.Wait()

		// Verify all waiters completed successfully
		for i, err := range waitResults {
			assert.NoError(t, err, "Waiter %d should have completed successfully", i)
		}
		assert.Equal(t, finitestate.StateCompleted, tx.GetState())
	})
}
