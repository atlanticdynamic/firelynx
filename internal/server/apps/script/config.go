package script

import (
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/platform"
)

// Config contains everything needed to instantiate a script app.
// This is a Data Transfer Object (DTO) with no dependencies on domain packages.
// All validation and resource compilation happens at the domain layer before creating this config.
type Config struct {
	// ID is the unique identifier for this app instance
	ID string

	// CompiledEvaluator is the pre-compiled script evaluator from domain validation.
	// This evaluator is ready to execute and contains all compiled resources.
	CompiledEvaluator platform.Evaluator

	// StaticData contains pre-processed static data from the domain configuration.
	// This data is embedded during domain validation for runtime use.
	StaticData map[string]any

	// Logger is the structured logger configured for this app instance
	Logger *slog.Logger

	// Timeout is the maximum execution time for script evaluation
	Timeout time.Duration
}
