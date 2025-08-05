package echo

import (
	"context"
	"fmt"
	"net/http"
)

// App is a simple application that echoes request information
type App struct {
	id       string
	response string
}

// New creates a new EchoApp with a custom response
func New(id string, response string) *App {
	return &App{
		id:       id,
		response: response,
	}
}

// String returns the unique identifier of the application
func (a *App) String() string {
	return a.id
}

// HandleHTTP processes HTTP requests by returning the configured response
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	// Set content type to plain text for simple response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Write the configured response
	if _, err := w.Write([]byte(a.response)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
