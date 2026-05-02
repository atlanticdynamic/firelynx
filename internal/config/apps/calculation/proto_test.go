package calculation

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoRoundTrip(t *testing.T) {
	app := FromProto("calc", &pbApps.CalculationApp{})
	require.NotNil(t, app)
	assert.Equal(t, "calc", app.ID)

	protoApp, ok := app.ToProto().(*pbApps.CalculationApp)
	require.True(t, ok)
	assert.NotNil(t, protoApp)
}

func TestFromProto_Nil(t *testing.T) {
	assert.Nil(t, FromProto("calc", nil))
}
