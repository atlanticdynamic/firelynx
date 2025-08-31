package script

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
)

// ScriptApp implements the server-side script application using go-polyscript
type ScriptApp struct {
	id                string
	evaluator         platform.Evaluator
	appStaticProvider data.Provider // Pre-created app-level static provider
	logger            *slog.Logger
	timeout           time.Duration
}

// New creates a new script app instance from a Config DTO
func New(cfg *Config) (*ScriptApp, error) {
	if cfg == nil {
		return nil, fmt.Errorf("script app config cannot be nil")
	}

	// Validate that evaluator exists (should be pre-compiled from domain validation)
	if cfg.CompiledEvaluator == nil {
		return nil, fmt.Errorf("script app must have a compiled evaluator")
	}

	// Pre-create app-level static provider for performance
	appStaticProvider := data.NewStaticProvider(cfg.StaticData)

	return &ScriptApp{
		id:                cfg.ID,
		evaluator:         cfg.CompiledEvaluator,
		appStaticProvider: appStaticProvider,
		logger:            cfg.Logger,
		timeout:           cfg.Timeout,
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
	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
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
	scriptData := map[string]any{
		"data":    maps.Clone(mergedStaticData),
		"request": r,
	}
	return scriptData, nil
}

// This function is no longer needed since evaluators are pre-compiled
// and passed through the Config DTO

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
