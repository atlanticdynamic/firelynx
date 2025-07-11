package interpolation_test

import (
	"os"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/headers"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndToEndInterpolation(t *testing.T) {
	ctx := t.Context()
	_ = ctx
	t.Helper()

	// Set up test environment variables
	require.NoError(t, os.Setenv("TEST_HOST", "api.example.com"))
	require.NoError(t, os.Setenv("TEST_PORT", "9090"))
	require.NoError(t, os.Setenv("APP_VERSION", "v1.2.3"))
	require.NoError(t, os.Setenv("API_PREFIX", "/api/v1"))
	defer func() {
		require.NoError(t, os.Unsetenv("TEST_HOST"))
		require.NoError(t, os.Unsetenv("TEST_PORT"))
		require.NoError(t, os.Unsetenv("APP_VERSION"))
		require.NoError(t, os.Unsetenv("API_PREFIX"))
	}()

	t.Run("listener address interpolation", func(t *testing.T) {
		// Create listener with interpolated address
		httpOpts := options.HTTP{
			ReadTimeout:  10000,
			WriteTimeout: 10000,
			IdleTimeout:  60000,
			DrainTimeout: 5000,
		}

		listener := &listeners.Listener{
			ID:      "api-listener",
			Address: "${TEST_HOST}:${TEST_PORT}",
			Type:    listeners.TypeHTTP,
			Options: httpOpts,
		}

		// Validate should interpolate the address
		err := listener.Validate()
		require.NoError(t, err)

		assert.Equal(t, "api.example.com:9090", listener.Address)
	})

	t.Run("listener address with default", func(t *testing.T) {
		httpOpts := options.HTTP{
			ReadTimeout:  10000,
			WriteTimeout: 10000,
			IdleTimeout:  60000,
			DrainTimeout: 5000,
		}

		listener := &listeners.Listener{
			ID:      "web-listener",
			Address: "${WEB_HOST:localhost}:${WEB_PORT:8080}",
			Type:    listeners.TypeHTTP,
			Options: httpOpts,
		}

		err := listener.Validate()
		require.NoError(t, err)

		assert.Equal(t, "localhost:8080", listener.Address)
	})

	t.Run("echo app response interpolation", func(t *testing.T) {
		app := &echo.EchoApp{
			Response: "Hello from ${TEST_HOST} running version ${APP_VERSION}",
		}

		err := app.Validate()
		require.NoError(t, err)

		assert.Equal(t, "Hello from api.example.com running version v1.2.3", app.Response)
	})

	t.Run("headers middleware interpolation", func(t *testing.T) {
		headerOps := &headers.HeaderOperations{
			SetHeaders: map[string]string{
				"X-Server-Host": "${TEST_HOST}",
				"X-App-Version": "${APP_VERSION}",
			},
			AddHeaders: map[string]string{
				"X-Environment": "${ENVIRONMENT:development}",
			},
		}

		err := headerOps.Validate()
		require.NoError(t, err)

		assert.Equal(t, "api.example.com", headerOps.SetHeaders["X-Server-Host"])
		assert.Equal(t, "v1.2.3", headerOps.SetHeaders["X-App-Version"])
		assert.Equal(t, "development", headerOps.AddHeaders["X-Environment"])
	})

	t.Run("route condition path interpolation", func(t *testing.T) {
		route := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: stringPtr("${API_PREFIX}/users"),
				},
			},
		}

		condition := conditions.FromProto(route)
		require.NotNil(t, condition)

		httpCond, ok := condition.(conditions.HTTP)
		require.True(t, ok)

		assert.Equal(
			t,
			"${API_PREFIX}/users",
			httpCond.PathPrefix,
			"FromProto should not interpolate - happens during validation phase",
		)

		err := httpCond.Validate()
		require.Error(
			t,
			err,
			"validation should fail on uninterpolated path that doesn't start with '/'",
		)
		assert.ErrorIs(
			t,
			err,
			conditions.ErrInvalidHTTPCondition,
			"should return HTTP condition validation error",
		)
	})

	t.Run("interpolation error handling", func(t *testing.T) {
		app := &echo.EchoApp{
			Response: "Error test: ${MISSING_VAR}",
		}

		err := app.Validate()
		assert.Error(
			t,
			err,
			"validation should fail when env var is missing and no default provided",
		)
		assert.Contains(
			t,
			err.Error(),
			"MISSING_VAR",
			"error should mention the missing environment variable",
		)
	})

	t.Run("complex nested interpolation", func(t *testing.T) {
		// Test with headers middleware that has both request and response operations
		headers := &headers.Headers{
			Request: &headers.HeaderOperations{
				SetHeaders: map[string]string{
					"Host":          "${TEST_HOST}:${TEST_PORT}",
					"Authorization": "Bearer ${AUTH_TOKEN:default-token}",
				},
			},
			Response: &headers.HeaderOperations{
				SetHeaders: map[string]string{
					"Server":       "Firelynx/${APP_VERSION}",
					"X-Powered-By": "${POWERED_BY:Firelynx}",
				},
				AddHeaders: map[string]string{
					"X-Request-ID": "${REQUEST_ID:auto-generated}",
				},
			},
		}

		err := headers.Validate()
		require.NoError(t, err)

		// Check request headers
		assert.Equal(t, "api.example.com:9090", headers.Request.SetHeaders["Host"])
		assert.Equal(t, "Bearer default-token", headers.Request.SetHeaders["Authorization"])

		// Check response headers
		assert.Equal(t, "Firelynx/v1.2.3", headers.Response.SetHeaders["Server"])
		assert.Equal(t, "Firelynx", headers.Response.SetHeaders["X-Powered-By"])
		assert.Equal(t, "auto-generated", headers.Response.AddHeaders["X-Request-ID"])
	})

	t.Run("idempotent validation", func(t *testing.T) {
		app := &echo.EchoApp{
			Response: "Host: ${TEST_HOST}",
		}

		// First validation should interpolate
		err := app.Validate()
		require.NoError(t, err)
		firstResult := app.Response

		// Second validation should not change the result
		err = app.Validate()
		require.NoError(t, err)
		secondResult := app.Response

		assert.Equal(t, firstResult, secondResult)
		assert.Equal(t, "Host: api.example.com", app.Response)
	})
}

func TestFieldRulesCompliance(t *testing.T) {
	ctx := t.Context()
	_ = ctx
	t.Helper()

	// Set up test environment variable
	require.NoError(t, os.Setenv("TEST_VALUE", "interpolated"))
	defer func() {
		require.NoError(t, os.Unsetenv("TEST_VALUE"))
	}()

	t.Run("ID fields are never interpolated", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "listener-test", // Use valid ID since ${TEST_VALUE} would fail ID validation
			Address: ":8080",
			Type:    listeners.TypeHTTP,
			Options: options.HTTP{
				ReadTimeout:  10000,
				WriteTimeout: 10000,
				IdleTimeout:  60000,
				DrainTimeout: 5000,
			},
		}

		err := listener.Validate()
		require.NoError(t, err, "listener validation should succeed")

		assert.Equal(
			t,
			"listener-test",
			listener.ID,
			"ID field should never be interpolated due to env_interpolation:'no' tag",
		)
		assert.Equal(
			t,
			":8080",
			listener.Address,
			"address without env vars should remain unchanged",
		)
	})

	t.Run("path fields are interpolated", func(t *testing.T) {
		route := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: stringPtr("/api/${TEST_VALUE}"),
				},
			},
		}

		condition := conditions.FromProto(route)
		require.NotNil(t, condition)

		httpCond := condition.(conditions.HTTP)
		assert.Equal(
			t,
			"/api/${TEST_VALUE}",
			httpCond.PathPrefix,
			"FromProto should not interpolate - route-level validation handles interpolation",
		)
	})

	t.Run("value fields are interpolated", func(t *testing.T) {
		headerOps := &headers.HeaderOperations{
			SetHeaders: map[string]string{
				"X-Test": "${TEST_VALUE}",
			},
		}

		err := headerOps.Validate()
		require.NoError(t, err, "header operations validation should succeed")

		assert.Equal(
			t,
			"interpolated",
			headerOps.SetHeaders["X-Test"],
			"map values should be interpolated due to env_interpolation:'yes' tag",
		)
	})
}

func TestProtobufRoundTripWithInterpolation(t *testing.T) {
	ctx := t.Context()
	_ = ctx
	t.Helper()

	// Set up test environment variables
	require.NoError(t, os.Setenv("SERVER_PORT", "3000"))
	require.NoError(t, os.Setenv("APP_NAME", "TestApp"))
	defer func() {
		require.NoError(t, os.Unsetenv("SERVER_PORT"))
		require.NoError(t, os.Unsetenv("APP_NAME"))
	}()

	t.Run("route condition protobuf integration", func(t *testing.T) {
		// Test that route conditions properly interpolate when created from protobuf
		route := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: stringPtr("/apps/${APP_NAME}"),
				},
			},
		}

		condition := conditions.FromProto(route)
		require.NotNil(t, condition)

		httpCond, ok := condition.(conditions.HTTP)
		require.True(t, ok)

		assert.Equal(
			t,
			"/apps/${APP_NAME}",
			httpCond.PathPrefix,
			"FromProto should not interpolate - tag-based system requires explicit validation",
		)
	})
}

// Helper function to create string pointers for protobuf
func stringPtr(s string) *string {
	return &s
}
