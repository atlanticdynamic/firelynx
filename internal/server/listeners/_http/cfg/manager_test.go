package cfg

import (
	"log/slog"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	// Test creating a manager with a nil logger
	manager := NewManager(nil)
	assert.NotNil(t, manager)
	assert.Nil(t, manager.current)
	assert.Nil(t, manager.pending)

	// Test creating a manager with a logger
	logger := slog.Default()
	manager = NewManager(logger)
	assert.NotNil(t, manager)
	assert.Equal(t, logger, manager.logger)
}

func TestManager_SetPending(t *testing.T) {
	// Create a manager
	manager := NewManager(nil)

	// Create a test adapter
	adapter := &Adapter{TxID: "tx-123"}

	// Test setting the pending adapter
	manager.SetPending(adapter)
	assert.Equal(t, adapter, manager.pending)
	assert.Nil(t, manager.current)
}

func TestManager_CommitPending(t *testing.T) {
	// Create a manager
	manager := NewManager(nil)

	// Set up a pending adapter
	pending := &Adapter{
		TxID:      "tx-123",
		Listeners: map[string]*http.ListenerConfig{"listener-1": nil},
		Routes:    map[string][]httpserver.Route{"listener-1": {}},
	}
	manager.SetPending(pending)

	// Test committing the pending adapter
	manager.CommitPending()
	assert.Equal(t, pending, manager.current)
	assert.Nil(t, manager.pending)

	// Test committing when there is no pending adapter
	manager.CommitPending()
	assert.Equal(t, pending, manager.current) // Current should remain unchanged
}

func TestManager_RollbackPending(t *testing.T) {
	// Create a manager
	manager := NewManager(nil)

	// Set up current and pending adapters
	current := &Adapter{TxID: "tx-123"}
	pending := &Adapter{TxID: "tx-456"}
	manager.current = current
	manager.pending = pending

	// Test rolling back the pending adapter
	manager.RollbackPending()
	assert.Equal(t, current, manager.current)
	assert.Nil(t, manager.pending)

	// Test rolling back when there is no pending adapter
	manager.RollbackPending()
	assert.Equal(t, current, manager.current) // Current should remain unchanged
}

func TestManager_GetCurrentOrPending(t *testing.T) {
	// Create a manager
	manager := NewManager(nil)

	// Test when both current and pending are nil
	adapter := manager.GetCurrentOrPending()
	assert.Nil(t, adapter)

	// Set up only current adapter
	current := &Adapter{TxID: "tx-123"}
	manager.current = current

	// Test when only current exists
	adapter = manager.GetCurrentOrPending()
	assert.Equal(t, current, adapter)

	// Set up pending adapter
	pending := &Adapter{TxID: "tx-456"}
	manager.pending = pending

	// Test when both current and pending exist
	adapter = manager.GetCurrentOrPending()
	assert.Equal(t, pending, adapter) // Should return pending
}

func TestManager_GetCurrent(t *testing.T) {
	// Create a manager
	manager := NewManager(nil)

	// Test when current is nil
	adapter := manager.GetCurrent()
	assert.Nil(t, adapter)

	// Set up current adapter
	current := &Adapter{TxID: "tx-123"}
	manager.current = current

	// Test getting the current adapter
	adapter = manager.GetCurrent()
	assert.Equal(t, current, adapter)

	// Set up pending adapter - should not affect GetCurrent
	pending := &Adapter{TxID: "tx-456"}
	manager.pending = pending

	// Test that GetCurrent still returns current, not pending
	adapter = manager.GetCurrent()
	assert.Equal(t, current, adapter)
}
