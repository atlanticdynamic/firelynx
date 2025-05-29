package apps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEchoApp(t *testing.T) {
	tests := []struct {
		name             string
		id               string
		config           any
		expectedResponse string
	}{
		{
			name:             "creates echo app with valid ID and no config",
			id:               "test-echo",
			config:           nil,
			expectedResponse: "test-echo", // defaults to ID when no config
		},
		{
			name:             "creates echo app ignoring non-echo config",
			id:               "echo-with-config",
			config:           struct{ foo string }{foo: "bar"}, // config is ignored
			expectedResponse: "echo-with-config",               // defaults to ID
		},
		{
			name:             "creates echo app with custom response",
			id:               "custom-echo",
			config:           &configEcho.EchoApp{Response: "Custom Response"},
			expectedResponse: "Custom Response",
		},
		{
			name:             "creates echo app with empty response string",
			id:               "empty-response-echo",
			config:           &configEcho.EchoApp{Response: ""},
			expectedResponse: "empty-response-echo", // defaults to ID when response is empty
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

			// Test the actual response by calling HandleHTTP
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			ctx := t.Context()

			err = echoApp.HandleHTTP(ctx, w, r, nil)
			require.NoError(t, err)

			// Verify the response matches expected
			assert.Equal(t, tt.expectedResponse, w.Body.String())
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
