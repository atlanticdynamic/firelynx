// Package apps provides interfaces and registry for application handlers
package apps

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// Verify that MockApp implements the App interface
var _ App = (*MockApp)(nil)

// MockApp is a mock implementation of the App interface for testing
type MockApp struct {
	mock.Mock
}

// ID returns the mocked unique identifier of the application
func (m *MockApp) ID() string {
	args := m.Called()
	return args.String(0)
}

// HandleHTTP is a mock implementation of the App.HandleHTTP method
func (m *MockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	params map[string]any,
) error {
	args := m.Called(ctx, w, r, params)
	return args.Error(0)
}

// NewMockApp creates a new MockApp instance with optional ID preset
func NewMockApp(id string) *MockApp {
	mockApp := &MockApp{}
	if id != "" {
		mockApp.On("ID").Return(id)
	}
	return mockApp
}
