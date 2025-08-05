package mocks

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// MockApp is a mock implementation of the App interface for testing
type MockApp struct {
	mock.Mock
}

// NewMockApp creates a new MockApp instance with optional ID preset
func NewMockApp(id string) *MockApp {
	mockApp := &MockApp{}
	if id != "" {
		mockApp.On("String").Return(id)
	}
	return mockApp
}

// String returns the mocked unique identifier of the application
func (m *MockApp) String() string {
	args := m.Called()
	return args.String(0)
}

// HandleHTTP is a mock implementation of the App.HandleHTTP method
func (m *MockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	args := m.Called(ctx, w, r)
	return args.Error(0)
}
