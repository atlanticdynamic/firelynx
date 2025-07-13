package echo

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/stretchr/testify/assert"
)

func TestEchoFromProto(t *testing.T) {
	tests := []struct {
		name  string
		proto *pbApps.EchoApp
		want  *EchoApp
	}{
		{
			name:  "normal conversion",
			proto: &pbApps.EchoApp{},
			want:  &EchoApp{},
		},
		{
			name:  "nil proto",
			proto: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EchoFromProto(tt.proto)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEchoApp_ToProto(t *testing.T) {
	echo := New()
	proto := echo.ToProto()

	// Verify the return value is not nil and is of the expected type
	assert.NotNil(t, proto)
	_, ok := proto.(*pbApps.EchoApp)
	assert.True(t, ok)
}
