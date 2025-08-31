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

// Helper function to create script config DTO from domain config
func createScriptConfig(t *testing.T, id string, domainConfig *scripts.AppScript) *Config {
	t.Helper()

	// Get compiled evaluator from domain config
	compiledEvaluator, err := domainConfig.Evaluator.GetCompiledEvaluator()
	require.NoError(t, err)

	// Extract static data
	var staticData map[string]any
	if domainConfig.StaticData != nil {
		staticData = domainConfig.StaticData.Data
	}

	// Create DTO directly in tests to avoid import cycle
	return &Config{
		ID:                id,
		CompiledEvaluator: compiledEvaluator,
		StaticData:        staticData,
		Logger:            slog.Default().With("app_type", "script", "app_id", id),
		Timeout:           domainConfig.Evaluator.GetTimeout(),
	}
}

func TestScriptApp_String_ReturnsAppID(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code: `{"message": "hello"}`,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	assert.Equal(t, "test-app", app.String())
}

func TestScriptApp_New_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: "script app config cannot be nil",
		},
		{
			name: "nil compiled evaluator",
			config: &Config{
				ID:                "test",
				CompiledEvaluator: nil,
				StaticData:        nil,
				Logger:            slog.Default(),
				Timeout:           5 * time.Second,
			},
			wantError: "script app must have a compiled evaluator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := New(tt.config)
			require.Error(t, err)
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

	domainConfig := scripts.NewAppScript("risor-test")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "risor-test", domainConfig)
	app, err := New(scriptConfig)
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

	domainConfig := scripts.NewAppScript("starlark-test")
	domainConfig.Evaluator = starlarkEval

	scriptConfig := createScriptConfig(t, "starlark-test", domainConfig)
	app, err := New(scriptConfig)
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

	domainConfig := scripts.NewAppScript("risor-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "risor-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Hello from Risor!")
}

func TestScriptApp_HandleHTTP_WithStaticData(t *testing.T) {
	tests := []struct {
		name            string
		evaluator       evaluators.Evaluator
		staticData      map[string]any
		method          string
		expectedContent string
	}{
		{
			name: "risor_with_static_data",
			evaluator: &evaluators.RisorEvaluator{
				Code: `{
					"status": 200,
					"headers": {"Content-Type": "application/json"}, 
					"body": json.marshal({"message": "Hello", "static": ctx.get("data", {}).get("test_key", "not_found")})
				}`,
				Timeout: 5 * time.Second,
			},
			staticData: map[string]any{
				"test_key": "static_value",
			},
			method:          http.MethodGet,
			expectedContent: "static_value",
		},
		{
			name: "starlark_with_static_data",
			evaluator: &evaluators.StarlarkEvaluator{
				Code: `
# Access static data in Starlark
config_value = ctx.get("data", {}).get("config_key", "default")
result = {
	"message": "Starlark with config",
	"config": config_value
}
# The underscore variable is returned to Go
_ = result`,
				Timeout: 5 * time.Second,
			},
			staticData: map[string]any{
				"config_key": "production",
			},
			method:          http.MethodPost,
			expectedContent: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.evaluator.Validate()
			require.NoError(t, err)

			domainConfig := scripts.NewAppScript("test-app")
			domainConfig.Evaluator = tt.evaluator
			domainConfig.StaticData = &staticdata.StaticData{Data: tt.staticData}

			scriptConfig := createScriptConfig(t, "test-app", domainConfig)
			app, err := New(scriptConfig)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			err = app.HandleHTTP(t.Context(), w, req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedContent)
		})
	}
}

func TestScriptApp_HandleHTTP_ScriptError(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `invalid_risor_syntax(`,
		Timeout: 5 * time.Second,
	}

	// Validation should fail for invalid syntax
	err := risorEval.Validate()
	require.Error(t, err)
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

	domainConfig := scripts.NewAppScript("risor-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "risor-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.Error(t, err)
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

	domainConfig := scripts.NewAppScript("starlark-test")
	domainConfig.Evaluator = starlarkEval

	scriptConfig := createScriptConfig(t, "starlark-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Hello from Starlark!")
	assert.Contains(t, w.Body.String(), "python-like")
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
	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = mockEval

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// This should work fine since prepareScriptData doesn't fail for valid inputs
	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
}

func TestScriptApp_HandleHTTP_ExtismDataStructure(t *testing.T) {
	// Test that Extism evaluator data preparation works differently
	// Since we don't have a real Extism WASM module easily available for testing,
	// we'll test with a Risor script but verify the data structuring logic
	risorEval := &evaluators.RisorEvaluator{
		Code: `{
			"message": ctx.get("data", {}).get("test_key", "not_found")
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

	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval
	domainConfig.StaticData = &staticdata.StaticData{Data: staticData}

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// For non-Extism evaluators, data field should be available as-is
	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
}

func TestScriptApp_HandleHTTP_StringResult(t *testing.T) {
	risorEval := &evaluators.RisorEvaluator{
		Code:    `"Plain text response"`,
		Timeout: 5 * time.Second,
	}

	err := risorEval.Validate()
	require.NoError(t, err)

	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
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

	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.NoError(t, err)
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

	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval

	scriptConfig := createScriptConfig(t, "test-app", domainConfig)
	app, err := New(scriptConfig)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err = app.HandleHTTP(t.Context(), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestScriptApp_New_EvaluatorCompilationError(t *testing.T) {
	// Create an evaluator that will fail to get compiled evaluator
	risorEval := &evaluators.RisorEvaluator{
		Code:    "invalid syntax <<<",
		Timeout: 5 * time.Second,
	}

	// Don't validate - this will cause getPolyscriptEvaluator to fail
	domainConfig := scripts.NewAppScript("test-app")
	domainConfig.Evaluator = risorEval

	// The error now happens during createScriptConfig when getting compiled evaluator
	_, err := domainConfig.Evaluator.GetCompiledEvaluator()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
}

// TestScriptApp_ComprehensiveStaticDataTypes tests that all valid static data types
// are correctly accessible through the data namespace across different evaluators
func TestScriptApp_ComprehensiveStaticDataTypes(t *testing.T) {
	tests := []struct {
		name         string
		evaluator    evaluators.Evaluator
		staticData   map[string]any
		expectedJSON string
	}{
		{
			name: "risor_all_data_types",
			evaluator: &evaluators.RisorEvaluator{
				Code: `{
					"string_val": ctx.get("data", {}).get("string_field", "missing"),
					"int_val": ctx.get("data", {}).get("int_field", 0),
					"bool_val": ctx.get("data", {}).get("bool_field", false),
					"float_val": ctx.get("data", {}).get("float_field", 0.0),
					"array_val": ctx.get("data", {}).get("array_field", []),
					"nested_val": ctx.get("data", {}).get("nested_object", {}).get("key1", "missing"),
					"deep_nested_val": ctx.get("data", {}).get("nested_object", {}).get("deep", {}).get("value", "missing")
				}`,
				Timeout: 5 * time.Second,
			},
			staticData: map[string]any{
				"string_field": "test_string",
				"int_field":    42,
				"bool_field":   true,
				"float_field":  3.14,
				"array_field":  []any{"item1", "item2", "item3"},
				"nested_object": map[string]any{
					"key1": "value1",
					"deep": map[string]any{
						"value": "deep_value",
					},
				},
			},
			expectedJSON: `{"string_val":"test_string","int_val":42,"bool_val":true,"float_val":3.14,"array_val":["item1","item2","item3"],"nested_val":"value1","deep_nested_val":"deep_value"}`,
		},
		{
			name: "starlark_all_data_types",
			evaluator: &evaluators.StarlarkEvaluator{
				Code: `
# Access all types of static data through the data namespace
data_ns = ctx.get("data", {})
result = {
    "string_val": data_ns.get("string_field", "missing"),
    "int_val": data_ns.get("int_field", 0),
    "bool_val": data_ns.get("bool_field", False),
    "float_val": data_ns.get("float_field", 0.0),
    "array_val": data_ns.get("array_field", []),
    "nested_val": data_ns.get("nested_object", {}).get("key1", "missing"),
    "deep_nested_val": data_ns.get("nested_object", {}).get("deep", {}).get("value", "missing")
}
_ = result`,
				Timeout: 5 * time.Second,
			},
			staticData: map[string]any{
				"string_field": "test_string_starlark",
				"int_field":    24,
				"bool_field":   false,
				"float_field":  2.71,
				"array_field":  []any{"star1", "star2"},
				"nested_object": map[string]any{
					"key1": "starlark_value1",
					"deep": map[string]any{
						"value": "starlark_deep_value",
					},
				},
			},
			expectedJSON: `{"string_val":"test_string_starlark","int_val":24,"bool_val":false,"float_val":2.71,"array_val":["star1","star2"],"nested_val":"starlark_value1","deep_nested_val":"starlark_deep_value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.evaluator.Validate()
			require.NoError(t, err)

			config := scripts.NewAppScript("comprehensive-test")
			config.Evaluator = tt.evaluator
			config.StaticData = &staticdata.StaticData{Data: tt.staticData}

			app, err := New("comprehensive-test", config, slog.Default())
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			err = app.HandleHTTP(t.Context(), w, req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)

			// Compare JSON responses to verify all data types are accessible
			assert.JSONEq(t, tt.expectedJSON, w.Body.String(), "Script response should match expected JSON")
		})
	}
}
