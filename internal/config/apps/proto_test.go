// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestAppTypeConversions(t *testing.T) {
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
	// Tests for the FromProto function

	t.Run("ScriptApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with a script app
		scriptType := pb.AppDefinition_TYPE_SCRIPT
		pbApp := &pb.AppDefinition{
			Id:   proto.String("test-script-app"),
			Type: &scriptType,
			Config: &pb.AppDefinition_Script{
				Script: &pb.ScriptApp{
					Evaluator: &pb.ScriptApp_Risor{
						Risor: &pb.RisorEvaluator{
							Code:    proto.String("return 'hello'"),
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
				CompositeScript: &pb.CompositeScriptApp{
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
				Echo: &pb.EchoApp{
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
				Script: &pb.ScriptApp{
					Evaluator: &pb.ScriptApp_Risor{
						Risor: &pb.RisorEvaluator{
							Code: proto.String("return 'hello'"),
						},
					},
				},
			},
		}

		// Conversion should fail due to type mismatch
		_, err := fromProto(pbApp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has type echo but no echo config")
	})

	t.Run("NilApp", func(t *testing.T) {
		_, err := fromProto(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app definition is nil")
	})

	t.Run("EmptyAppConfig", func(t *testing.T) {
		pbApp := &pb.AppDefinition{
			Id: proto.String("empty-config-app"),
			// No AppConfig set
		}

		_, err := fromProto(pbApp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown or empty config type")
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
					Script: &pb.ScriptApp{
						Evaluator: &pb.ScriptApp_Risor{
							Risor: &pb.RisorEvaluator{
								Code: proto.String("return 'script'"),
							},
						},
					},
				},
			},
			{
				Id:   proto.String("composite-app"),
				Type: &compositeType,
				Config: &pb.AppDefinition_CompositeScript{
					CompositeScript: &pb.CompositeScriptApp{
						ScriptAppIds: []string{"app1", "app2"},
					},
				},
			},
			{
				Id:   proto.String("echo-app"),
				Type: &echoType,
				Config: &pb.AppDefinition_Echo{
					Echo: &pb.EchoApp{
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
}

func TestToProtoConversions(t *testing.T) {
	// Tests for the ToProto function

	t.Run("ScriptApp", func(t *testing.T) {
		// Create a domain App with script config
		timeout := 5 * time.Second
		risorEval := &evaluators.RisorEvaluator{
			Code:    "return 'hello'",
			Timeout: timeout,
		}
		scriptApp := scripts.NewAppScript(nil, risorEval)

		app := App{
			ID:     "test-script-app",
			Config: scriptApp,
		}

		// Convert to protobuf
		pbApp := ToProto([]App{app})[0]

		// Verify conversion
		assert.Equal(t, "test-script-app", pbApp.GetId())
		assert.Equal(t, pb.AppDefinition_TYPE_SCRIPT, pbApp.GetType(), "AppType should be SCRIPT")
		assert.NotNil(t, pbApp.GetScript(), "Expected Script field to be set")
		assert.NotNil(t, pbApp.GetScript().GetRisor(), "Expected Risor evaluator to be set")
		assert.Equal(t, "return 'hello'", pbApp.GetScript().GetRisor().GetCode())
	})

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
}
