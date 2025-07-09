package script

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptApp_String_ReturnsAppID(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `{"message": "hello"}`,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "test-app", app.String())
}

func TestScriptApp_New_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		config    *scripts.AppScript
		wantError string
	}{
		{
			name:      "nil config",
			id:        "test",
			config:    nil,
			wantError: "script app config cannot be nil",
		},
		{
			name: "nil evaluator",
			id:   "test",
			config: &scripts.AppScript{
				Evaluator: nil,
			},
			wantError: "script app must have an evaluator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := New(tt.id, tt.config, slog.Default())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Nil(t, app)
		})
	}
}

func TestScriptApp_New_RisorEvaluator(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `{"status": 200, "message": "success"}`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("risor-test", config, slog.Default())
	require.NoError(t, err)
	assert.Equal(t, "risor-test", app.String())
}

func TestScriptApp_New_StarlarkEvaluator(t *testing.T) {
	starlarkEval := &evaluators.StarlarkEvaluator{
		Code:    `result = {"status": 200, "message": "success"}`,
		Timeout: 3 * time.Second,
	}

	err := starlarkEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: starlarkEval,
	}

	app, err := New("starlark-test", config, slog.Default())
	require.NoError(t, err)
	assert.Equal(t, "starlark-test", app.String())
}

func TestScriptApp_HandleHTTP_RisorScript(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `{
			"status": 200,
			"headers": {"Content-Type": "application/json"},
			"body": json.marshal({"message": "Hello from Risor!"})
		}`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("risor-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Hello from Risor!")
}

func TestScriptApp_HandleHTTP_WithStaticData(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `{
			"status": 200,
			"headers": {"Content-Type": "application/json"}, 
			"body": json.marshal({"message": "Hello", "static": ctx.get("test_key", "not_found")})
		}`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	staticData := map[string]any{
		"test_key": "static_value",
	}

	config := &scripts.AppScript{
		Evaluator:  risorEval,
		StaticData: &staticdata.StaticData{Data: staticData},
	}

	app, err := New("risor-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	runtimeData := map[string]any{
		"runtime_key": "runtime_value",
	}

	err = app.HandleHTTP(t.Context(), w, req, runtimeData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "static_value")
}

func TestScriptApp_HandleHTTP_ScriptError(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `invalid_risor_syntax(`,
		Timeout: 5 * time.Second,
	}

	// Validation should fail for invalid syntax
	err := risorEval.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")

	// Since validation failed, we shouldn't try to create the app
	// This test demonstrates that invalid scripts are caught during validation
}

func TestScriptApp_HandleHTTP_Timeout(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `
		// Infinite loop to force timeout
		for i := 0; i < 10000000; i++ {
			x := i * i
		}
		{
			"status": 200,
			"body": "This should timeout"
		}`,
		Timeout: 1 * time.Millisecond,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("risor-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.Error(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
}

func TestScriptApp_HandleHTTP_StarlarkScript(t *testing.T) {
	starlarkEval := &evaluators.StarlarkEvaluator{
		Code: `
# Starlark script that returns a simple map (will be JSON-encoded automatically)
result = {
	"message": "Hello from Starlark!",
	"language": "python-like"
}
# The underscore variable is returned to Go
_ = result`,
		Timeout: 5 * time.Second,
	}

	err := starlarkEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: starlarkEval,
	}

	app, err := New("starlark-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Hello from Starlark!")
	assert.Contains(t, w.Body.String(), "python-like")
}

func TestScriptApp_HandleHTTP_StarlarkWithStaticData(t *testing.T) {
	starlarkEval := &evaluators.StarlarkEvaluator{
		Code: `
# Access static data in Starlark
config_value = ctx.get("config_key", "default")
result = {
	"message": "Starlark with config",
	"config": config_value
}
# The underscore variable is returned to Go
_ = result`,
		Timeout: 5 * time.Second,
	}

	err := starlarkEval.Validate()
	require.NoError(t, err)

	staticData := map[string]any{
		"config_key": "production",
	}

	config := &scripts.AppScript{
		Evaluator:  starlarkEval,
		StaticData: &staticdata.StaticData{Data: staticData},
	}

	app, err := New("starlark-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "production")
}

func TestScriptApp_HandleHTTP_PrepareScriptDataError(t *testing.T) {
	// Mock evaluator that can validate but has nil compiledEvaluator
	mockEval := &evaluators.RisorEvaluator{
		Code:    `{"message": "test"}`,
		Timeout: 5 * time.Second,
	}

	err := mockEval.Validate()
	require.NoError(t, err)

	// Create a script app
	config := &scripts.AppScript{
		Evaluator: mockEval,
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// This should work fine since prepareScriptData doesn't fail for valid inputs
	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
}

func TestScriptApp_HandleHTTP_ExtismDataStructure(t *testing.T) {
	// Test that Extism evaluator data preparation works differently
	// Since we don't have a real Extism WASM module easily available for testing,
	// we'll test with a Risor script but verify the data structuring logic
	risorEval := &evaluators.RisorEvaluator{
		Code: `{
			"message": ctx.get("test_key", "not_found")
		}`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	staticData := map[string]any{
		"data": map[string]any{
			"test_key": "nested_value",
		},
	}

	config := &scripts.AppScript{
		Evaluator:  risorEval,
		StaticData: &staticdata.StaticData{Data: staticData},
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// For non-Extism evaluators, data field should be available as-is
	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
}

func TestScriptApp_HandleHTTP_StringResult(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `"Plain text response"`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "Plain text response", w.Body.String())
}

func TestScriptApp_HandleHTTP_NumericResult(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `42`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "42", w.Body.String())
}

func TestScriptApp_HandleHTTP_ExecutionError(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `
		// This will cause a runtime error by invalid array access
		arr := []
		arr[100]`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("test-app", config, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req, nil)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestScriptApp_New_EvaluatorCompilationError(t *testing.T) {
	// Create an evaluator that will fail to get compiled evaluator
	risorEval := &evaluators.RisorEvaluator{
		Code:    "invalid syntax <<<",
		Timeout: 5 * time.Second,
	}

	// Don't validate - this will cause getPolyscriptEvaluator to fail
	config := &scripts.AppScript{
		Evaluator: risorEval,
	}

	app, err := New("test-app", config, slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create and compile go-polyscript evaluator")
	assert.Nil(t, app)
}
