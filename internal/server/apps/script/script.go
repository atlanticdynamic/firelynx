package script

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/robbyt/go-polyscript/platform"
)

// ScriptApp implements the server-side script application using go-polyscript
type ScriptApp struct {
	id        string
	config    *scripts.AppScript
	evaluator platform.Evaluator
	logger    *slog.Logger
}

// New creates a new script app instance using go-polyscript
func New(id string, config *scripts.AppScript, logger *slog.Logger) (*ScriptApp, error) {
	if config == nil {
		return nil, fmt.Errorf("script app config cannot be nil")
	}

	// Validate that evaluator exists
	if config.Evaluator == nil {
		return nil, fmt.Errorf("script app must have an evaluator")
	}

	// Create and compile the go-polyscript evaluator at instantiation time
	evaluator, err := createPolyscriptEvaluator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create and compile go-polyscript evaluator: %w", err)
	}

	return &ScriptApp{
		id:        id,
		config:    config,
		evaluator: evaluator,
		logger: logger.With(
			"app_id",
			id,
			"app_type",
			"script",
			"evaluator_type",
			config.Evaluator.Type(),
		),
	}, nil
}

// String returns the unique identifier of the application
func (s *ScriptApp) String() string {
	return s.id
}

// HandleHTTP handles HTTP requests by executing the script using go-polyscript
func (s *ScriptApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	staticData map[string]any,
) error {
	timeout := getEvaluatorTimeout(s.config.Evaluator)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare runtime data with the actual request object at top level
	// According to go-polyscript data system, all data should be at top level of ctx
	runtimeData := map[string]any{
		"request": r, // Pass the actual *http.Request at top level
	}

	// Merge static data from config and runtime data
	if s.config.StaticData != nil {
		maps.Copy(runtimeData, s.config.StaticData.Data)
	}
	maps.Copy(runtimeData, staticData)

	enrichedCtx, err := s.evaluator.AddDataToContext(timeoutCtx, runtimeData)
	if err != nil {
		s.logger.Error("Failed to add runtime data", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}

	start := time.Now()
	result, err := s.evaluator.Eval(enrichedCtx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("Script execution failed",
			"error", err,
			"duration", duration,
		)

		if timeoutCtx.Err() == context.DeadlineExceeded {
			http.Error(w, "Script Execution Timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Script Execution Error", http.StatusInternalServerError)
		}
		return err
	}

	s.logger.Info("Script executed successfully", "duration", duration)

	if err := handleScriptResult(w, result); err != nil {
		s.logger.Error("Failed to handle script result", "error", err)
		http.Error(w, "Result Processing Error", http.StatusInternalServerError)
		return err
	}

	return nil
}

// createPolyscriptEvaluator gets the pre-compiled evaluator from domain validation
// All evaluators must be compiled during the Validate() phase in the domain layer
func createPolyscriptEvaluator(
	config *scripts.AppScript,
	logger *slog.Logger,
) (platform.Evaluator, error) {
	// All evaluators must be pre-compiled during domain validation
	compiledEvaluator := config.Evaluator.GetCompiledEvaluator()
	if compiledEvaluator == nil {
		return nil, fmt.Errorf(
			"evaluator not compiled during validation phase - this indicates a domain validation bug for evaluator type: %T",
			config.Evaluator,
		)
	}

	return compiledEvaluator, nil
}

// getEvaluatorTimeout gets timeout from evaluator, with fallback
func getEvaluatorTimeout(eval evaluators.Evaluator) time.Duration {
	switch e := eval.(type) {
	case *evaluators.RisorEvaluator:
		if e.Timeout > 0 {
			return e.Timeout
		}
		return 30 * time.Second
	case *evaluators.StarlarkEvaluator:
		if e.Timeout > 0 {
			return e.Timeout
		}
		return 30 * time.Second
	case *evaluators.ExtismEvaluator:
		if e.Timeout > 0 {
			return e.Timeout
		}
		return 60 * time.Second
	default:
		return 30 * time.Second
	}
}

// handleScriptResult processes the script execution result and writes the HTTP response
func handleScriptResult(w http.ResponseWriter, result platform.EvaluatorResponse) error {
	value := result.Interface()

	switch v := value.(type) {
	case map[string]any:
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(v)

	case string:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err := w.Write([]byte(v))
		return err

	case []byte:
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err := w.Write(v)
		return err

	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err := fmt.Fprintf(w, "%v", v)
		return err
	}
}
