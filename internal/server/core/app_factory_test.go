package core

import (
	"context"
	"net/http"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAppInstances(t *testing.T) {
	// Setup test app collection
	appCollection := apps.AppCollection{
		{
			ID:     "test-echo",
			Config: echo.New(),
		},
	}

	// Register test app creator function
	originalImpls := serverApps.AvailableAppImplementations
	defer func() {
		// Restore original implementations after test
		serverApps.AvailableAppImplementations = originalImpls
	}()

	// Mock the app implementations
	serverApps.AvailableAppImplementations = map[string]serverApps.AppCreator{
		"echo": func(id string, _ any) (serverApps.App, error) {
			return &testApp{id: id}, nil
		},
	}

	// Test creating instances
	instances, err := CreateAppInstances(appCollection)

	// Verify
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Contains(t, instances, "test-echo")
	assert.Equal(t, "test-echo", instances["test-echo"].ID())
}

// testApp implements App for testing
type testApp struct {
	id string
}

func (a *testApp) ID() string {
	return a.id
}

func (a *testApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	return nil
}
