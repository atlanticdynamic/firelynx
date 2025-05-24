// Package cfg provides configuration management for the HTTP listener.
package cfg

import (
	"log/slog"
	"sync"
)

// Manager provides thread-safe access to current and pending HTTP configurations.
// It stores adapter instances that represent configurations at different stages
// of the transaction lifecycle.
type Manager struct {
	current *Adapter     // Current active configuration
	pending *Adapter     // Pending configuration being prepared
	mutex   sync.RWMutex // Mutex for thread-safe access
	logger  *slog.Logger // Logger
}

// NewManager creates a new configuration manager with the provided logger.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default().WithGroup("http.ConfigManager")
	}

	return &Manager{
		logger: logger,
	}
}

// SetPending stores a new adapter as the pending configuration.
// This is called during the prepare phase of a saga transaction.
func (m *Manager) SetPending(adapter *Adapter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if adapter != nil {
		m.logger.Debug("Setting pending HTTP configuration", "tx_id", adapter.TxID)
	}

	m.pending = adapter
}

// GetPending returns the current pending adapter, if any.
// Returns nil if no pending configuration exists.
func (m *Manager) GetPending() *Adapter {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.pending
}

// CommitPending commits the pending adapter as the current adapter.
// This is called during the commit phase of a transaction after all
// participants have successfully prepared.
func (m *Manager) CommitPending() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.pending != nil {
		listenerCount := len(m.pending.Listeners)
		routeCount := 0
		for _, routes := range m.pending.Routes {
			routeCount += len(routes)
		}

		m.logger.Debug("Committing pending HTTP configuration",
			"tx_id", m.pending.TxID,
			"listener_count", listenerCount,
			"route_count", routeCount)

		m.current = m.pending
		m.pending = nil
	}
}

// RollbackPending discards the pending adapter.
// This is called during the rollback phase of a transaction when
// one or more participants have failed to prepare.
func (m *Manager) RollbackPending() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.pending != nil {
		m.logger.Debug("Rolling back pending HTTP configuration",
			"tx_id", m.pending.TxID)
		m.pending = nil
	}
}

// GetCurrent returns the current adapter.
// Returns nil if no configuration has been committed yet.
func (m *Manager) GetCurrent() *Adapter {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.current
}

// HasPendingChanges returns true if there is a pending configuration
// waiting to be committed.
func (m *Manager) HasPendingChanges() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.pending != nil
}
