package testutil

import (
	"context"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// MockApp implements the apps.App interface for testing
type MockApp struct {
	AppID        string
	HandleCalled bool
	LastRequest  *http.Request
	LastData     map[string]any
	ReturnError  error
}

func (m *MockApp) ID() string {
	return m.AppID
}

func (m *MockApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	staticData map[string]any,
) error {
	m.HandleCalled = true
	m.LastRequest = r
	m.LastData = staticData
	return m.ReturnError
}

// MockRegistry implements the apps.Registry interface for testing
type MockRegistry struct {
	Apps map[string]apps.App
}

func (m *MockRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := m.Apps[id]
	return app, ok
}

func (m *MockRegistry) RegisterApp(app apps.App) error {
	m.Apps[app.ID()] = app
	return nil
}

func (m *MockRegistry) UnregisterApp(id string) error {
	delete(m.Apps, id)
	return nil
}
