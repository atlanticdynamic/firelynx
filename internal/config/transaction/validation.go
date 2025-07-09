package transaction

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	httpCfg "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
)

type appFactory interface {
	CreateAppsFromDefinitions(defs []serverApps.AppDefinition) (*serverApps.AppInstances, error)
}

type middlewareFactory interface {
	CreateFromDefinitions(middleware.MiddlewareCollection) (*httpCfg.MiddlewareCollection, error)
}

// RunValidation performs comprehensive validation of the entire transaction
func (tx *ConfigTransaction) RunValidation() error {
	logger := tx.logger.WithGroup("validation")

	// Handle nil config case
	if tx.domainConfig == nil {
		tx.setStateInvalid([]error{ErrNilConfig})
		return fmt.Errorf("%w: %w", ErrValidationFailed, ErrNilConfig)
	}
	if tx.domainConfig.ValidationCompleted {
		logger.Warn("Configuration validation already completed, running again...")
	}

	// Transition to validating state
	err := tx.fsm.Transition(finitestate.StateValidating)
	if err != nil {
		logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateValidating,
			"currentState", tx.GetState())
		return err
	}
	logger.Debug("Validation started", "state", finitestate.StateValidating)

	var validationErrors []error

	// 1. Validate domain config structure
	if err := tx.domainConfig.Validate(); err != nil {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("domain config validation failed: %w", err),
		)
	}

	// 2. Validate and instantiate apps
	if err := validateAndCreateApps(tx); err != nil {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("app instantiation validation failed: %w", err),
		)
	}

	// 3. Validate and instantiate middleware
	if err := validateAndCreateMiddleware(tx); err != nil {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("middleware instantiation validation failed: %w", err),
		)
	}

	// 4. Validate resource conflicts
	if err := validateAllResourceConflicts(tx); err != nil {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("resource conflict validation failed: %w", err),
		)
	}

	// If any validation failed, mark as invalid
	if len(validationErrors) > 0 {
		combinedErr := errors.Join(validationErrors...)
		tx.setStateInvalid(validationErrors)
		return fmt.Errorf("%w: %w", ErrValidationFailed, combinedErr)
	}

	// All validation passed
	tx.setStateValid()
	return nil
}

// setStateValid marks the transaction as valid after successful validation
func (tx *ConfigTransaction) setStateValid() {
	logger := tx.logger.WithGroup("validation")
	err := tx.fsm.Transition(finitestate.StateValidated)
	if err != nil {
		logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateValidated,
			"currentState", tx.GetState(),
		)
		return
	}

	tx.IsValid.Store(true)
	logger.Debug(
		"Validation successful",
		"state", finitestate.StateValidated)
}

// setStateInvalid marks the transaction as invalid after failed validation
func (tx *ConfigTransaction) setStateInvalid(errs []error) {
	logger := tx.logger.WithGroup("validation")
	err := tx.fsm.Transition(finitestate.StateInvalid)
	if err != nil {
		logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateInvalid,
			"currentState", tx.GetState())
		return
	}

	tx.IsValid.Store(false)

	logger.Debug(
		"Validation failed",
		"errors", errs,
		"errorCount", len(errs),
		"state", finitestate.StateInvalid)
}

// collectApps extracts app definitions from domain config, including expanded apps from routes
func collectApps(cfg *config.Config) []serverApps.AppDefinition {
	// First collect unique apps from routes (these have merged static data)
	uniqueApps := make(map[string]apps.App)

	// Collect expanded apps from routes
	for _, endpoint := range cfg.Endpoints {
		for _, route := range endpoint.Routes {
			if route.App != nil {
				// Use the expanded app instance which has merged static data
				uniqueApps[route.App.ID] = *route.App
			}
		}
	}

	// Add any apps that don't have routes (not expanded)
	for _, app := range cfg.Apps {
		if _, exists := uniqueApps[app.ID]; !exists {
			uniqueApps[app.ID] = app
		}
	}

	// Convert to app definitions
	definitions := make([]serverApps.AppDefinition, 0, len(uniqueApps))
	for _, app := range uniqueApps {
		definitions = append(definitions, serverApps.AppDefinition{
			ID:     app.ID,
			Config: app.Config,
		})
	}

	return definitions
}

// collectMiddlewares extracts middleware collection from domain config
func collectMiddlewares(cfg *config.Config) middleware.MiddlewareCollection {
	var allMiddlewares middleware.MiddlewareCollection
	for _, endpoint := range cfg.Endpoints {
		allMiddlewares = allMiddlewares.Merge(endpoint.Middlewares)
		for _, route := range endpoint.Routes {
			allMiddlewares = allMiddlewares.Merge(route.Middlewares)
		}
	}
	return allMiddlewares
}

// validateAndCreateApps validates and creates app instances
func validateAndCreateApps(tx *ConfigTransaction) error {
	definitions := collectApps(tx.domainConfig)
	appCollection, err := tx.app.factory.CreateAppsFromDefinitions(definitions)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAppCreationFailed, err)
	}
	tx.app.collection = appCollection
	return nil
}

// validateAndCreateMiddleware validates middleware configs and creates instances
func validateAndCreateMiddleware(tx *ConfigTransaction) error {
	allMiddlewares := collectMiddlewares(tx.domainConfig)

	if err := allMiddlewares.Validate(); err != nil {
		return fmt.Errorf("middleware config validation failed: %w", err)
	}

	middlewareCollection, err := tx.middleware.factory.CreateFromDefinitions(allMiddlewares)
	if err != nil {
		return fmt.Errorf("middleware instantiation failed: %w", err)
	}
	tx.middleware.collection = middlewareCollection
	return nil
}

// validateAllResourceConflicts validates that resources don't conflict
func validateAllResourceConflicts(tx *ConfigTransaction) error {
	allMiddlewares := collectMiddlewares(tx.domainConfig)
	return validateResourceConflicts(allMiddlewares)
}
