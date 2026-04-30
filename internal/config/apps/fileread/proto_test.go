package fileread

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProtoRoundTrip(t *testing.T) {
	app := FromProto("files", &pbApps.FileReadApp{
		BaseDirectory: proto.String("/tmp/files"),
	})
	require.NotNil(t, app)
	assert.Equal(t, "files", app.ID)
	assert.Equal(t, "/tmp/files", app.BaseDirectory)

	protoApp, ok := app.ToProto().(*pbApps.FileReadApp)
	require.True(t, ok)
	assert.Equal(t, "/tmp/files", protoApp.GetBaseDirectory())
}
