package config

import (
	"errors"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("Version", func(t *testing.T) {
		// Group all version-related validation tests
		t.Run("InvalidVersionFromBytes", func(t *testing.T) {
			configBytes := []byte(`
version = "v2"

[logging]
format = "txt"
level = "debug"
`)
			config, err := NewConfigFromBytes(configBytes)
			require.Error(t, err, "Expected error for unsupported version")
			assert.Nil(t, config, "Config should be nil when version validation fails")
			assert.ErrorContains(t, err, "unsupported config version: v2")
		})

		t.Run("DomainModelValidation", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Version = "v999"

			err := config.Validate()
			require.Error(t, err, "Expected error for unsupported version")
			assert.ErrorIs(t, err, ErrUnsupportedConfigVer)
		})

		t.Run("EmptyVersionDefaultsToUnknown", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Version = ""

			err := config.Validate()
			require.Error(t, err, "Expected error for empty version")
			assert.ErrorIs(t, err, ErrUnsupportedConfigVer)
			assert.ErrorContains(t, err, VersionUnknown)
		})

		t.Run("ValidVersionSucceeds", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Version = VersionLatest

			err := config.Validate()
			if err != nil {
				assert.NotErrorIs(t, err, ErrUnsupportedConfigVer,
					"Valid version should not trigger version error")
			}
		})
	})

	t.Run("Listeners", func(t *testing.T) {
		// Group all listener-related validation tests
		t.Run("DuplicateID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Listeners = append(config.Listeners, listeners.Listener{
				ID:      "listener1", // Duplicate of existing ID
				Address: ":9090",     // Different address
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for duplicate listener ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrDuplicateID.Error())
		})

		t.Run("DuplicateAddress", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Listeners = append(config.Listeners, listeners.Listener{
				ID:      "listener2", // Different ID
				Address: ":8080",     // Duplicate of existing address
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for duplicate listener address")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrDuplicateID.Error())
		})

		t.Run("EmptyID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Listeners = append(config.Listeners, listeners.Listener{
				ID:      "", // Empty ID
				Address: ":9090",
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for empty listener ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrEmptyID.Error())
		})

		t.Run("EmptyAddress", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Listeners = append(config.Listeners, listeners.Listener{
				ID:      "listener2",
				Address: "", // Empty address
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for empty listener address")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrMissingRequiredField.Error())
		})
	})

	t.Run("Endpoints", func(t *testing.T) {
		// Group all endpoint-related validation tests
		t.Run("DuplicateID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Endpoints = append(config.Endpoints, endpoints.Endpoint{
				ID:          "endpoint1", // Duplicate of existing ID
				ListenerIDs: []string{"listener1"},
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for duplicate endpoint ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrDuplicateID.Error())
		})

		t.Run("EmptyID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Endpoints = append(config.Endpoints, endpoints.Endpoint{
				ID:          "", // Empty ID
				ListenerIDs: []string{"listener1"},
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for empty endpoint ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrEmptyID.Error())
		})

		t.Run("NonExistentListenerID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Endpoints = append(config.Endpoints, endpoints.Endpoint{
				ID: "endpoint2",
				ListenerIDs: []string{
					"non_existent_listener",
				}, // Reference to non-existent listener
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for non-existent listener ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrListenerNotFound.Error())
		})

		t.Run("EmptyAppIDInRoute", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Endpoints[0].Routes = append(config.Endpoints[0].Routes, routes.Route{
				AppID:     "", // Empty app ID
				Condition: conditions.NewHTTP("/empty"),
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for empty app ID in route")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrEmptyID.Error())
		})
	})

	t.Run("Apps", func(t *testing.T) {
		// Group all app-related validation tests
		t.Run("DuplicateID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Apps = append(config.Apps, apps.App{
				ID: "app1", // Duplicate of existing ID
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for duplicate app ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrDuplicateID.Error())
		})

		t.Run("EmptyID", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Apps = append(config.Apps, apps.App{
				ID: "", // Empty ID
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for empty app ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrEmptyID.Error())
		})

		t.Run("NonExistentAppIDInRoute", func(t *testing.T) {
			config := createValidDomainConfig(t)
			config.Endpoints[0].Routes = append(config.Endpoints[0].Routes, routes.Route{
				AppID:     "non_existent_app", // Reference to non-existent app
				Condition: conditions.NewHTTP("/non-existent"),
			})

			err := config.Validate()
			require.Error(t, err, "Expected error for non-existent app ID")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrAppNotFound.Error())
		})

		t.Run("CompositeScripts", func(t *testing.T) {
			t.Run("NonExistentScriptAppID", func(t *testing.T) {
				config := createValidDomainConfig(t)
				config.Apps = append(config.Apps, apps.App{
					ID: "composite_app",
					Config: apps.CompositeScriptApp{
						ScriptAppIDs: []string{
							"app1",
							"non_existent_app",
						}, // One valid, one non-existent
					},
				})

				err := config.Validate()
				require.Error(t, err, "Expected error for non-existent script app ID")
				assert.ErrorIs(t, err, ErrFailedToValidateConfig)
				assert.ErrorContains(t, err, ErrAppNotFound.Error())
			})

			t.Run("ValidCompositeScript", func(t *testing.T) {
				config := createValidDomainConfig(t)
				// Add a second app that can be referenced
				config.Apps = append(config.Apps, apps.App{
					ID: "app2",
					Config: apps.ScriptApp{
						Evaluator: apps.RisorEvaluator{
							Code: "function handle(req) { return req; }",
						},
					},
				})
				// Add the composite app that references both valid apps
				config.Apps = append(config.Apps, apps.App{
					ID: "composite_app",
					Config: apps.CompositeScriptApp{
						ScriptAppIDs: []string{"app1", "app2"}, // Both valid
					},
				})

				err := config.Validate()
				if err != nil {
					assert.NotContains(t, err.Error(), "composite script",
						"Error should not be about composite scripts")
				}
			})
		})
	})

	t.Run("Routes", func(t *testing.T) {
		// Group all route-related validation tests
		t.Run("ConflictingRoutes", func(t *testing.T) {
			// Create a configuration with two endpoints that have routes with the same HTTP path
			configBytes := []byte(`
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "http_listener"
address = ":8080"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/foo"
`)
			config, err := NewConfigFromBytes(configBytes)

			require.Error(t, err, "Expected error for conflicting routes")
			assert.ErrorIs(t, err, ErrFailedToValidateConfig)
			assert.ErrorContains(t, err, ErrRouteConflict.Error())
			assert.Nil(t, config, "Config should be nil when validation fails")
		})

		t.Run("DifferentPathsNoConflict", func(t *testing.T) {
			// Create a configuration with two endpoints with different HTTP paths
			configBytes := []byte(`
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "http_listener"
address = ":8080"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/bar"
`)
			config, err := NewConfigFromBytes(configBytes)
			require.NoError(t, err, "Different paths should not cause a conflict")
			assert.NotNil(t, config, "Config should be created successfully")
		})

		t.Run("SamePathDifferentListeners", func(t *testing.T) {
			// Create a configuration with same HTTP path on different listeners
			configBytes := []byte(`
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "http_listener1"
address = ":8080"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[listeners]]
id = "http_listener2"
address = ":9090"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener1"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener2"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/foo"
`)
			config, err := NewConfigFromBytes(configBytes)
			require.NoError(t, err, "Same paths on different listeners should not cause a conflict")
			assert.NotNil(t, config, "Config should be created successfully")
		})
	})
}

// Unit tests for specific validation functions follow

func TestConfig_validateVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
		errorIs     error
	}{
		{
			name:        "Latest version",
			version:     VersionLatest,
			expectError: false,
		},
		{
			name:        "Empty version defaults to unknown",
			version:     "",
			expectError: true,
			errorIs:     ErrUnsupportedConfigVer,
		},
		{
			name:        "Unsupported version",
			version:     "v999",
			expectError: true,
			errorIs:     ErrUnsupportedConfigVer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				Version: tc.version,
			}
			err := config.validateVersion()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateListeners(t *testing.T) {
	t.Run("ValidListeners", func(t *testing.T) {
		config := &Config{
			Listeners: []listeners.Listener{
				{
					ID:      "listener1",
					Address: ":8080",
					Options: options.NewHTTP(),
				},
				{
					ID:      "listener2",
					Address: ":9090",
					Options: options.NewGRPC(),
				},
			},
		}

		listenerIds, errs := config.validateListeners()
		assert.Empty(t, errs, "Expected no validation errors")
		assert.Equal(t, 2, len(listenerIds), "Expected two valid listener IDs")
		assert.True(t, listenerIds["listener1"], "Expected listener1 in map")
		assert.True(t, listenerIds["listener2"], "Expected listener2 in map")
	})

	t.Run("DuplicateID", func(t *testing.T) {
		config := &Config{
			Listeners: []listeners.Listener{
				{
					ID:      "listener1",
					Address: ":8080",
					Options: options.NewHTTP(),
				},
				{
					ID:      "listener1", // Duplicate ID
					Address: ":9090",
					Options: options.NewGRPC(),
				},
			},
		}

		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs, "Expected validation errors")

		foundDupError := false
		for _, err := range errs {
			if errors.Is(err, ErrDuplicateID) {
				foundDupError = true
				break
			}
		}
		assert.True(t, foundDupError, "Expected to find duplicate ID error")
	})

	t.Run("DuplicateAddress", func(t *testing.T) {
		config := &Config{
			Listeners: []listeners.Listener{
				{
					ID:      "listener1",
					Address: ":8080",
					Options: options.NewHTTP(),
				},
				{
					ID:      "listener2",
					Address: ":8080", // Duplicate address
					Options: options.NewGRPC(),
				},
			},
		}

		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs, "Expected validation errors")

		foundDupError := false
		for _, err := range errs {
			if errors.Is(err, ErrDuplicateID) {
				foundDupError = true
				break
			}
		}
		assert.True(t, foundDupError, "Expected to find duplicate ID error")
	})

	t.Run("InvalidListener", func(t *testing.T) {
		config := &Config{
			Listeners: []listeners.Listener{
				{
					ID:      "listener1",
					Address: "", // Empty address is invalid
					Options: options.NewHTTP(),
				},
			},
		}

		_, errs := config.validateListeners()
		assert.NotEmpty(t, errs, "Expected validation errors")
		assert.True(t, len(errs) >= 1, "Expected at least one error for invalid listener")
	})
}

func TestConfig_validateEndpoints(t *testing.T) {
	validListenerIds := map[string]bool{
		"listener1": true,
		"listener2": true,
	}

	t.Run("ValidEndpoints", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
				},
				{
					ID:          "endpoint2",
					ListenerIDs: []string{"listener2"},
				},
			},
		}

		errs := config.validateEndpoints(validListenerIds)
		assert.Empty(t, errs, "Expected no validation errors")
	})

	t.Run("DuplicateID", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
				},
				{
					ID:          "endpoint1", // Duplicate ID
					ListenerIDs: []string{"listener2"},
				},
			},
		}

		errs := config.validateEndpoints(validListenerIds)
		assert.NotEmpty(t, errs, "Expected validation errors")
		assert.Len(t, errs, 1, "Expected one error for duplicate ID")
		assert.ErrorIs(t, errs[0], ErrDuplicateID)
	})

	t.Run("NonExistentListenerID", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"nonexistent"}, // Reference to nonexistent listener
				},
			},
		}

		errs := config.validateEndpoints(validListenerIds)
		assert.NotEmpty(t, errs, "Expected validation errors")
		assert.Len(t, errs, 1, "Expected one error for invalid listener reference")
		assert.ErrorIs(t, errs[0], ErrListenerNotFound)
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "",                      // Invalid: empty ID
					ListenerIDs: []string{"nonexistent"}, // Invalid: nonexistent listener
				},
			},
		}

		errs := config.validateEndpoints(validListenerIds)
		assert.NotEmpty(t, errs, "Expected validation errors")
		assert.Len(t, errs, 2, "Expected two errors: empty ID and invalid listener")
	})
}

func TestConfig_validateAppsAndRoutes(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := createValidDomainConfig(t)
		err := config.validateAppsAndRoutes()
		assert.NoError(t, err, "Expected no validation errors")
	})

	t.Run("InvalidApp", func(t *testing.T) {
		config := createValidDomainConfig(t)
		config.Apps = append(config.Apps, apps.App{
			ID: "", // Invalid: empty ID
		})

		err := config.validateAppsAndRoutes()
		assert.Error(t, err, "Expected validation errors")
		assert.ErrorContains(t, err, "empty ID")
	})

	t.Run("NonexistentAppReference", func(t *testing.T) {
		config := createValidDomainConfig(t)
		// Add a route that references a nonexistent app
		config.Endpoints[0].Routes = append(config.Endpoints[0].Routes, routes.Route{
			AppID:     "nonexistent",
			Condition: conditions.NewHTTP("/nonexistent"),
		})

		err := config.validateAppsAndRoutes()
		assert.Error(t, err, "Expected validation errors")
		assert.ErrorContains(t, err, "app not found")
	})
}

func TestConfig_collectRouteReferences(t *testing.T) {
	t.Run("CollectsAllReferences", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID: "endpoint1",
					Routes: []routes.Route{
						{AppID: "app1"},
						{AppID: "app2"},
					},
				},
				{
					ID: "endpoint2",
					Routes: []routes.Route{
						{AppID: "app3"},
					},
				},
			},
		}

		refs := config.collectRouteReferences()
		assert.Len(t, refs, 3, "Expected 3 route references")

		// Check all expected AppIDs are in the references
		appIDs := make(map[string]bool)
		for _, ref := range refs {
			appIDs[ref.AppID] = true
		}
		assert.True(t, appIDs["app1"], "Expected app1 in references")
		assert.True(t, appIDs["app2"], "Expected app2 in references")
		assert.True(t, appIDs["app3"], "Expected app3 in references")
	})

	t.Run("EmptyWithNoRoutes", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:     "endpoint1",
					Routes: []routes.Route{},
				},
			},
		}

		refs := config.collectRouteReferences()
		assert.Empty(t, refs, "Expected no route references")
	})

	t.Run("EmptyWithNoEndpoints", func(t *testing.T) {
		config := &Config{
			Endpoints: []endpoints.Endpoint{},
		}

		refs := config.collectRouteReferences()
		assert.Empty(t, refs, "Expected no route references")
	})
}

func TestConfig_validateRouteConflicts(t *testing.T) {
	t.Run("ConflictingRoutes", func(t *testing.T) {
		// Create a config with conflicting routes
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: conditions.NewHTTP("/conflict"),
						},
					},
				},
				{
					ID:          "endpoint2",
					ListenerIDs: []string{"listener1"}, // Same listener
					Routes: []routes.Route{
						{
							AppID:     "app2",
							Condition: conditions.NewHTTP("/conflict"), // Same path = conflict
						},
					},
				},
			},
		}

		err := config.validateRouteConflicts()
		assert.Error(t, err, "Expected error for conflicting routes")
		assert.Contains(t, err.Error(), "condition 'http_path:/conflict'")
	})

	t.Run("DifferentListenersNoConflict", func(t *testing.T) {
		// Create a config with same paths but different listeners
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: conditions.NewHTTP("/same-path"),
						},
					},
				},
				{
					ID:          "endpoint2",
					ListenerIDs: []string{"listener2"}, // Different listener
					Routes: []routes.Route{
						{
							AppID: "app2",
							Condition: conditions.NewHTTP(
								"/same-path",
							), // Same path, but different listener = no conflict
						},
					},
				},
			},
		}

		err := config.validateRouteConflicts()
		assert.NoError(t, err, "Expected no error for same paths on different listeners")
	})

	t.Run("DifferentPathsNoConflict", func(t *testing.T) {
		// Create a config with different paths on same listener
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: conditions.NewHTTP("/path1"),
						},
					},
				},
				{
					ID:          "endpoint2",
					ListenerIDs: []string{"listener1"}, // Same listener
					Routes: []routes.Route{
						{
							AppID:     "app2",
							Condition: conditions.NewHTTP("/path2"), // Different path = no conflict
						},
					},
				},
			},
		}

		err := config.validateRouteConflicts()
		assert.NoError(t, err, "Expected no error for different paths on same listener")
	})

	t.Run("SkipsNilConditions", func(t *testing.T) {
		// Create a config with nil conditions
		config := &Config{
			Endpoints: []endpoints.Endpoint{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"listener1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: nil, // Nil condition
						},
					},
				},
			},
		}

		err := config.validateRouteConflicts()
		assert.NoError(t, err, "Expected no error for nil conditions")
	})
}
