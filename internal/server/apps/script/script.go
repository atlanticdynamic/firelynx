package script

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"mime"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
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
	evaluator, err := getPolyscriptEvaluator(config)
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
	routeStaticData map[string]any,
) error {
	timeout := s.config.Evaluator.GetTimeout()
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	runtimeData := s.prepareRuntimeData(r, routeStaticData)

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

// prepareRuntimeData creates runtime data for script execution following go-polyscript patterns
func (s *ScriptApp) prepareRuntimeData(
	r *http.Request,
	routeStaticData map[string]any,
) map[string]any {
	// Start with the HTTP request for contextProvider to process
	runtimeData := map[string]any{
		"request": r,
	}

	// Add app-level static data from config
	if s.config.StaticData != nil {
		maps.Copy(runtimeData, s.config.StaticData.Data)
	}

	// Merge route-level static data (from endpoint config)
	maps.Copy(runtimeData, routeStaticData)

	// Extract JSON body fields for direct access by scripts (especially Extism)
	// We read the body twice: once for JSON parsing, once for go-polyscript's contextProvider
	// The contextProvider calls helpers.RequestToMap() which reads r.Body to create a "Body" field
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err == nil {
		switch mediaType {
		case "application/json":
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				s.logger.Error("Failed to read request body", "error", err)
				return runtimeData // Return empty data if body read fails
			}

			r.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Reset for go-polyscript to read
			var bodyData map[string]any
			if json.Unmarshal(bodyBytes, &bodyData) == nil {
				maps.Copy(runtimeData, bodyData)
			}
		}
	}

	return runtimeData
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
