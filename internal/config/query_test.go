package config

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestConfig creates a test configuration with various listeners, endpoints, and apps
func setupTestConfig() *Config {
	// Create test listeners
	httpOptions := options.NewHTTP()

	httpListener := listeners.Listener{
		ID:      "http_listener",
		Address: "localhost:8080",
		Type:    listeners.TypeHTTP,
		Options: httpOptions,
	}
	httpListener2 := listeners.Listener{
		ID:      "http_listener_2",
		Address: "localhost:8081",
		Type:    listeners.TypeHTTP,
		Options: httpOptions,
	}

	// Create test apps
	echoApp := apps.App{
		ID:     "echo_app",
		Config: echo.New(),
	}
	// Update the response for this echo app
	echoApp.Config.(*echo.EchoApp).Response = "test response 1"

	// Create a script app with Risor evaluator
	risorEval := &evaluators.RisorEvaluator{
		Code:    "print('Hello, world')",
		Timeout: 5 * time.Second,
	}
	risorScriptApp := &scripts.AppScript{
		Evaluator: risorEval,
	}
	scriptApp := apps.App{
		ID:     "risor_app",
		Config: risorScriptApp,
	}

	// Create another script app with Starlark evaluator
	starlarkEval := &evaluators.StarlarkEvaluator{
		Code:    "print('Hello from Starlark')",
		Timeout: 5 * time.Second,
	}
	starlarkScriptApp := &scripts.AppScript{
		Evaluator: starlarkEval,
	}
	starScriptApp := apps.App{
		ID:     "starlark_app",
		Config: starlarkScriptApp,
	}

	// Create test endpoints and routes for HTTP
	httpCondition := conditions.NewHTTP("/test", "")
	httpRoute := routes.Route{
		AppID:     "echo_app",
		Condition: httpCondition,
	}
	httpEndpoint := endpoints.Endpoint{
		ID:         "http_endpoint",
		ListenerID: "http_listener",
		Routes:     []routes.Route{httpRoute},
	}

	// Create test endpoints and routes for second HTTP listener
	httpCondition2 := conditions.NewHTTP("/risor", "")
	httpRoute2 := routes.Route{
		AppID:     "risor_app",
		Condition: httpCondition2,
	}
	httpEndpoint2 := endpoints.Endpoint{
		ID:         "http_endpoint_2",
		ListenerID: "http_listener_2",
		Routes:     []routes.Route{httpRoute2},
	}

	// Create an endpoint attached to both listeners
	multiCondition := conditions.NewHTTP("/multi", "")
	multiRoute := routes.Route{
		AppID:     "starlark_app",
		Condition: multiCondition,
	}
	// Note: Since we've moved to single ListenerID, this test case needs to be modified
	// We'll create two separate endpoints for each listener instead of one with multiple listeners
	multiHttpEndpoint := endpoints.Endpoint{
		ID:         "multi_http_endpoint",
		ListenerID: "http_listener",
		Routes:     []routes.Route{multiRoute},
	}
	multiHttpEndpoint2 := endpoints.Endpoint{
		ID:         "multi_http_endpoint_2",
		ListenerID: "http_listener_2",
		Routes:     []routes.Route{multiRoute},
	}

	// Create the final config
	config := &Config{
		Version:   version.Version,
		Listeners: listeners.ListenerCollection{httpListener, httpListener2},
		Endpoints: endpoints.EndpointCollection{
			httpEndpoint,
			httpEndpoint2,
			multiHttpEndpoint,
			multiHttpEndpoint2,
		},
		Apps: apps.AppCollection{echoApp, scriptApp, starScriptApp},
	}

	return config
}

func TestGetListenerByID(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name           string
		listenerID     string
		expectedFound  bool
		expectedResult string
	}{
		{
			name:           "Find existing HTTP listener",
			listenerID:     "http_listener",
			expectedFound:  true,
			expectedResult: "http_listener",
		},
		{
			name:           "Find existing second HTTP listener",
			listenerID:     "http_listener_2",
			expectedFound:  true,
			expectedResult: "http_listener_2",
		},
		{
			name:          "Non-existent listener returns nil",
			listenerID:    "missing_listener",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetListenerByID(tt.listenerID)
			if tt.expectedFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult, result.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFindListener(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name           string
		listenerID     string
		expectedFound  bool
		expectedResult string
	}{
		{
			name:           "Find existing HTTP listener",
			listenerID:     "http_listener",
			expectedFound:  true,
			expectedResult: "http_listener",
		},
		{
			name:          "Non-existent listener returns nil",
			listenerID:    "missing_listener",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetListenerByID(tt.listenerID)
			if tt.expectedFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult, result.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestGetEndpointsForListener(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name          string
		listenerID    string
		expectedCount int
	}{
		{
			name:          "Get endpoints for HTTP listener",
			listenerID:    "http_listener",
			expectedCount: 2, // http_endpoint and multi_http_endpoint
		},
		{
			name:          "Get endpoints for second HTTP listener",
			listenerID:    "http_listener_2",
			expectedCount: 2, // http_endpoint_2 and multi_http_endpoint_2
		},
		{
			name:          "Non-existent listener returns empty list",
			listenerID:    "missing_listener",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetEndpointsForListener(tt.listenerID)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestFindEndpoint(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name           string
		endpointID     string
		expectedFound  bool
		expectedResult string
	}{
		{
			name:           "Find existing HTTP endpoint",
			endpointID:     "http_endpoint",
			expectedFound:  true,
			expectedResult: "http_endpoint",
		},
		{
			name:           "Find existing second HTTP endpoint",
			endpointID:     "http_endpoint_2",
			expectedFound:  true,
			expectedResult: "http_endpoint_2",
		},
		{
			name:          "Non-existent endpoint returns nil",
			endpointID:    "missing_endpoint",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetEndpointByID(tt.endpointID)
			if tt.expectedFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult, result.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestGetEndpointByID(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name           string
		endpointID     string
		expectedFound  bool
		expectedResult string
	}{
		{
			name:           "Find existing HTTP endpoint",
			endpointID:     "http_endpoint",
			expectedFound:  true,
			expectedResult: "http_endpoint",
		},
		{
			name:          "Non-existent endpoint returns nil",
			endpointID:    "missing_endpoint",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetEndpointByID(tt.endpointID)
			if tt.expectedFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult, result.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFindApp(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name           string
		appID          string
		expectedFound  bool
		expectedResult string
	}{
		{
			name:           "Find existing echo app",
			appID:          "echo_app",
			expectedFound:  true,
			expectedResult: "echo_app",
		},
		{
			name:           "Find existing risor app",
			appID:          "risor_app",
			expectedFound:  true,
			expectedResult: "risor_app",
		},
		{
			name:          "Non-existent app returns nil",
			appID:         "missing_app",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.FindApp(tt.appID)
			if tt.expectedFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult, result.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestGetAppsByType(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name          string
		evalType      string
		expectedCount int
	}{
		{
			name:          "Get Risor evaluator apps",
			evalType:      "Risor",
			expectedCount: 1,
		},
		{
			name:          "Get Starlark evaluator apps",
			evalType:      "Starlark",
			expectedCount: 1,
		},
		{
			name:          "Non-existent eval type returns empty list",
			evalType:      "missing_type",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetAppsByType(tt.evalType)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestGetListenersByType(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name          string
		listenerType  listeners.Type
		expectedCount int
	}{
		{
			name:          "Get HTTP type listeners",
			listenerType:  listeners.TypeHTTP,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetListenersByType(tt.listenerType)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestGetHTTPListeners(t *testing.T) {
	config := setupTestConfig()
	result := config.GetHTTPListeners()
	assert.Len(t, result, 2)
	// Check that we have both HTTP listeners
	ids := []string{result[0].ID, result[1].ID}
	assert.Contains(t, ids, "http_listener")
	assert.Contains(t, ids, "http_listener_2")
}

func TestGetEndpointsByListenerID(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name          string
		listenerID    string
		expectedCount int
	}{
		{
			name:          "Get endpoints for HTTP listener",
			listenerID:    "http_listener",
			expectedCount: 2, // http_endpoint and multi_http_endpoint
		},
		{
			name:          "Get endpoints for second HTTP listener",
			listenerID:    "http_listener_2",
			expectedCount: 2, // http_endpoint_2 and multi_http_endpoint_2
		},
		{
			name:          "Non-existent listener returns empty list",
			listenerID:    "missing_listener",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetEndpointsByListenerID(tt.listenerID)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestGetEndpointIDsForListener(t *testing.T) {
	config := setupTestConfig()

	tests := []struct {
		name          string
		listenerID    string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "Get endpoint IDs for HTTP listener",
			listenerID:    "http_listener",
			expectedCount: 2,
			expectedIDs:   []string{"http_endpoint", "multi_http_endpoint"},
		},
		{
			name:          "Get endpoint IDs for second HTTP listener",
			listenerID:    "http_listener_2",
			expectedCount: 2,
			expectedIDs:   []string{"http_endpoint_2", "multi_http_endpoint_2"},
		},
		{
			name:          "Non-existent listener returns empty list",
			listenerID:    "missing_listener",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetEndpointIDsForListener(tt.listenerID)
			assert.Len(t, result, tt.expectedCount)

			// Check that all expected IDs are in the result
			if tt.expectedCount > 0 {
				for _, id := range tt.expectedIDs {
					found := false
					for _, resultID := range result {
						if resultID == id {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find endpoint ID %s in results", id)
				}
			}
		})
	}
}

func TestGetEndpointToListenerIDMapping(t *testing.T) {
	t.Parallel()

	// Create a test configuration
	config := setupTestConfig()

	// Get the endpoint to listener ID mapping
	mapping := config.GetEndpointToListenerIDMapping()

	// Expected mapping based on the test config
	expected := map[string]string{
		"http_endpoint":         "http_listener",
		"multi_http_endpoint":   "http_listener",
		"http_endpoint_2":       "http_listener_2",
		"multi_http_endpoint_2": "http_listener_2",
	}

	// Verify the mapping
	assert.Equal(t, expected, mapping)

	// Test with an empty config
	emptyConfig := &Config{
		Endpoints: endpoints.EndpointCollection{},
	}
	emptyMapping := emptyConfig.GetEndpointToListenerIDMapping()
	assert.Empty(t, emptyMapping, "Empty config should produce empty mapping")
}
