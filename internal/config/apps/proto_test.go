package apps

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestAppTypeConversions(t *testing.T) {
	t.Parallel()

	t.Run("DomainToProto", func(t *testing.T) {
		testCases := []struct {
			name     string
			appType  AppType
			expected pb.AppDefinition_Type
		}{
			{
				name:     "Script",
				appType:  AppTypeScript,
				expected: pb.AppDefinition_TYPE_SCRIPT,
			},
			{
				name:     "Composite",
				appType:  AppTypeComposite,
				expected: pb.AppDefinition_TYPE_COMPOSITE_SCRIPT,
			},
			{
				name:     "Echo",
				appType:  AppTypeEcho,
				expected: pb.AppDefinition_TYPE_ECHO,
			},
			{
				name:     "Unknown",
				appType:  AppTypeUnknown,
				expected: pb.AppDefinition_TYPE_UNSPECIFIED,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := appTypeToProto(tc.appType)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("ProtoToDomain", func(t *testing.T) {
		testCases := []struct {
			name     string
			pbType   pb.AppDefinition_Type
			expected AppType
		}{
			{
				name:     "Script",
				pbType:   pb.AppDefinition_TYPE_SCRIPT,
				expected: AppTypeScript,
			},
			{
				name:     "Composite",
				pbType:   pb.AppDefinition_TYPE_COMPOSITE_SCRIPT,
				expected: AppTypeComposite,
			},
			{
				name:     "Echo",
				pbType:   pb.AppDefinition_TYPE_ECHO,
				expected: AppTypeEcho,
			},
			{
				name:     "Unspecified",
				pbType:   pb.AppDefinition_TYPE_UNSPECIFIED,
				expected: AppTypeUnknown,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := appTypeFromProto(tc.pbType)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestFromProtoConversions(t *testing.T) {
	t.Parallel()

	t.Run("ScriptApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with a script app
		scriptType := pb.AppDefinition_TYPE_SCRIPT
		pbApp := &pb.AppDefinition{
			Id:   proto.String("test-script-app"),
			Type: &scriptType,
			Config: &pb.AppDefinition_Script{
				Script: &pbApps.ScriptApp{
					Evaluator: &pbApps.ScriptApp_Risor{
						Risor: &pbApps.RisorEvaluator{
							Source:  &pbApps.RisorEvaluator_Code{Code: "return 'hello'"},
							Timeout: durationpb.New(5 * time.Second),
						},
					},
				},
			},
		}

		// Convert to domain model
		app, err := fromProto(pbApp)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Check conversion
		assert.Equal(t, "test-script-app", app.ID)

		// Verify config type
		scriptConfig, ok := app.Config.(*scripts.AppScript)
		require.True(t, ok, "Expected AppScript config type")

		// Verify evaluator type
		risorEval, ok := scriptConfig.Evaluator.(*evaluators.RisorEvaluator)
		require.True(t, ok, "Expected RisorEvaluator type")

		assert.Equal(t, "return 'hello'", risorEval.Code)
	})

	t.Run("CompositeScriptApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with a composite script app
		compositeType := pb.AppDefinition_TYPE_COMPOSITE_SCRIPT
		pbApp := &pb.AppDefinition{
			Id:   proto.String("test-composite-app"),
			Type: &compositeType,
			Config: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pbApps.CompositeScriptApp{
					ScriptAppIds: []string{"script1", "script2"},
				},
			},
		}

		// Convert to domain model
		app, err := fromProto(pbApp)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Check conversion
		assert.Equal(t, "test-composite-app", app.ID)
		assert.NotNil(t, app.Config, "Config should not be nil")

		// Verify the config type
		_, ok := app.Config.(*composite.CompositeScript)
		require.True(t, ok, "Expected CompositeScript config type")
	})

	t.Run("EchoApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with an echo app
		echoType := pb.AppDefinition_TYPE_ECHO
		pbApp := &pb.AppDefinition{
			Id:   proto.String("test-echo-app"),
			Type: &echoType,
			Config: &pb.AppDefinition_Echo{
				Echo: &pbApps.EchoApp{
					Response: proto.String("Hello, world!"),
				},
			},
		}

		// Convert to domain model
		app, err := fromProto(pbApp)
		require.NoError(t, err, "Echo app conversion should succeed")
		require.NotNil(t, app, "Echo app should be converted")

		// Check conversion
		assert.Equal(t, "test-echo-app", app.ID)

		// Verify config type
		echoConfig, ok := app.Config.(*echo.EchoApp)
		require.True(t, ok, "Expected EchoApp config type")
		assert.Equal(t, "Hello, world!", echoConfig.Response)
	})

	t.Run("TypeMismatch", func(t *testing.T) {
		// Create a protobuf AppDefinition with mismatched type and config
		echoType := pb.AppDefinition_TYPE_ECHO
		pbApp := &pb.AppDefinition{
			Id:   proto.String("mismatched-app"),
			Type: &echoType,
			Config: &pb.AppDefinition_Script{
				Script: &pbApps.ScriptApp{
					Evaluator: &pbApps.ScriptApp_Risor{
						Risor: &pbApps.RisorEvaluator{
							Source: &pbApps.RisorEvaluator_Code{Code: "return 'hello'"},
						},
					},
				},
			},
		}

		// Conversion should fail due to type mismatch
		_, err := fromProto(pbApp)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})

	t.Run("NilApp", func(t *testing.T) {
		_, err := fromProto(nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAppDefinitionNil)
	})

	t.Run("EmptyAppConfig", func(t *testing.T) {
		pbApp := &pb.AppDefinition{
			Id: proto.String("empty-config-app"),
			// No AppConfig set
		}

		_, err := fromProto(pbApp)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNoConfigSpecified)
	})

	t.Run("MultipleApps", func(t *testing.T) {
		// Create multiple app definitions with types
		scriptType := pb.AppDefinition_TYPE_SCRIPT
		compositeType := pb.AppDefinition_TYPE_COMPOSITE_SCRIPT
		echoType := pb.AppDefinition_TYPE_ECHO

		pbApps := []*pb.AppDefinition{
			{
				Id:   proto.String("script-app"),
				Type: &scriptType,
				Config: &pb.AppDefinition_Script{
					Script: &pbApps.ScriptApp{
						Evaluator: &pbApps.ScriptApp_Risor{
							Risor: &pbApps.RisorEvaluator{
								Source: &pbApps.RisorEvaluator_Code{Code: "return 'script'"},
							},
						},
					},
				},
			},
			{
				Id:   proto.String("composite-app"),
				Type: &compositeType,
				Config: &pb.AppDefinition_CompositeScript{
					CompositeScript: &pbApps.CompositeScriptApp{
						ScriptAppIds: []string{"app1", "app2"},
					},
				},
			},
			{
				Id:   proto.String("echo-app"),
				Type: &echoType,
				Config: &pb.AppDefinition_Echo{
					Echo: &pbApps.EchoApp{
						Response: proto.String("Echo response"),
					},
				},
			},
		}

		// Convert all apps
		apps, err := FromProto(pbApps)
		require.NoError(t, err, "Should convert all apps")
		require.Len(t, apps, 3, "Should convert all 3 apps")

		// Verify each app was converted correctly
		assert.Equal(t, "script-app", apps[0].ID)
		assert.Equal(t, "composite-app", apps[1].ID)
		assert.Equal(t, "echo-app", apps[2].ID)

		// Verify config types
		_, ok := apps[0].Config.(*scripts.AppScript)
		assert.True(t, ok, "First app should be a script app")

		_, ok = apps[1].Config.(*composite.CompositeScript)
		assert.True(t, ok, "Second app should be a composite app")

		_, ok = apps[2].Config.(*echo.EchoApp)
		assert.True(t, ok, "Third app should be an echo app")
	})

	t.Run("FromProtoEmptyList", func(t *testing.T) {
		// Test with empty list
		apps, err := FromProto([]*pb.AppDefinition{})
		assert.NoError(t, err)
		assert.Nil(t, apps)

		// Test with nil list
		apps, err = FromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, apps)
	})

	t.Run("FromProtoErrorPropagation", func(t *testing.T) {
		// Create multiple apps where one has an error
		echoType := pb.AppDefinition_TYPE_ECHO

		pbApps := []*pb.AppDefinition{
			{
				Id:   proto.String("valid-echo"),
				Type: &echoType,
				Config: &pb.AppDefinition_Echo{
					Echo: &pbApps.EchoApp{
						Response: proto.String("valid"),
					},
				},
			},
			{
				// This app has nil as the entire definition, which should cause an error
				Id: nil, // This will cause fromProto to fail
			},
		}

		// Should fail on the second app and return an error
		_, err := FromProto(pbApps)
		assert.Error(t, err)
	})

	t.Run("UnknownConfigTypeError", func(t *testing.T) {
		// Create an app with an unknown config type by creating a custom protobuf message
		// This tests the default case in fromProto switch statement
		unkType := pb.AppDefinition_Type(999) // Invalid enum value
		pbApp := &pb.AppDefinition{
			Id:   proto.String("unknown-config-app"),
			Type: &unkType,
			// Don't set any Config field - this will trigger the nil case in switch
		}

		// Conversion should fail
		_, err := fromProto(pbApp)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNoConfigSpecified)
	})
}

func TestToProtoConversions(t *testing.T) {
	t.Parallel()

	t.Run("CompositeScriptApp", func(t *testing.T) {
		// Create a domain App with composite script config
		comp := composite.NewCompositeScript([]string{"script1", "script2"}, nil)
		app := App{
			ID:     "test-composite-app",
			Config: comp,
		}

		// Convert to protobuf
		pbApp := ToProto([]App{app})[0]

		// Verify conversion
		assert.Equal(t, "test-composite-app", pbApp.GetId())
		assert.Equal(
			t,
			pb.AppDefinition_TYPE_COMPOSITE_SCRIPT,
			pbApp.GetType(),
			"AppType should be COMPOSITE_SCRIPT",
		)
		assert.NotNil(t, pbApp.GetCompositeScript(), "Expected CompositeScript field to be set")
		assert.Equal(
			t,
			[]string{"script1", "script2"},
			pbApp.GetCompositeScript().GetScriptAppIds(),
		)
	})

	t.Run("EchoApp", func(t *testing.T) {
		// Create a domain App with echo config
		echoApp := echo.New()
		echoApp.Response = "Hello, world!"

		app := App{
			ID:     "test-echo-app",
			Config: echoApp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		assert.Len(t, pbApps, 1, "Expected 1 app to be converted")

		pbApp := pbApps[0]
		assert.Equal(t, "test-echo-app", pbApp.GetId())
		assert.Equal(t, pb.AppDefinition_TYPE_ECHO, pbApp.GetType(), "AppType should be ECHO")
		assert.NotNil(t, pbApp.GetEcho(), "Expected Echo field to be set")
		assert.Equal(t, "Hello, world!", pbApp.GetEcho().GetResponse())
	})

	t.Run("MultipleApps", func(t *testing.T) {
		// Create apps of different types
		timeout := 5 * time.Second

		// Script app
		risorEval := &evaluators.RisorEvaluator{
			Code:    "return 'script'",
			Timeout: timeout,
		}
		scriptApp := scripts.NewAppScript(nil, risorEval)

		// Composite app
		compApp := composite.NewCompositeScript([]string{"app1"}, nil)

		// Echo app
		echoApp := echo.New()
		echoApp.Response = "Echo response"

		apps := []App{
			{ID: "script-app", Config: scriptApp},
			{ID: "composite-app", Config: compApp},
			{ID: "echo-app", Config: echoApp},
		}

		// Convert to protobuf
		pbApps := AppCollection(apps).ToProto()
		assert.Len(t, pbApps, 3, "Should convert all apps")

		// Verify each app was converted correctly
		assert.Equal(t, "script-app", pbApps[0].GetId())
		assert.Equal(
			t,
			pb.AppDefinition_TYPE_SCRIPT,
			pbApps[0].GetType(),
			"First app should be SCRIPT type",
		)
		assert.NotNil(t, pbApps[0].GetScript(), "Script app should have Script field set")

		assert.Equal(t, "composite-app", pbApps[1].GetId())
		assert.Equal(
			t,
			pb.AppDefinition_TYPE_COMPOSITE_SCRIPT,
			pbApps[1].GetType(),
			"Second app should be COMPOSITE_SCRIPT type",
		)
		assert.NotNil(
			t,
			pbApps[1].GetCompositeScript(),
			"Composite app should have CompositeScript field set",
		)

		assert.Equal(t, "echo-app", pbApps[2].GetId())
		assert.Equal(
			t,
			pb.AppDefinition_TYPE_ECHO,
			pbApps[2].GetType(),
			"Third app should be ECHO type",
		)
		assert.NotNil(t, pbApps[2].GetEcho(), "Echo app should have Echo field set")
	})

	t.Run("EmptyApps", func(t *testing.T) {
		pbApps := ToProto(nil)
		assert.Nil(t, pbApps, "Expected nil result for nil input")

		pbApps = ToProto([]App{})
		assert.Nil(t, pbApps, "Expected nil result for empty input")
	})

	t.Run("ScriptAppWithStaticData", func(t *testing.T) {
		// Create a script app with static data
		risorEval := &evaluators.RisorEvaluator{
			Code:    "return 'hello'",
			Timeout: 5 * time.Second,
		}

		// Create static data
		staticDataStruct := &staticdata.StaticData{
			Data: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
		}
		scriptApp := scripts.NewAppScript(staticDataStruct, risorEval)

		app := App{
			ID:     "script-with-static",
			Config: scriptApp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Verify static data is converted
		assert.NotNil(t, pbApp.GetScript().GetStaticData(), "Static data should be present")
		assert.NotEmpty(t, pbApp.GetScript().GetStaticData().GetData(), "Static data should contain values")
	})

	t.Run("CompositeScriptWithStaticData", func(t *testing.T) {
		// Create static data
		staticDataStruct := &staticdata.StaticData{
			Data: map[string]any{
				"composite_key": "composite_value",
			},
		}
		comp := composite.NewCompositeScript([]string{"script1", "script2"}, staticDataStruct)

		app := App{
			ID:     "composite-with-static",
			Config: comp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Verify static data is converted
		assert.NotNil(t, pbApp.GetCompositeScript().GetStaticData(), "Static data should be present")
		assert.NotEmpty(t, pbApp.GetCompositeScript().GetStaticData().GetData(), "Static data should contain values")
	})

	t.Run("RisorEvaluator", func(t *testing.T) {
		// Create a script app with Risor evaluator
		risorEval := &evaluators.RisorEvaluator{
			Code:    "return 'risor test'",
			Timeout: 8 * time.Second,
		}
		scriptApp := scripts.NewAppScript(nil, risorEval)

		app := App{
			ID:     "risor-app",
			Config: scriptApp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Verify Risor evaluator is converted
		assert.NotNil(t, pbApp.GetScript().GetRisor(), "Risor evaluator should be present")
		assert.Equal(t, "return 'risor test'", pbApp.GetScript().GetRisor().GetCode())
	})

	t.Run("StarlarkEvaluator", func(t *testing.T) {
		// Create a script app with Starlark evaluator
		starlarkEval := &evaluators.StarlarkEvaluator{
			Code:    "result = 'starlark test'",
			Timeout: 10 * time.Second,
		}
		scriptApp := scripts.NewAppScript(nil, starlarkEval)

		app := App{
			ID:     "starlark-app",
			Config: scriptApp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Verify Starlark evaluator is converted
		assert.NotNil(t, pbApp.GetScript().GetStarlark(), "Starlark evaluator should be present")
		assert.Equal(t, "result = 'starlark test'", pbApp.GetScript().GetStarlark().GetCode())
	})

	t.Run("ExtismEvaluator", func(t *testing.T) {
		// Create a script app with Extism evaluator
		extismEval := &evaluators.ExtismEvaluator{
			Code:    "base64encodedwasm",
			Timeout: 15 * time.Second,
		}
		scriptApp := scripts.NewAppScript(nil, extismEval)

		app := App{
			ID:     "extism-app",
			Config: scriptApp,
		}

		// Convert to protobuf
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Verify Extism evaluator is converted
		assert.NotNil(t, pbApp.GetScript().GetExtism(), "Extism evaluator should be present")
		assert.Equal(t, "base64encodedwasm", pbApp.GetScript().GetExtism().GetCode())
	})

	t.Run("UnknownAppType", func(t *testing.T) {
		// Create an app with a custom config type that doesn't match any known types
		customApp := &customAppConfig{name: "unknown"}
		app := App{
			ID:     "unknown-app",
			Config: customApp,
		}

		// Convert to protobuf - should handle unknown type gracefully
		pbApps := ToProto([]App{app})
		pbApp := pbApps[0]

		// Should have UNSPECIFIED type for unknown app types
		assert.Equal(t, pb.AppDefinition_TYPE_UNSPECIFIED, pbApp.GetType())
		assert.Equal(t, "unknown-app", pbApp.GetId())
		// Config should be nil since we don't know how to convert unknown types
		assert.Nil(t, pbApp.GetConfig())
	})
}

// customAppConfig is a mock AppConfig for testing unknown types
type customAppConfig struct {
	name string
}

func (c *customAppConfig) Type() string                 { return "unknown" }
func (c *customAppConfig) Validate() error              { return nil }
func (c *customAppConfig) ToProto() any                 { return nil }
func (c *customAppConfig) String() string               { return c.name }
func (c *customAppConfig) ToTree() *fancy.ComponentTree { return nil }
