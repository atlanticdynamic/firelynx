package transaction

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*ConfigTransaction, slog.Handler) {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	cfg := &config.Config{}

	tx, err := FromTest("test_transaction", cfg, handler)
	require.NoError(t, err)
	require.NotNil(t, tx)

	return tx, handler
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates new transaction with correct initial state", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.CurrentState())
		assert.Equal(t, SourceTest, tx.Source)
		assert.Equal(t, "test_transaction", tx.SourceDetail)
		assert.NotEmpty(t, tx.ID)
		assert.NotNil(t, tx.Logger)
		assert.NotNil(t, tx.LogCollector)
		assert.False(t, tx.IsValid)
		assert.Empty(t, tx.ValidationErrors)
	})
}

func TestTransactionLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("validates happy path lifecycle", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.CurrentState())

		require.NoError(t, tx.BeginValidation())
		assert.Equal(t, finitestate.StateValidating, tx.CurrentState())

		require.NoError(t, tx.MarkValid())
		assert.Equal(t, finitestate.StateValidated, tx.CurrentState())
		assert.True(t, tx.IsValid)

		require.NoError(t, tx.BeginPreparation())
		assert.Equal(t, finitestate.StatePreparing, tx.CurrentState())

		require.NoError(t, tx.MarkPrepared())
		assert.Equal(t, finitestate.StatePrepared, tx.CurrentState())

		require.NoError(t, tx.BeginCommit())
		assert.Equal(t, finitestate.StateCommitting, tx.CurrentState())

		require.NoError(t, tx.MarkCommitted())
		assert.Equal(t, finitestate.StateCommitted, tx.CurrentState())

		require.NoError(t, tx.MarkCompleted())
		assert.Equal(t, finitestate.StateCompleted, tx.CurrentState())
	})

	t.Run("validates failed validation path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.BeginValidation())

		validationErrs := []error{
			errors.New("validation error 1"),
			errors.New("validation error 2"),
		}

		require.NoError(t, tx.MarkInvalid(validationErrs))
		assert.Equal(t, finitestate.StateInvalid, tx.CurrentState())
		assert.False(t, tx.IsValid)
		assert.Len(t, tx.ValidationErrors, 2)

		err := tx.BeginPreparation()
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("validates rollback path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.BeginValidation())
		require.NoError(t, tx.MarkValid())
		require.NoError(t, tx.BeginPreparation())
		require.NoError(t, tx.MarkPrepared())

		require.NoError(t, tx.BeginRollback())
		assert.Equal(t, finitestate.StateRollingBack, tx.CurrentState())

		require.NoError(t, tx.MarkRolledBack())
		assert.Equal(t, finitestate.StateRolledBack, tx.CurrentState())
	})

	t.Run("validates failure path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.BeginValidation())
		require.NoError(t, tx.MarkValid())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(testErr))
		assert.Equal(t, finitestate.StateFailed, tx.CurrentState())
		assert.Contains(t, tx.ValidationErrors, testErr)
	})
}

func TestConstructors(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(os.Stdout, nil)
	cfg := &config.Config{}

	t.Run("constructs from file", func(t *testing.T) {
		tx, err := FromFile("testdata/config.toml", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceFile, tx.Source)
		assert.Contains(t, tx.SourceDetail, "testdata/config.toml")
	})

	t.Run("constructs from API", func(t *testing.T) {
		tx, err := FromAPI("req-123", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceAPI, tx.Source)
		assert.Equal(t, "gRPC API", tx.SourceDetail)
		assert.Equal(t, "req-123", tx.RequestID)
	})

	t.Run("constructs from test", func(t *testing.T) {
		tx, err := FromTest("unit_test", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceTest, tx.Source)
		assert.Equal(t, "unit_test", tx.SourceDetail)
	})
}

func TestLogCollection(t *testing.T) {
	t.Parallel()

	t.Run("collects and plays back logs", func(t *testing.T) {
		tx, handler := setupTest(t)

		require.NoError(t, tx.BeginValidation())
		require.NoError(t, tx.MarkValid())
		require.NoError(t, tx.BeginPreparation())
		require.NoError(t, tx.MarkPrepared())
		require.NoError(t, tx.BeginCommit())
		require.NoError(t, tx.MarkCommitted())
		require.NoError(t, tx.MarkCompleted())

		err := tx.PlaybackLogs(handler)
		require.NoError(t, err)
	})
}

func TestGetDuration(t *testing.T) {
	t.Parallel()

	t.Run("reports transaction duration", func(t *testing.T) {
		tx, _ := setupTest(t)

		time.Sleep(10 * time.Millisecond)

		duration := tx.GetTotalDuration()
		assert.Greater(t, duration, 0*time.Millisecond)
	})
}
