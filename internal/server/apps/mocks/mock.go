package mocks

import (
	"context"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/mock"
)

// Verify that MockApp implements the App interface
var _ apps.App = (*MockApp)(nil)

// MockApp is a mock implementation of the App interface for testing
type MockApp struct {
	mock.Mock
}

// New creates a new MockApp instance with optional ID preset
func New(id string) *MockApp {
	mockApp := &MockApp{}
	if id != "" {
		mockApp.On("ID").Return(id)
	}
	return mockApp
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
