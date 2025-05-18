package cfg

import (
	"log/slog"
	"sync"
)

// Manager manages current and pending HTTP configuration adapters.
// It provides thread-safe access to adapters during the transaction lifecycle.
type Manager struct {
	current *Adapter
	pending *Adapter
	mutex   sync.RWMutex
	logger  *slog.Logger
}

// NewManager creates a new adapter manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		logger: logger,
	}
}

// SetPending sets the pending adapter. This is called during the
// prepare phase of a transaction.
func (m *Manager) SetPending(adapter *Adapter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.pending = adapter
}

// CommitPending commits the pending adapter as the current adapter.
// This is called during the commit phase of a transaction.
func (m *Manager) CommitPending() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.pending != nil {
		listenerCount := len(m.pending.Listeners)
		routeCount := 0
		for _, routes := range m.pending.Routes {
			routeCount += len(routes)
		}

		m.logger.Debug("Committing pending HTTP adapter",
			"tx_id", m.pending.TxID,
			"listener_count", listenerCount,
			"route_count", routeCount)
		m.current = m.pending
		m.pending = nil
	}
}

// RollbackPending discards the pending adapter. This is called during
// the rollback phase of a transaction.
func (m *Manager) RollbackPending() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.pending != nil {
		m.logger.Debug("Rolling back pending HTTP adapter", "tx_id", m.pending.TxID)
		m.pending = nil
	}
}

// GetCurrentOrPending returns the pending adapter if there is one,
// otherwise returns the current adapter. This is used to get the
// configuration for rendering responses.
func (m *Manager) GetCurrentOrPending() *Adapter {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.pending != nil {
		return m.pending
	}
	return m.current
}

// GetCurrent returns the current adapter.
func (m *Manager) GetCurrent() *Adapter {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.current
}
