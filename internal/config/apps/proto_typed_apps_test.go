package apps

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/calculation"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/fileread"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestAppTypeConversions_TypedApps(t *testing.T) {
	assert.Equal(t, pb.AppDefinition_TYPE_CALCULATION, appTypeToProto(AppTypeCalculation))
	assert.Equal(t, pb.AppDefinition_TYPE_FILEREAD, appTypeToProto(AppTypeFileRead))
	assert.Equal(t, AppTypeCalculation, appTypeFromProto(pb.AppDefinition_TYPE_CALCULATION))
	assert.Equal(t, AppTypeFileRead, appTypeFromProto(pb.AppDefinition_TYPE_FILEREAD))
}

func TestFromProto_TypedApps(t *testing.T) {
	t.Run("calculation", func(t *testing.T) {
		appType := pb.AppDefinition_TYPE_CALCULATION
		pbApp := &pb.AppDefinition{
			Id:   proto.String("calc"),
			Type: &appType,
			Config: &pb.AppDefinition_Calculation{
				Calculation: &pbApps.CalculationApp{},
			},
		}

		app, err := fromProto(pbApp)
		require.NoError(t, err)
		assert.Equal(t, "calc", app.ID)
		_, ok := app.Config.(*calculation.App)
		assert.True(t, ok)
	})

	t.Run("fileread", func(t *testing.T) {
		appType := pb.AppDefinition_TYPE_FILEREAD
		pbApp := &pb.AppDefinition{
			Id:   proto.String("files"),
			Type: &appType,
			Config: &pb.AppDefinition_Fileread{
				Fileread: &pbApps.FileReadApp{
					BaseDirectory: proto.String("/tmp/files"),
				},
			},
		}

		app, err := fromProto(pbApp)
		require.NoError(t, err)
		assert.Equal(t, "files", app.ID)
		cfg, ok := app.Config.(*fileread.App)
		require.True(t, ok)
		assert.Equal(t, "/tmp/files", cfg.BaseDirectory)
	})
}

func TestToProto_TypedApps(t *testing.T) {
	collection := NewAppCollection(
		App{ID: "calc", Config: calculation.New("calc")},
		App{ID: "files", Config: &fileread.App{ID: "files", BaseDirectory: "/tmp/files"}},
	)

	got := collection.ToProto()
	require.Len(t, got, 2)
	assert.Equal(t, pb.AppDefinition_TYPE_CALCULATION, got[0].GetType())
	assert.NotNil(t, got[0].GetCalculation())
	assert.Equal(t, pb.AppDefinition_TYPE_FILEREAD, got[1].GetType())
	assert.Equal(t, "/tmp/files", got[1].GetFileread().GetBaseDirectory())
}
