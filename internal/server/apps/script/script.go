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
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
)

// ScriptApp implements the server-side script application using go-polyscript
type ScriptApp struct {
	id                string
	config            *scripts.AppScript
	evaluator         platform.Evaluator
	appStaticProvider data.Provider // Pre-created app-level static provider
	logger            *slog.Logger
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
	evaluator, err := getPolyscriptEvaluator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create and compile go-polyscript evaluator: %w", err)
	}

	// Pre-create app-level static provider for performance
	var appStaticData map[string]any
	if config.StaticData != nil {
		appStaticData = config.StaticData.Data
	}
	appStaticProvider := data.NewStaticProvider(appStaticData)

	return &ScriptApp{
		id:                id,
		config:            config,
		evaluator:         evaluator,
		appStaticProvider: appStaticProvider,
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
) error {
	timeout := s.config.Evaluator.GetTimeout()
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare script data with proper structure for WASM modules
	scriptData, err := s.prepareScriptData(timeoutCtx, r)
	if err != nil {
		s.logger.Error("Failed to prepare script data", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}

	// Create context provider and add all merged data to context
	contextProvider := data.NewContextProvider(constants.EvalData)
	enrichedCtx, err := contextProvider.AddDataToContext(timeoutCtx, scriptData)
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
			return err
		}

		http.Error(w, "Script Execution Error", http.StatusInternalServerError)
		return err
	}

	s.logger.Debug("Script executed successfully", "duration", duration)

	if err := handleScriptResult(w, result); err != nil {
		s.logger.Error("Failed to handle script result", "error", err)
		http.Error(w, "Result Processing Error", http.StatusInternalServerError)
		return err
	}

	return nil
}

// prepareScriptData prepares data for script execution, structuring it appropriately
// for the target script's expected format based on the evaluator type
func (s *ScriptApp) prepareScriptData(
	ctx context.Context,
	r *http.Request,
) (map[string]any, error) {
	// Get app-level static data (route static data is now embedded during app creation)
	appStaticData, err := s.appStaticProvider.GetData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get app static data: %w", err)
	}

	// Use app static data which now includes merged route data from app creation time
	mergedStaticData := maps.Clone(appStaticData)

	// All evaluators now use consistent namespaced structure
	scriptData := maps.Clone(mergedStaticData)
	scriptData["request"] = r
	return scriptData, nil
}

// getPolyscriptEvaluator extracts the pre-compiled evaluator from domain validation
// All evaluators must be compiled during the Validate() phase in the domain layer
func getPolyscriptEvaluator(
	config *scripts.AppScript,
) (platform.Evaluator, error) {
	compiledEvaluator, err := config.Evaluator.GetCompiledEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to get compiled evaluator: %w", err)
	}

	return compiledEvaluator, nil
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
