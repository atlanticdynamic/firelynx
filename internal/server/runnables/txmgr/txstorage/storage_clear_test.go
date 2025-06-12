package txstorage

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_Clear(t *testing.T) {
	tests := []struct {
		name              string
		setupTxs          []string // states of transactions to create
		keepLast          int
		expectedCleared   int
		expectedRemaining int
		description       string
	}{
		{
			name:              "clear with keep last 0",
			setupTxs:          []string{"invalid", "error", "invalid", "error"},
			keepLast:          0,
			expectedCleared:   4,
			expectedRemaining: 0,
			description:       "should clear all terminal transactions when keepLast=0",
		},
		{
			name:              "clear with keep last 2",
			setupTxs:          []string{"invalid", "error", "invalid", "error"},
			keepLast:          2,
			expectedCleared:   2,
			expectedRemaining: 2,
			description:       "should keep last 2 transactions and clear the rest",
		},
		{
			name:              "clear with non-terminal transactions",
			setupTxs:          []string{"validating", "invalid", "executing", "error"},
			keepLast:          2,
			expectedCleared:   2, // total=4, keepLast=2, so delete 2 terminal transactions
			expectedRemaining: 2, // 2 non-terminal remain (terminal ones deleted)
			description:       "should delete terminal transactions to reach keepLast total",
		},
		{
			name:              "clear when no transactions exist",
			setupTxs:          []string{},
			keepLast:          5,
			expectedCleared:   0,
			expectedRemaining: 0,
			description:       "should return 0 cleared when no transactions exist",
		},
		{
			name:              "clear when fewer transactions than keepLast",
			setupTxs:          []string{"invalid", "error"},
			keepLast:          5,
			expectedCleared:   0,
			expectedRemaining: 2,
			description:       "should not clear anything when total < keepLast",
		},
		{
			name:              "clear with all non-terminal transactions",
			setupTxs:          []string{"validating", "executing", "created"},
			keepLast:          1,
			expectedCleared:   0,
			expectedRemaining: 3,
			description:       "should not clear any non-terminal transactions",
		},
		{
			name: "clear mixed terminal and non-terminal",
			setupTxs: []string{
				"invalid",
				"validating",
				"error",
				"executing",
				"invalid",
				"error",
			},
			keepLast:          3,
			expectedCleared:   3, // total=6, keepLast=3, so delete 3 terminal transactions
			expectedRemaining: 3, // 2 non-terminal + 1 terminal kept
			description:       "should delete terminal transactions to reach keepLast total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()

			// Create test transactions with specified states
			for i, state := range tt.setupTxs {
				tx := createTestTransactionWithState(t, state)
				err := storage.Add(tx)
				require.NoError(t, err, "failed to add transaction %d", i)
			}

			// Perform the clear operation
			cleared, err := storage.Clear(tt.keepLast)

			// Verify results
			require.NoError(t, err, "Clear should not return an error")
			assert.Equal(
				t,
				tt.expectedCleared,
				cleared,
				"unexpected number of cleared transactions",
			)

			remaining := len(storage.GetAll())
			assert.Equal(
				t,
				tt.expectedRemaining,
				remaining,
				"unexpected number of remaining transactions",
			)

			t.Logf("%s: cleared=%d, remaining=%d", tt.description, cleared, remaining)
		})
	}
}

func TestMemoryStorage_Clear_InvalidKeepLast(t *testing.T) {
	storage := NewMemoryStorage()

	// Add some test transactions
	tx := createTestTransactionWithState(t, "invalid")
	err := storage.Add(tx)
	require.NoError(t, err)

	// Test negative keepLast
	cleared, err := storage.Clear(-1)
	assert.Error(t, err, "should return error for negative keepLast")
	assert.Equal(t, 0, cleared, "should not clear any transactions on error")
	assert.Contains(t, err.Error(), "keepLast must be non-negative")
}

func TestMemoryStorage_Clear_PreservesOrder(t *testing.T) {
	storage := NewMemoryStorage()

	// Create transactions in specific order
	states := []string{"invalid", "error", "invalid", "error"}
	var originalTxs []*transaction.ConfigTransaction

	for _, state := range states {
		tx := createTestTransactionWithState(t, state)
		originalTxs = append(originalTxs, tx)
		err := storage.Add(tx)
		require.NoError(t, err)
	}

	// Clear keeping last 2
	cleared, err := storage.Clear(2)
	require.NoError(t, err)
	assert.Equal(t, 2, cleared)

	// Check that the remaining transactions are the last 2 in original order
	remaining := storage.GetAll()
	assert.Len(t, remaining, 2)

	// Should be the last 2 transactions from the original list
	assert.Equal(t, originalTxs[2].ID, remaining[0].ID, "first remaining should be 3rd original")
	assert.Equal(t, originalTxs[3].ID, remaining[1].ID, "second remaining should be 4th original")
}

// createTestTransactionWithState creates a test transaction with the specified state
func createTestTransactionWithState(t *testing.T, state string) *transaction.ConfigTransaction {
	t.Helper()

	// Create minimal config for the transaction
	cfg := &config.Config{
		Version: config.VersionLatest,
	}

	// Create transaction - it starts in "created" state
	tx, err := transaction.New(
		transaction.SourceTest,
		"test-detail",
		"test-request-id",
		cfg,
		nil, // handler
	)
	require.NoError(t, err, "failed to create test transaction")

	// Transition to desired state by calling appropriate methods
	switch state {
	case "created":
		// Already in created state, nothing to do
	case "validating":
		err = tx.BeginValidation()
		require.NoError(t, err)
	case "validated":
		err = tx.BeginValidation()
		require.NoError(t, err)
		// Set valid flag so MarkValidated succeeds
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
	case "executing":
		err = tx.BeginValidation()
		require.NoError(t, err)
		tx.IsValid.Store(true)
		err = tx.MarkValidated()
		require.NoError(t, err)
		err = tx.BeginExecution()
		require.NoError(t, err)
	case "invalid":
		err = tx.BeginValidation()
		require.NoError(t, err)
		err = tx.MarkInvalid(assert.AnError)
		require.NoError(t, err)
	case "error":
		err = tx.MarkError(assert.AnError)
		require.NoError(t, err)
	default:
		t.Fatalf("unknown state: %s", state)
	}

	return tx
}
