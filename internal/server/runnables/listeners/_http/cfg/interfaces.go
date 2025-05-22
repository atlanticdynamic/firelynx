package cfg

import (
	"github.com/atlanticdynamic/firelynx/internal/config"
)

// ConfigProvider is a local interface that defines the minimum requirements
// for accessing configuration from a transaction.
type ConfigProvider interface {
	// GetTransactionID returns the unique identifier for this transaction
	GetTransactionID() string

	// GetConfig returns the configuration associated with this transaction
	GetConfig() *config.Config
}
