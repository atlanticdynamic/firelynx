package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// Test types used in unit tests

// testAppRegistry is a mock implementation of apps.Registry for testing
type testAppRegistry struct {
	apps map[string]apps.App
}

func (r *testAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *testAppRegistry) RegisterApp(app apps.App) error {
	if r.apps == nil {
		r.apps = make(map[string]apps.App)
	}
	r.apps[app.ID()] = app
	return nil
}

func (r *testAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}

// testApp is a mock implementation of apps.App for testing
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
	staticData map[string]any,
) error {
	response := map[string]string{
		"message": "Hello from test app " + a.id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}
