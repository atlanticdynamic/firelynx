package script

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	extism "github.com/robbyt/go-polyscript/engines/extism"
	risor "github.com/robbyt/go-polyscript/engines/risor"
	starlark "github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
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

// createPolyscriptEvaluator creates and compiles a go-polyscript evaluator from domain config
// This loads the script content (from code or URI) and compiles it for execution
func createPolyscriptEvaluator(
	config *scripts.AppScript,
	logger *slog.Logger,
) (platform.Evaluator, error) {
	handler := logger.Handler()

	switch e := config.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		loader, err := createLoaderFromEvaluator(e.Code, e.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to create risor loader: %w", err)
		}

		// Compile the Risor script with static data
		if config.StaticData != nil {
			return risor.FromRisorLoaderWithData(handler, loader, config.StaticData.Data)
		}
		return risor.FromRisorLoader(handler, loader)

	case *evaluators.StarlarkEvaluator:
		loader, err := createLoaderFromEvaluator(e.Code, e.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to create starlark loader: %w", err)
		}

		// Compile the Starlark script with static data
		if config.StaticData != nil {
			return starlark.FromStarlarkLoaderWithData(handler, loader, config.StaticData.Data)
		}
		return starlark.FromStarlarkLoader(handler, loader)

	case *evaluators.ExtismEvaluator:
		loader, err := createLoaderFromEvaluator(e.Code, e.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to create extism loader: %w", err)
		}

		// Compile the WASM module with static data
		if config.StaticData != nil {
			return extism.FromExtismLoaderWithData(handler, loader, config.StaticData.Data, e.Entrypoint)
		}
		return extism.FromExtismLoader(handler, loader, e.Entrypoint)

	default:
		return nil, fmt.Errorf("unsupported evaluator type: %T", e)
	}
}

// createLoaderFromEvaluator creates a go-polyscript loader from code or URI
func createLoaderFromEvaluator(code, uri string) (loader.Loader, error) {
	if code != "" {
		return loader.NewFromString(code)
	} else if uri != "" {
		return createLoaderFromURI(uri)
	} else {
		return nil, fmt.Errorf("evaluator must have either code or uri")
	}
}

// createLoaderFromURI creates a go-polyscript Loader from a URI
func createLoaderFromURI(uri string) (loader.Loader, error) {
	if strings.HasPrefix(uri, "file://") {
		return loader.NewFromDisk(uri)
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return loader.NewFromHTTP(uri)
	} else {
		return nil, fmt.Errorf("unsupported URI scheme: %s", uri)
	}
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
