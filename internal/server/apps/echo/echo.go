package echo

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
)

// App is a simple application that echoes request information
type App struct {
	id string
}

// New creates a new EchoApp
func New(id string) *App {
	return &App{
		id: id,
	}
}

// ID returns the unique identifier of the application
func (a *App) ID() string {
	return a.id
}

// HandleHTTP processes HTTP requests by echoing back request details
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	staticData map[string]any,
) error {
	// Create a response object with request details
	response := map[string]any{
		"app_id":      a.id,
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       r.URL.Query(),
		"headers":     headerToMap(r.Header),
		"static_data": staticData,
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Encode response to JSON
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

// headerToMap converts http.Header to a map for JSON serialization
func headerToMap(header http.Header) map[string][]string {
	result := make(map[string][]string)
	maps.Copy(result, header)
	return result
}
