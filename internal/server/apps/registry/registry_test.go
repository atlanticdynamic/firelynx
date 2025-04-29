package registry

import (
	"fmt"
	"sync"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleRegistry(t *testing.T) {
	registry := New()

	// Check that registry is initialized correctly
	assert.NotNil(t, registry, "Registry should not be nil")
	assert.NotNil(t, registry.apps, "Apps map should not be nil")
	assert.Empty(t, registry.apps, "Apps map should be empty")
}

func TestSimpleRegistry_GetApp(t *testing.T) {
	registry := New()

	// Test getting a non-existent app
	app, exists := registry.GetApp("non-existent")
	assert.False(t, exists, "App should not exist")
	assert.Nil(t, app, "App should be nil")

	// Add an app and test getting it
	testApp := mocks.NewMockApp("test-app")
	registry.apps["test-app"] = testApp

	app, exists = registry.GetApp("test-app")
	assert.True(t, exists, "App should exist")
	assert.Equal(t, testApp, app, "Retrieved app should match the stored app")
}

func TestSimpleRegistry_RegisterApp(t *testing.T) {
	registry := New()

	// Register a new app
	testApp := mocks.NewMockApp("test-app")
	err := registry.RegisterApp(testApp)
	require.NoError(t, err, "RegisterApp should not return an error")

	// Verify the app was registered
	assert.Len(t, registry.apps, 1, "Registry should have 1 app")
	app, exists := registry.apps["test-app"]
	assert.True(t, exists, "App should exist in registry")
	assert.Equal(t, testApp, app, "Registered app should match the stored app")

	// Register a different app with the same ID
	replacementApp := mocks.NewMockApp("test-app")
	err = registry.RegisterApp(replacementApp)
	require.NoError(t, err, "RegisterApp should not return an error when replacing an app")

	// Verify the app was replaced
	assert.Len(t, registry.apps, 1, "Registry should still have 1 app")
	app, exists = registry.apps["test-app"]
	assert.True(t, exists, "App should exist in registry")
	assert.Equal(t, replacementApp, app, "App should have been replaced")
	assert.NotEqual(t, testApp, app, "App should not be the original app")
}

func TestSimpleRegistry_UnregisterApp(t *testing.T) {
	registry := New()

	// Add some apps
	testApp1 := mocks.NewMockApp("test-app-1")
	testApp2 := mocks.NewMockApp("test-app-2")
	registry.apps["test-app-1"] = testApp1
	registry.apps["test-app-2"] = testApp2

	// Unregister an app
	err := registry.UnregisterApp("test-app-1")
	require.NoError(t, err, "UnregisterApp should not return an error")

	// Verify the app was removed
	assert.Len(t, registry.apps, 1, "Registry should have 1 app left")
	_, exists := registry.apps["test-app-1"]
	assert.False(t, exists, "App should not exist in registry")
	_, exists = registry.apps["test-app-2"]
	assert.True(t, exists, "Other app should still exist in registry")

	// Unregister a non-existent app (should not error)
	err = registry.UnregisterApp("non-existent")
	require.NoError(t, err, "UnregisterApp should not return an error for non-existent apps")
}

func TestSimpleRegistry_Concurrency(t *testing.T) {
	registry := New()
	numGoroutines := 100
	numOperationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(routineID int) {
			defer wg.Done()

			for j := range numOperationsPerGoroutine {
				// Create a unique ID for this operation
				appID := fmt.Sprintf("app-%d-%d", routineID, j%10)

				// Register app
				app := mocks.NewMockApp(appID)
				err := registry.RegisterApp(app)
				assert.NoError(t, err, "RegisterApp should not error under concurrent access")

				// Get app
				_, exists := registry.GetApp(appID)
				assert.True(t, exists, "App should exist after registration")

				// Every few operations, unregister an app (2 operations ago)
				if j%10 == 5 {
					if j >= 2 {
						unregisterID := fmt.Sprintf("app-%d-%d", routineID, (j-2)%10)
						err = registry.UnregisterApp(unregisterID)
						assert.NoError(
							t,
							err,
							"UnregisterApp should not error under concurrent access",
						)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	t.Log("Final registry apps count:", len(registry.apps))

	// Verify that the registry has the expected number of apps
	expectedAppCount := numGoroutines * numOperationsPerGoroutine / 10
	assert.LessOrEqual(
		t,
		len(registry.apps),
		expectedAppCount,
		"Registry should have fewer or equal apps than expected",
	)
}
