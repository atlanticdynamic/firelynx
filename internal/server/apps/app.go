package apps

import (
	"context"
	"net/http"
)

// App defines the interface that all applications must implement.
// This interface defines what applications can do within the server context.
// Consumers (like HTTP layer) may define their own structurally identical interfaces.
type App interface {
	// String returns the unique identifier of the application
	String() string

	// HandleHTTP processes HTTP requests for this application
	HandleHTTP(
		ctx context.Context,
		resp http.ResponseWriter,
		req *http.Request,
	) error
}
