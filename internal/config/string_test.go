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

func TestConfigString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		setupConfig    func() *Config
		expectedSubstr []string
	}{
		{
			name: "Empty config",
			setupConfig: func() *Config {
				return &Config{
					Version: version.Version,
				}
			},
			expectedSubstr: []string{
				"Firelynx Config (" + version.Version + ")",
			},
		},
		{
			name: "Config with listeners",
			setupConfig: func() *Config {
				return &Config{
					Version: version.Version,
					Listeners: listeners.ListenerCollection{
						{
							ID:      "http-listener",
							Address: "127.0.0.1:8080",
							Options: options.HTTP{
								ReadTimeout:  30 * time.Second,
								WriteTimeout: 30 * time.Second,
							},
						},
					},
				}
			},
			expectedSubstr: []string{
				"Listeners (1)",
				"http-listener",
				"127.0.0.1:8080",
				"ReadTimeout: 30s",
			},
		},
		{
			name: "Config with endpoints",
			setupConfig: func() *Config {
				return &Config{
					Version: version.Version,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "test-endpoint",
							ListenerID: "http-listener",
							Routes: routes.RouteCollection{
								{
									AppID:     "echo-app",
									Condition: conditions.NewHTTP("/api", ""),
								},
							},
						},
					},
				}
			},
			expectedSubstr: []string{
				"Endpoints (1)",
				"test-endpoint",
				"Listeners: http-listener",
				"Routes",
				"Condition: http_path = /api",
			},
		},
		{
			name: "Config with apps",
			setupConfig: func() *Config {
				return &Config{
					Version: version.Version,
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "echo-app",
							Config: &echo.EchoApp{Response: "Hello World"},
						},
						apps.App{
							ID: "script-app",
							Config: scripts.NewAppScript(
								"script-app",
								nil,
								&evaluators.RisorEvaluator{
									Code: "return { body: 'Hello' }",
								},
							),
						},
					),
				}
			},
			expectedSubstr: []string{
				"Apps (2)",
				"echo-app",
				"script-app",
				"Type: Script",
				"Evaluator: Risor",
			},
		},
		{
			name: "Full config",
			setupConfig: func() *Config {
				return &Config{
					Version: version.Version,
					Listeners: listeners.ListenerCollection{
						{
							ID:      "http-listener",
							Address: "127.0.0.1:8080",
							Options: options.HTTP{
								ReadTimeout:  30 * time.Second,
								WriteTimeout: 30 * time.Second,
							},
						},
					},
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "test-endpoint",
							ListenerID: "http-listener",
							Routes: routes.RouteCollection{
								{
									AppID:     "echo-app",
									Condition: conditions.NewHTTP("/api", ""),
								},
							},
						},
					},
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "echo-app",
							Config: &echo.EchoApp{Response: "Hello World"},
						},
					),
				}
			},
			expectedSubstr: []string{
				"Firelynx Config (" + version.Version + ")",
				"Listeners (1)",
				"http-listener",
				"Endpoints (1)",
				"test-endpoint",
				"Apps (1)",
				"echo-app",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.setupConfig()
			result := config.String()

			// Verify the result is not empty
			require.NotEmpty(t, result)

			// Verify it contains all expected substrings
			for _, substr := range tc.expectedSubstr {
				assert.Contains(t,
					result, substr,
					"Expected string representation to contain '%s', but got:\n%s",
					substr, result)
			}
		})
	}
}

func TestConfigTree(t *testing.T) {
	t.Parallel()

	// Test that the ConfigTree function returns the same result as String
	config := &Config{
		Version: version.Version,
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http-listener",
				Address: "127.0.0.1:8080",
				Options: options.HTTP{
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "test-endpoint",
				ListenerID: "http-listener",
				Routes: routes.RouteCollection{
					{
						AppID:     "echo-app",
						Condition: conditions.NewHTTP("/api", ""),
					},
				},
			},
		},
		Apps: apps.NewAppCollection(
			apps.App{
				ID:     "echo-app",
				Config: &echo.EchoApp{Response: "Hello World"},
			},
		),
	}

	// Verify that ConfigTree and String return the same result
	stringResult := config.String()
	treeResult := ConfigTree(config)

	assert.Equal(t, stringResult, treeResult)
}
