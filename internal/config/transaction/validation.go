package transaction

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	httpCfg "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
)

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

// validateAndCreateApps validates and creates app instances using DTO pattern
func validateAndCreateApps(tx *ConfigTransaction) error {
	// Convert domain apps to DTOs and create instances directly
	appInstances, err := convertAndCreateApps(tx.domainConfig)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAppCreationFailed, err)
	}
	tx.app.collection = appInstances
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
