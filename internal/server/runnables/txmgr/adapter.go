// Package txmgr provides transaction management for configuration changes.
//
// HTTP Listener Rewrite Plan:
// According to the HTTP listener rewrite plan, HTTP-specific configuration logic
// has been moved to the HTTP listener package. Each SagaParticipant implements its own
// configuration extraction, keeping this package focused on orchestrating the configuration
// transaction process rather than handling HTTP-specific details.
package txmgr

import (
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// ConfigAdapter converts domain config to package-specific configs for runtime components.
// This is the only component that should have knowledge of domain config types.
type ConfigAdapter struct {
	domainConfig  *config.Config
	appCollection apps.Registry
	logger        *slog.Logger
}

// NewConfigAdapter creates a new adapter for converting domain config to runtime configs.
func NewConfigAdapter(
	domainConfig *config.Config,
	appCollection apps.Registry,
	logger *slog.Logger,
) *ConfigAdapter {
	if logger == nil {
		logger = slog.Default().WithGroup("core.ConfigAdapter")
	}

	return &ConfigAdapter{
		domainConfig:  domainConfig,
		appCollection: appCollection,
		logger:        logger,
	}
}

// GetDomainConfig returns the current domain configuration being used by this adapter.
func (a *ConfigAdapter) GetDomainConfig() *config.Config {
	return a.domainConfig
}

// GetAppRegistry returns the app registry being used by this adapter.
func (a *ConfigAdapter) GetAppRegistry() apps.Registry {
	return a.appCollection
}

// SetDomainConfig updates the domain config used by this adapter.
func (a *ConfigAdapter) SetDomainConfig(domainConfig *config.Config) {
	a.domainConfig = domainConfig
}

// ValidateConfig performs basic validation on the domain config.
// This is a placeholder for more comprehensive validation that could be added later.
func (a *ConfigAdapter) ValidateConfig() error {
	if a.domainConfig == nil {
		return fmt.Errorf("domain config is nil")
	}

	// Log domain config state for debugging
	a.logger.Debug("Validating domain config",
		"endpoints", len(a.domainConfig.Endpoints),
		"listeners", len(a.domainConfig.Listeners),
		"apps", len(a.domainConfig.Apps))

	return nil
}
