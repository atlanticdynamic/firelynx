package toml

import (
	"testing"

	pbMiddleware "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	middlewareLogger "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared TOML configuration template with manual field configuration
const manualLoggerTOMLConfig = `
version = "v1"

[[listeners]]
id = "http"
address = "127.0.0.1:8080"
type = "http"

[[endpoints]]
id = "manual-endpoint"
listener_id = "http"

[[endpoints.middlewares]]
id = "manual-logger"
type = "console_logger"

[endpoints.middlewares.console_logger]
output = "/tmp/manual.log"

[endpoints.middlewares.console_logger.options]
format = "json"
level = "warn"

[endpoints.middlewares.console_logger.fields]
method = true
path = true
status_code = true
client_ip = true
duration = true
query_params = true
protocol = true
host = true

[endpoints.middlewares.console_logger.fields.request]
enabled = true
headers = true
include_headers = ["User-Agent", "Content-Type", "Accept"]
exclude_headers = ["Authorization", "Cookie"]

[endpoints.middlewares.console_logger.fields.response]
enabled = true
headers = true
body_size = true
include_headers = ["Content-Type", "Cache-Control"]
exclude_headers = ["Set-Cookie"]

[[endpoints.routes]]
app_id = "test-echo"

[endpoints.routes.http]
path_prefix = "/manual-test"

[[apps]]
id = "test-echo"

[apps.echo]
response = "Manual Test Response"
`

// TestProcessConsoleLoggerFields tests the low-level field processing function
func TestProcessConsoleLoggerFields(t *testing.T) {
	config := &pbMiddleware.ConsoleLoggerConfig{}

	fieldsMap := map[string]any{
		"method":       true,
		"path":         true,
		"status_code":  true,
		"client_ip":    true,
		"duration":     true,
		"query_params": true,
		"protocol":     true,
		"host":         true,
		"request": map[string]any{
			"enabled":         true,
			"headers":         true,
			"include_headers": []any{"User-Agent", "Content-Type", "Accept"},
			"exclude_headers": []any{"Authorization", "Cookie"},
		},
		"response": map[string]any{
			"enabled":         true,
			"headers":         true,
			"body_size":       true,
			"include_headers": []any{"Content-Type", "Cache-Control"},
			"exclude_headers": []any{"Set-Cookie"},
		},
	}

	errs := processConsoleLoggerFields(config, fieldsMap)
	require.Empty(t, errs, "Field processing should succeed")

	// Verify fields were set correctly
	require.NotNil(t, config.Fields, "Fields should be initialized")
	assert.True(t, *config.Fields.Method, "Method should be true")
	assert.True(t, *config.Fields.Path, "Path should be true")
	assert.True(t, *config.Fields.StatusCode, "StatusCode should be true")
	assert.True(t, *config.Fields.ClientIp, "ClientIp should be true")
	assert.True(t, *config.Fields.Duration, "Duration should be true")
	assert.True(t, *config.Fields.QueryParams, "QueryParams should be true")
	assert.True(t, *config.Fields.Protocol, "Protocol should be true")
	assert.True(t, *config.Fields.Host, "Host should be true")

	// Verify request config
	require.NotNil(t, config.Fields.Request, "Request config should be initialized")
	assert.True(t, *config.Fields.Request.Enabled, "Request enabled should be true")
	assert.True(t, *config.Fields.Request.Headers, "Request headers should be true")
	assert.Equal(
		t,
		[]string{"User-Agent", "Content-Type", "Accept"},
		config.Fields.Request.IncludeHeaders,
	)
	assert.Equal(t, []string{"Authorization", "Cookie"}, config.Fields.Request.ExcludeHeaders)

	// Verify response config
	require.NotNil(t, config.Fields.Response, "Response config should be initialized")
	assert.True(t, *config.Fields.Response.Enabled, "Response enabled should be true")
	assert.True(t, *config.Fields.Response.Headers, "Response headers should be true")
	assert.True(t, *config.Fields.Response.BodySize, "Response body_size should be true")
	assert.Equal(
		t,
		[]string{"Content-Type", "Cache-Control"},
		config.Fields.Response.IncludeHeaders,
	)
	assert.Equal(t, []string{"Set-Cookie"}, config.Fields.Response.ExcludeHeaders)
}

func TestProcessConsoleLoggerFieldsEmpty(t *testing.T) {
	config := &pbMiddleware.ConsoleLoggerConfig{}
	fieldsMap := map[string]any{}

	errs := processConsoleLoggerFields(config, fieldsMap)
	assert.Empty(t, errs, "Empty fields should not cause errors")
	assert.NotNil(t, config.Fields, "Fields should be initialized even when empty")
}

// TestManualLoggerTOMLIntegration tests TOML parsing to protobuf conversion
func TestManualLoggerTOMLIntegration(t *testing.T) {
	loader := NewTomlLoader([]byte(manualLoggerTOMLConfig))
	config, err := loader.LoadProto()
	require.NoError(t, err, "TOML loading should succeed")

	require.NotEmpty(t, config.Endpoints, "Should have endpoints")
	endpoint := config.Endpoints[0]

	require.NotEmpty(t, endpoint.Middlewares, "Should have middlewares")
	middleware := endpoint.Middlewares[0]

	require.Equal(t, "manual-logger", middleware.GetId(), "Middleware ID should match")

	consoleLogger := middleware.GetConsoleLogger()
	require.NotNil(t, consoleLogger, "Console logger config should exist")

	// Verify output and options
	assert.Equal(t, "/tmp/manual.log", consoleLogger.GetOutput(), "Output should be set")
	require.NotNil(t, consoleLogger.Options, "Options should be set")
	assert.Equal(t, "FORMAT_JSON", consoleLogger.Options.Format.String(), "Format should be JSON")
	assert.Equal(t, "LEVEL_WARN", consoleLogger.Options.Level.String(), "Level should be WARN")

	// Verify fields configuration
	require.NotNil(t, consoleLogger.Fields, "Fields should be configured")
	assert.True(t, consoleLogger.Fields.GetMethod(), "Method should be enabled")
	assert.True(t, consoleLogger.Fields.GetPath(), "Path should be enabled")
	assert.True(t, consoleLogger.Fields.GetStatusCode(), "Status code should be enabled")
	assert.True(t, consoleLogger.Fields.GetClientIp(), "Client IP should be enabled")
	assert.True(t, consoleLogger.Fields.GetDuration(), "Duration should be enabled")
	assert.True(t, consoleLogger.Fields.GetQueryParams(), "Query params should be enabled")
	assert.True(t, consoleLogger.Fields.GetProtocol(), "Protocol should be enabled")
	assert.True(t, consoleLogger.Fields.GetHost(), "Host should be enabled")

	// Verify request configuration
	request := consoleLogger.Fields.GetRequest()
	require.NotNil(t, request, "Request config should exist")
	assert.True(t, request.GetEnabled(), "Request should be enabled")
	assert.True(t, request.GetHeaders(), "Request headers should be enabled")
	assert.Equal(t, []string{"User-Agent", "Content-Type", "Accept"}, request.GetIncludeHeaders())
	assert.Equal(t, []string{"Authorization", "Cookie"}, request.GetExcludeHeaders())

	// Verify response configuration
	response := consoleLogger.Fields.GetResponse()
	require.NotNil(t, response, "Response config should exist")
	assert.True(t, response.GetEnabled(), "Response should be enabled")
	assert.True(t, response.GetHeaders(), "Response headers should be enabled")
	assert.True(t, response.GetBodySize(), "Response body size should be enabled")
	assert.Equal(t, []string{"Content-Type", "Cache-Control"}, response.GetIncludeHeaders())
	assert.Equal(t, []string{"Set-Cookie"}, response.GetExcludeHeaders())
}

// TestManualLoggerEndToEnd tests the complete TOML → protobuf → domain → middleware pipeline
func TestManualLoggerEndToEnd(t *testing.T) {
	// Step 1: Load TOML to protobuf
	loader := NewTomlLoader([]byte(manualLoggerTOMLConfig))
	pbConfig, err := loader.LoadProto()
	require.NoError(t, err, "TOML loading should succeed")

	require.NotEmpty(t, pbConfig.Endpoints, "Should have endpoints")
	endpoint := pbConfig.Endpoints[0]

	require.NotEmpty(t, endpoint.Middlewares, "Should have middlewares")
	middleware := endpoint.Middlewares[0]

	consoleLoggerConfig := middleware.GetConsoleLogger()
	require.NotNil(t, consoleLoggerConfig, "Console logger config should exist")

	// Step 2: Convert protobuf to domain model
	domainConfig, err := configLogger.FromProto(consoleLoggerConfig)
	require.NoError(t, err, "Protobuf to domain conversion should succeed")

	// Verify domain config has manual fields
	assert.True(t, domainConfig.Fields.Method, "Domain config should have method enabled")
	assert.True(t, domainConfig.Fields.Path, "Domain config should have path enabled")
	assert.True(t, domainConfig.Fields.StatusCode, "Domain config should have status_code enabled")
	assert.True(t, domainConfig.Fields.ClientIP, "Domain config should have client_ip enabled")
	assert.True(t, domainConfig.Fields.Duration, "Domain config should have duration enabled")
	assert.True(
		t,
		domainConfig.Fields.QueryParams,
		"Domain config should have query_params enabled",
	)
	assert.True(t, domainConfig.Fields.Protocol, "Domain config should have protocol enabled")
	assert.True(t, domainConfig.Fields.Host, "Domain config should have host enabled")

	// Check request direction config
	assert.True(t, domainConfig.Fields.Request.Enabled, "Request should be enabled")
	assert.True(t, domainConfig.Fields.Request.Headers, "Request headers should be enabled")
	assert.Equal(
		t,
		[]string{"User-Agent", "Content-Type", "Accept"},
		domainConfig.Fields.Request.IncludeHeaders,
	)
	assert.Equal(t, []string{"Authorization", "Cookie"}, domainConfig.Fields.Request.ExcludeHeaders)

	// Check response direction config
	assert.True(t, domainConfig.Fields.Response.Enabled, "Response should be enabled")
	assert.True(t, domainConfig.Fields.Response.Headers, "Response headers should be enabled")
	assert.True(t, domainConfig.Fields.Response.BodySize, "Response body_size should be enabled")
	assert.Equal(
		t,
		[]string{"Content-Type", "Cache-Control"},
		domainConfig.Fields.Response.IncludeHeaders,
	)
	assert.Equal(t, []string{"Set-Cookie"}, domainConfig.Fields.Response.ExcludeHeaders)

	// Step 3: Create middleware from domain config
	middlewareInstance, err := middlewareLogger.NewConsoleLogger("manual-logger", domainConfig)
	require.NoError(t, err, "Middleware creation should succeed")
	require.NotNil(t, middlewareInstance, "Middleware should be created")

	// Verify the domain config is still intact after middleware creation
	assert.True(
		t,
		domainConfig.Fields.Method,
		"Domain config should still have method enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.Path,
		"Domain config should still have path enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.StatusCode,
		"Domain config should still have status_code enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.ClientIP,
		"Domain config should still have client_ip enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.Duration,
		"Domain config should still have duration enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.QueryParams,
		"Domain config should still have query_params enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.Protocol,
		"Domain config should still have protocol enabled after middleware creation",
	)
	assert.True(
		t,
		domainConfig.Fields.Host,
		"Domain config should still have host enabled after middleware creation",
	)
}
