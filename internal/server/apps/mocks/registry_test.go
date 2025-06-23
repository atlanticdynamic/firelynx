package mocks_test

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockRegistry_GetApp(t *testing.T) {
	// Verify MockRegistry implements expected interface
	assert.Implements(t, (*interface {
		GetApp(id string) (apps.App, bool)
	})(nil), &mocks.MockRegistry{})

	mockRegistry := mocks.NewMockRegistry()
	mockApp := mocks.NewMockApp("test-app")

	// Set expectation: GetApp should be called with "test-app" and return mockApp, true
	mockRegistry.On("GetApp", "test-app").Return(mockApp, true).Once()

	app, ok := mockRegistry.GetApp("test-app")
	assert.True(t, ok)
	assert.Equal(t, mockApp, app)
	mockRegistry.AssertExpectations(t)

	// Set expectation: GetApp called with missing app returns nil, false
	mockRegistry.On("GetApp", "missing-app").Return(nil, false).Once()
	app, ok = mockRegistry.GetApp("missing-app")
	assert.False(t, ok)
	assert.Nil(t, app)
	mockRegistry.AssertExpectations(t)
}

func TestMockRegistry_RegisterApp(t *testing.T) {
	mockRegistry := mocks.NewMockRegistry()
	mockApp := mocks.NewMockApp("test-app")

	// Set expectation: RegisterApp should be called and return nil error
	mockRegistry.On("RegisterApp", mockApp).Return(nil).Once()

	err := mockRegistry.RegisterApp(mockApp)
	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)

	// Set expectation: RegisterApp returns an error
	expectedErr := assert.AnError
	mockRegistry.On("RegisterApp", mock.Anything).Return(expectedErr).Once()
	err = mockRegistry.RegisterApp(mocks.NewMockApp("fail-app"))
	assert.ErrorIs(t, err, expectedErr)
	mockRegistry.AssertExpectations(t)
}

func TestMockRegistry_UnregisterApp(t *testing.T) {
	mockRegistry := mocks.NewMockRegistry()

	// Set expectation: UnregisterApp should be called and return nil error
	mockRegistry.On("UnregisterApp", "test-app").Return(nil).Once()
	err := mockRegistry.UnregisterApp("test-app")
	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)

	// Set expectation: UnregisterApp returns an error
	expectedErr := assert.AnError
	mockRegistry.On("UnregisterApp", "fail-app").Return(expectedErr).Once()
	err = mockRegistry.UnregisterApp("fail-app")
	assert.ErrorIs(t, err, expectedErr)
	mockRegistry.AssertExpectations(t)
}
