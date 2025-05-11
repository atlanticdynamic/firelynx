// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestFromProtoConversions(t *testing.T) {
	// Tests for the FromProto function

	t.Run("ScriptApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with a script app
		pbApp := &pb.AppDefinition{
			Id: proto.String("test-script-app"),
			AppConfig: &pb.AppDefinition_Script{
				Script: &pb.AppScript{
					Evaluator: &pb.AppScript_Risor{
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
		pbApp := &pb.AppDefinition{
			Id: proto.String("test-composite-app"),
			AppConfig: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
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
	})

	t.Run("EchoApp", func(t *testing.T) {
		// Create a protobuf AppDefinition with an echo app
		pbApp := &pb.AppDefinition{
			Id: proto.String("test-echo-app"),
			AppConfig: &pb.AppDefinition_Echo{
				Echo: &pb.EchoApp{
					Response: proto.String("Hello, world!"),
				},
			},
		}

		// Convert to domain model
		app, err := fromProto(pbApp)

		// The test expects this to succeed, but currently it will fail
		// because the Echo app type is not handled in fromProto
		require.NoError(t, err, "Echo app conversion should succeed")
		require.NotNil(t, app, "Echo app should be converted")

		// Check conversion
		assert.Equal(t, "test-echo-app", app.ID)

		// Verify config type
		echoConfig, ok := app.Config.(*echo.EchoApp)
		require.True(t, ok, "Expected EchoApp config type")
		assert.Equal(t, "Hello, world!", echoConfig.Response)
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
		// Create multiple app definitions
		pbApps := []*pb.AppDefinition{
			{
				Id: proto.String("script-app"),
				AppConfig: &pb.AppDefinition_Script{
					Script: &pb.AppScript{
						Evaluator: &pb.AppScript_Risor{
							Risor: &pb.RisorEvaluator{
								Code: proto.String("return 'script'"),
							},
						},
					},
				},
			},
			{
				Id: proto.String("composite-app"),
				AppConfig: &pb.AppDefinition_CompositeScript{
					CompositeScript: &pb.AppCompositeScript{
						ScriptAppIds: []string{"app1", "app2"},
					},
				},
			},
			{
				Id: proto.String("echo-app"),
				AppConfig: &pb.AppDefinition_Echo{
					Echo: &pb.EchoApp{
						Response: proto.String("Echo response"),
					},
				},
			},
		}

		// Convert all apps
		apps, err := FromProto(pbApps)

		// This will fail until we add Echo app support
		require.NoError(t, err, "Should convert all apps")
		require.Len(t, apps, 3, "Should convert all 3 apps")

		// Verify each app was converted correctly
		assert.Equal(t, "script-app", apps[0].ID)
		assert.Equal(t, "composite-app", apps[1].ID)
		assert.Equal(t, "echo-app", apps[2].ID)
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
		assert.NotNil(t, pbApp.GetScript(), "Expected Script field to be set")
		assert.NotNil(t, pbApp.GetScript().GetRisor(), "Expected Risor evaluator to be set")
		assert.Equal(t, "return 'hello'", pbApp.GetScript().GetRisor().GetCode())
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

		// Verify that the EchoApp field is properly set
		// This will fail until we implement EchoApp support
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

		// Echo app
		echoApp := echo.New()
		echoApp.Response = "Echo response"

		apps := []App{
			{ID: "script-app", Config: scriptApp},
			{ID: "echo-app", Config: echoApp},
		}

		// Convert to protobuf
		pbApps := AppCollection(apps).ToProto()
		assert.Len(t, pbApps, 2, "Should convert all apps")

		// Verify each app was converted correctly
		assert.Equal(t, "script-app", pbApps[0].GetId())
		assert.NotNil(t, pbApps[0].GetScript(), "Script app should have Script field set")

		assert.Equal(t, "echo-app", pbApps[1].GetId())
		// This will fail until we implement EchoApp support
		assert.NotNil(t, pbApps[1].GetEcho(), "Echo app should have Echo field set")
	})

	t.Run("EmptyApps", func(t *testing.T) {
		pbApps := ToProto(nil)
		assert.Nil(t, pbApps, "Expected nil result for nil input")

		pbApps = ToProto([]App{})
		assert.Nil(t, pbApps, "Expected nil result for empty input")
	})
}
