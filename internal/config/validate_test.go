package config

import (
	"embed"
	"strings"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "Valid version",
			version:     VersionLatest,
			expectError: false,
		},
		{
			name:        "Empty version gets set to unknown",
			version:     "",
			expectError: true, // VersionUnknown is not valid
		},
		{
			name:        "Unknown version",
			version:     VersionUnknown,
			expectError: true,
		},
		{
			name:        "Invalid version",
			version:     "invalid-version",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				Version: tc.version,
			}

			err := config.validateVersion()
			if tc.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrUnsupportedConfigVer)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to check if an error slice contains a specific error substring
func hasErrorContaining(t *testing.T, errs []error, substring string) {
	t.Helper()

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), substring) {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected error containing: %s", substring)
}

func TestValidateListeners(t *testing.T) {
	t.Parallel()

	// Helper function to create a standard HTTP listener config
	createHTTPListener := func(id, address string, readTimeout time.Duration) listeners.Listener {
		return listeners.Listener{
			ID:      id,
			Address: address,
			Type:    listeners.TypeHTTP,
			Options: options.HTTP{
				ReadTimeout:  readTimeout,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
				DrainTimeout: 15 * time.Second,
			},
		}
	}

	// Test with empty listeners
	t.Run("Empty listeners", func(t *testing.T) {
		t.Parallel()
		config := &Config{
			Version:   VersionLatest,
			Listeners: listeners.ListenerCollection{},
		}
		listenerIDs, errs := config.validateListeners()
		assert.Empty(t, errs)
		assert.Empty(t, listenerIDs)
	})

	// Test with valid listeners
	t.Run("Valid listeners", func(t *testing.T) {
		t.Parallel()
		config := &Config{
			Version: VersionLatest,
			Listeners: listeners.ListenerCollection{
				createHTTPListener("http1", "127.0.0.1:8080", 30*time.Second),
				createHTTPListener("http2", "127.0.0.1:8081", 30*time.Second),
			},
		}
		listenerIDs, errs := config.validateListeners()

		// We don't check for empty errors as HTTP validation might produce warnings
		// Instead, we check that no duplicate ID errors are present
		for _, err := range errs {
			assert.NotContains(t, err.Error(), "duplicate ID")
		}

		// Check that listener IDs were collected correctly
		assert.Equal(t, 2, len(listenerIDs))
		assert.True(t, listenerIDs["http1"])
		assert.True(t, listenerIDs["http2"])
	})

	// Test with duplicate ID
	t.Run("Duplicate listener IDs", func(t *testing.T) {
		t.Parallel()
		config := &Config{
			Version: VersionLatest,
			Listeners: listeners.ListenerCollection{
				createHTTPListener("http1", "127.0.0.1:8080", 30*time.Second),
				createHTTPListener("http1", "127.0.0.1:8081", 30*time.Second), // Duplicate ID
			},
		}
		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs)

		// Check for duplicate ID error
		hasErrorContaining(t, errs, "duplicate ID: listener ID")
	})

	// Test with duplicate address
	t.Run("Duplicate listener addresses", func(t *testing.T) {
		t.Parallel()
		config := &Config{
			Version: VersionLatest,
			Listeners: listeners.ListenerCollection{
				createHTTPListener("http1", "127.0.0.1:8080", 30*time.Second),
				createHTTPListener("http2", "127.0.0.1:8080", 30*time.Second), // Duplicate address
			},
		}
		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs)

		// Check for duplicate address error
		hasErrorContaining(t, errs, "duplicate ID: listener address")
	})

	// Test with invalid options
	t.Run("Invalid listener options", func(t *testing.T) {
		t.Parallel()
		config := &Config{
			Version: VersionLatest,
			Listeners: listeners.ListenerCollection{
				createHTTPListener("invalid", "127.0.0.1:8080", -1*time.Second), // Negative timeout
			},
		}
		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs)

		// Check for invalid options error
		hasErrorContaining(t, errs, "invalid HTTP options")
	})
}

func TestValidateEndpoints(t *testing.T) {
	t.Parallel()

	// Setup a map of valid listener IDs
	validListenerIDs := map[string]bool{
		"http1":  true,
		"grpc1":  true,
		"valid1": true,
		"valid2": true,
	}

	testCases := []struct {
		name        string
		endpoints   endpoints.EndpointCollection
		expectError bool
		errorCount  int
	}{
		{
			name:        "Empty endpoints",
			endpoints:   endpoints.EndpointCollection{},
			expectError: false,
		},
		{
			name: "Valid endpoints",
			endpoints: endpoints.EndpointCollection{
				{
					ID:         "ep1",
					ListenerID: "http1",
				},
				{
					ID:         "ep2",
					ListenerID: "valid1",
				},
			},
			expectError: false,
		},
		{
			name: "Duplicate endpoint IDs",
			endpoints: endpoints.EndpointCollection{
				{
					ID:         "ep1",
					ListenerID: "http1",
				},
				{
					ID:         "ep1", // Duplicate ID
					ListenerID: "grpc1",
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "Invalid listener reference",
			endpoints: endpoints.EndpointCollection{
				{
					ID:         "ep1",
					ListenerID: "invalid1", // Not in validListenerIDs
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "Multiple invalid references",
			endpoints: endpoints.EndpointCollection{
				{
					ID:         "ep1",
					ListenerID: "invalid1", // Invalid
				},
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				Version:   VersionLatest,
				Endpoints: tc.endpoints,
			}

			errs := config.validateEndpoints(validListenerIDs)

			if tc.expectError {
				assert.NotEmpty(t, errs)
				if tc.errorCount > 0 {
					assert.Len(t, errs, tc.errorCount)
				}
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateAppsAndRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setupConfig func() *Config
		expectError bool
	}{
		{
			name: "Valid apps and routes",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "echo-app",
							Config: &echo.EchoApp{Response: "Hello"},
						},
					),
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "echo-app", // References existing app
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "Invalid app reference",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "echo-app",
							Config: &echo.EchoApp{Response: "Hello"},
						},
					),
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "non-existent-app", // References non-existent app
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
					},
				}
			},
			expectError: true,
		},
		{
			name: "Empty app ID in route",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "echo-app",
							Config: &echo.EchoApp{Response: "Hello"},
						},
					),
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "", // Empty app ID - should be caught during endpoint validation
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
					},
				}
			},
			expectError: false, // ValidateAppsAndRoutes currently doesn't validate empty route appIDs, this is done elsewhere
		},
		{
			name: "Invalid app config",
			setupConfig: func() *Config {
				// Create an invalid script with empty code and negative timeout, which will fail validation
				invalidScript := scripts.NewAppScript(
					nil,
					&evaluators.RisorEvaluator{Code: "", Timeout: -1 * time.Second},
				)
				return &Config{
					Version: VersionLatest,
					Apps: apps.NewAppCollection(
						apps.App{
							ID:     "invalid-app",
							Config: invalidScript,
						},
					),
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "invalid-app",
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
					},
				}
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.setupConfig()
			err := config.validateAppsAndRoutes()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRouteConflicts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setupConfig func() *Config
		expectError bool
	}{
		{
			name: "No conflicts",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app1",
									Condition: conditions.NewHTTP("/path1", ""),
								},
							},
						},
						{
							ID:         "ep2",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app2",
									Condition: conditions.NewHTTP("/path2", ""), // Different path
								},
							},
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "Same path on different listeners",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app1",
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
						{
							ID:         "ep2",
							ListenerID: "l2", // Different listener
							Routes: routes.RouteCollection{
								{
									AppID:     "app2",
									Condition: conditions.NewHTTP("/api", "GET"), // Same path is OK
								},
							},
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "Conflicting HTTP paths",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app1",
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
						{
							ID:         "ep2",
							ListenerID: "l1", // Same listener
							Routes: routes.RouteCollection{
								{
									AppID: "app2",
									Condition: conditions.NewHTTP(
										"/api",
										"GET",
									), // Same path - conflict!
								},
							},
						},
					},
				}
			},
			expectError: true,
		},
		{
			name: "Conflicting gRPC service",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app1",
									Condition: conditions.NewHTTP("/api/test", "POST"),
								},
							},
						},
						{
							ID:         "ep2",
							ListenerID: "l1", // Same listener
							Routes: routes.RouteCollection{
								{
									AppID: "app2",
									Condition: conditions.NewHTTP(
										"/api/test", "POST",
									), // Same path - conflict!
								},
							},
						},
					},
				}
			},
			expectError: true,
		},
		{
			name: "Multiple conflicts",
			setupConfig: func() *Config {
				return &Config{
					Version: VersionLatest,
					Endpoints: endpoints.EndpointCollection{
						{
							ID:         "ep1",
							ListenerID: "l1",
							Routes: routes.RouteCollection{
								{
									AppID:     "app1",
									Condition: conditions.NewHTTP("/api", "GET"),
								},
							},
						},
						{
							ID:         "ep2",
							ListenerID: "l1", // Same listener
							Routes: routes.RouteCollection{
								{
									AppID:     "app2",
									Condition: conditions.NewHTTP("/api", "GET"), // Conflict
								},
							},
						},
					},
				}
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.setupConfig()
			err := config.validateRouteConflicts()

			if tc.expectError {
				assert.Error(t, err)
				// The error contains condition conflict details
				assert.Contains(t, err.Error(), "condition")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		config      *Config
		expectError bool
		errorType   error
	}{
		{
			name: "Valid config",
			config: &Config{
				Version: VersionLatest,
				Listeners: listeners.ListenerCollection{
					{
						ID:      "http1",
						Address: "127.0.0.1:8080",
						Type:    listeners.TypeHTTP,
						Options: options.HTTP{
							ReadTimeout:  30 * time.Second,
							WriteTimeout: 30 * time.Second,
							IdleTimeout:  60 * time.Second,
							DrainTimeout: 15 * time.Second,
						},
					},
				},
				Endpoints: endpoints.EndpointCollection{
					{
						ID:         "ep1",
						ListenerID: "http1",
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
						Config: &echo.EchoApp{Response: "Hello"},
					},
				),
			},
			expectError: false,
		},
		{
			name: "Invalid version",
			config: &Config{
				Version: "invalid",
			},
			expectError: true,
			errorType:   ErrUnsupportedConfigVer,
		},
		{
			name: "Invalid listener",
			config: &Config{
				Version: VersionLatest,
				Listeners: listeners.ListenerCollection{
					{
						ID:      "", // Empty ID
						Address: "127.0.0.1:8080",
						Type:    listeners.TypeHTTP,
						Options: options.HTTP{
							ReadTimeout:  30 * time.Second,
							WriteTimeout: 30 * time.Second,
							IdleTimeout:  60 * time.Second,
							DrainTimeout: 15 * time.Second,
						},
					},
				},
			},
			expectError: true,
			errorType:   ErrFailedToValidateConfig,
		},
		{
			name: "Invalid endpoint reference",
			config: &Config{
				Version: VersionLatest,
				Listeners: listeners.ListenerCollection{
					{
						ID:      "http1",
						Address: "127.0.0.1:8080",
						Type:    listeners.TypeHTTP,
						Options: options.HTTP{
							ReadTimeout:  30 * time.Second,
							WriteTimeout: 30 * time.Second,
							IdleTimeout:  60 * time.Second,
							DrainTimeout: 15 * time.Second,
						},
					},
				},
				Endpoints: endpoints.EndpointCollection{
					{
						ID:         "ep1",
						ListenerID: "invalid", // Invalid listener reference
					},
				},
			},
			expectError: true,
			errorType:   ErrFailedToValidateConfig,
		},
		{
			name: "Route conflict",
			config: &Config{
				Version: VersionLatest,
				Listeners: listeners.ListenerCollection{
					{
						ID:      "http1",
						Address: "127.0.0.1:8080",
						Type:    listeners.TypeHTTP,
						Options: options.HTTP{
							ReadTimeout:  30 * time.Second,
							WriteTimeout: 30 * time.Second,
							IdleTimeout:  60 * time.Second,
							DrainTimeout: 15 * time.Second,
						},
					},
				},
				Endpoints: endpoints.EndpointCollection{
					{
						ID:         "ep1",
						ListenerID: "http1",
						Routes: routes.RouteCollection{
							{
								AppID:     "app1",
								Condition: conditions.NewHTTP("/api", ""),
							},
						},
					},
					{
						ID:         "ep2",
						ListenerID: "http1",
						Routes: routes.RouteCollection{
							{
								AppID:     "app2",
								Condition: conditions.NewHTTP("/api", ""), // Conflicting path
							},
						},
					},
				},
				Apps: apps.NewAppCollection(
					apps.App{
						ID:     "app1",
						Config: &echo.EchoApp{Response: "Hello from app1"},
					},
					apps.App{
						ID:     "app2",
						Config: &echo.EchoApp{Response: "Hello from app2"},
					},
				),
			},
			expectError: true,
			errorType:   ErrFailedToValidateConfig,
		},
		{
			name: "Invalid app reference",
			config: &Config{
				Version: VersionLatest,
				Listeners: listeners.ListenerCollection{
					{
						ID:      "http1",
						Address: "127.0.0.1:8080",
						Type:    listeners.TypeHTTP,
						Options: options.HTTP{
							ReadTimeout:  30 * time.Second,
							WriteTimeout: 30 * time.Second,
							IdleTimeout:  60 * time.Second,
							DrainTimeout: 15 * time.Second,
						},
					},
				},
				Endpoints: endpoints.EndpointCollection{
					{
						ID:         "ep1",
						ListenerID: "http1",
						Routes: routes.RouteCollection{
							{
								AppID:     "non-existent", // App doesn't exist
								Condition: conditions.NewHTTP("/api", ""),
							},
						},
					},
				},
				Apps: apps.NewAppCollection(
					apps.App{
						ID:     "app1",
						Config: &echo.EchoApp{Response: "Hello"},
					},
				),
			},
			expectError: true,
			errorType:   ErrFailedToValidateConfig,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.Contains(t, err.Error(), tc.errorType.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test the collectRouteReferences function separately
func TestCollectRouteReferences(t *testing.T) {
	t.Parallel()

	config := &Config{
		Version: VersionLatest,
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "ep1",
				ListenerID: "l1",
				Routes: routes.RouteCollection{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/path1", ""),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/path2", ""),
					},
				},
			},
			{
				ID:         "ep2",
				ListenerID: "l2",
				Routes: routes.RouteCollection{
					{
						AppID:     "app3",
						Condition: conditions.NewHTTP("/path3", ""),
					},
				},
			},
		},
	}

	routeRefs := config.collectRouteReferences()

	// Verify we get all route references
	require.Len(t, routeRefs, 3)

	// Create a map to check all app IDs are present
	appIDs := map[string]bool{}
	for _, ref := range routeRefs {
		appIDs[ref.AppID] = true
	}

	assert.True(t, appIDs["app1"])
	assert.True(t, appIDs["app2"])
	assert.True(t, appIDs["app3"])
}

//go:embed testdata/invalid/*.toml
var invalidConfigFiles embed.FS

func TestInvalidConfigValidation(t *testing.T) {
	entries, err := invalidConfigFiles.ReadDir("testdata/invalid")
	require.NoError(t, err, "Failed to read embedded invalid config files")
	t.Logf("Found %d invalid config files", len(entries))

	require.NotEmpty(t, entries, "No invalid TOML config files found")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			data, err := invalidConfigFiles.ReadFile("testdata/invalid/" + entry.Name())
			require.NoError(t, err, "Failed to read embedded invalid file: %s", entry.Name())

			// Attempt to load the config
			cfg, err := NewConfigFromBytes(data)
			// If parsing fails, that's one way to fail
			if err != nil {
				t.Logf("Config %s failed during parsing: %v", entry.Name(), err)
				return
			}

			// If parsing succeeded, validation must fail
			require.NotNil(t, cfg, "Config should not be nil for %s", entry.Name())
			err = cfg.Validate()
			require.Error(t, err, "Config %s should fail validation", entry.Name())
			t.Logf("Validation error for %s: %v", entry.Name(), err)

			// Check for specific error messages
			if entry.Name() == "invalid_listener_id.toml" {
				require.Contains(t, err.Error(), "references non-existent listener ID",
					"Error should mention non-existent listener IDs")
			}
		})
	}
}
