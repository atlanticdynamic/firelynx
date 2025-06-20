package cfg

import (
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// ConfigProvider defines the minimal interface required to extract configuration
// from a transaction. This allows components to depend on this interface rather
// than the full transaction type.
type ConfigProvider interface {
	// GetTransactionID returns the unique identifier for this transaction
	GetTransactionID() string

	// GetConfig returns the configuration associated with this transaction
	GetConfig() *config.Config

	// GetAppCollection returns the app collection for linking routes to app instances.
	// Returns nil if no app collection is available.
	GetAppCollection() apps.AppLookup

	// GetMiddlewarePool returns the middleware pool for reusing middleware instances.
	// Structure: map[middleware-type]map[middleware-id]middleware-instance
	GetMiddlewarePool() MiddlewarePool
}
