package mocks

import (
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/mock"
)

// MockRegistry implements the apps.Registry interface for testing
type MockRegistry struct {
	mock.Mock
}

// NewMockRegistry creates a new initialized MockRegistry
func NewMockRegistry() *MockRegistry {
	return &MockRegistry{}
}

// GetApp returns an app by ID from the registry
func (m *MockRegistry) GetApp(id string) (apps.App, bool) {
	args := m.Called(id)
	app, _ := args.Get(0).(apps.App)
	return app, args.Bool(1)
}

// RegisterApp registers an app in the registry
func (m *MockRegistry) RegisterApp(app apps.App) error {
	args := m.Called(app)
	return args.Error(0)
}

// UnregisterApp removes an app from the registry
func (m *MockRegistry) UnregisterApp(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
