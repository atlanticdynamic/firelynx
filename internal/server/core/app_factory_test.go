package core

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			mockApp := mocks.NewMockApp(id)
			mockApp.On("String").Return(id)
			mockApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil)
			return mockApp, nil
		},
	}

	// Test creating instances
	instances, err := CreateAppInstances(appCollection)

	// Verify
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Contains(t, instances, "test-echo")
	assert.Equal(t, "test-echo", instances["test-echo"].String())
}
