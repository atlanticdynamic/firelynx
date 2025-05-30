package cfg

import (
	"log/slog"
	"testing"

	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
)

func TestManager_NewManager(t *testing.T) {
	// Create manager with nil logger
	manager := NewManager(nil)
	assert.NotNil(t, manager, "Manager should not be nil when created with nil logger")
	assert.Nil(t, manager.current, "New manager should have nil current adapter")
	assert.Nil(t, manager.pending, "New manager should have nil pending adapter")

	// Create manager with provided logger
	logger := slog.Default()
	manager = NewManager(logger)
	assert.NotNil(t, manager, "Manager should not be nil when created with logger")
	assert.Nil(t, manager.current, "New manager should have nil current adapter")
	assert.Nil(t, manager.pending, "New manager should have nil pending adapter")
}

func TestManager_SetPending(t *testing.T) {
	// Create test adapter
	adapter := &Adapter{
		TxID:      "test-tx-id",
		Listeners: make(map[string]ListenerConfig),
		Routes:    make(map[string][]httpserver.Route),
	}

	// Create manager
	manager := NewManager(nil)

	// Set pending adapter
	manager.SetPending(adapter)
	assert.Equal(t, adapter, manager.pending, "Pending adapter should match what was set")
	assert.Nil(t, manager.current, "Current adapter should remain nil after setting pending")

	// Set pending to nil
	manager.SetPending(nil)
	assert.Nil(t, manager.pending, "Pending adapter should be nil after setting to nil")
}

func TestManager_CommitPending(t *testing.T) {
	// Create test adapter
	adapter := &Adapter{
		TxID:      "test-tx-id",
		Listeners: make(map[string]ListenerConfig),
		Routes:    make(map[string][]httpserver.Route),
	}

	// Create manager
	manager := NewManager(nil)

	// Commit with nil pending should not change anything
	manager.CommitPending()
	assert.Nil(
		t,
		manager.current,
		"Current adapter should remain nil when committing with nil pending",
	)

	// Set pending adapter
	manager.SetPending(adapter)

	// Commit pending
	manager.CommitPending()
	assert.Equal(t, adapter, manager.current, "Current adapter should match what was committed")
	assert.Nil(t, manager.pending, "Pending adapter should be nil after committing")
}

func TestManager_RollbackPending(t *testing.T) {
	// Create test adapter
	adapter := &Adapter{
		TxID:      "test-tx-id",
		Listeners: make(map[string]ListenerConfig),
		Routes:    make(map[string][]httpserver.Route),
	}

	// Create manager
	manager := NewManager(nil)

	// Rollback with nil pending should not change anything
	manager.RollbackPending()
	assert.Nil(
		t,
		manager.pending,
		"Pending adapter should remain nil when rolling back with nil pending",
	)

	// Set pending adapter
	manager.SetPending(adapter)

	// Rollback pending
	manager.RollbackPending()
	assert.Nil(t, manager.pending, "Pending adapter should be nil after rolling back")
	assert.Nil(t, manager.current, "Current adapter should remain nil after rolling back")
}

func TestManager_GettersAndHasPendingChanges(t *testing.T) {
	// Create test adapter
	adapter := &Adapter{
		TxID:      "test-tx-id",
		Listeners: make(map[string]ListenerConfig),
		Routes:    make(map[string][]httpserver.Route),
	}

	// Create manager
	manager := NewManager(nil)

	// Initial state
	assert.Nil(t, manager.GetCurrent(), "GetCurrent should return nil for new manager")
	assert.Nil(t, manager.GetPending(), "GetPending should return nil for new manager")
	assert.False(
		t,
		manager.HasPendingChanges(),
		"HasPendingChanges should return false for new manager",
	)

	// Set pending adapter
	manager.SetPending(adapter)
	assert.Nil(t, manager.GetCurrent(), "GetCurrent should still return nil after setting pending")
	assert.Equal(t, adapter, manager.GetPending(), "GetPending should return the pending adapter")
	assert.True(
		t,
		manager.HasPendingChanges(),
		"HasPendingChanges should return true after setting pending",
	)

	// Commit pending
	manager.CommitPending()
	assert.Equal(t, adapter, manager.GetCurrent(), "GetCurrent should return the committed adapter")
	assert.Nil(t, manager.GetPending(), "GetPending should return nil after committing")
	assert.False(
		t,
		manager.HasPendingChanges(),
		"HasPendingChanges should return false after committing",
	)
}
