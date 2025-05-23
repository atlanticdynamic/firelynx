package apps

import (
	"context"
	"net/http"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEchoApp(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		config any
	}{
		{
			name:   "creates echo app with valid ID",
			id:     "test-echo",
			config: nil,
		},
		{
			name:   "creates echo app ignoring config",
			id:     "echo-with-config",
			config: struct{ foo string }{foo: "bar"}, // config is ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := createEchoApp(tt.id, tt.config)
			require.NoError(t, err)
			require.NotNil(t, app)

			// Verify it returns the correct ID
			assert.Equal(t, tt.id, app.String())

			// Verify it's actually an echo.App instance
			echoApp, ok := app.(*echo.App)
			assert.True(t, ok, "should return an echo.App instance")
			assert.NotNil(t, echoApp)
		})
	}
}

// MockApp is a test implementation of the App interface
type MockApp struct {
	id string
}

func (m *MockApp) String() string {
	return m.id
}

func (m *MockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	return nil
}

// mockInstantiator is a test instantiator that returns a MockApp
func mockInstantiator(id string, _ any) (App, error) {
	return &MockApp{id: id}, nil
}
