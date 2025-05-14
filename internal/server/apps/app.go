package apps

import (
	"context"
	"net/http"
)

// Registry is an interface for a collection of applications
type Registry interface {
	// GetApp retrieves an application instance by ID
	GetApp(id string) (App, bool)
}

// App is the interface that all application handlers must implement
type App interface {
	// ID returns the unique identifier of the application
	String() string

	HTTPHandler
}

type HTTPHandler interface {
	HandleHTTP(
		ctx context.Context,
		resp http.ResponseWriter,
		req *http.Request,
		data map[string]any,
	) error
}
