package apps

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPHandler defines the interface for handling HTTP requests.
type HTTPHandler interface {
	// HandleHTTP processes HTTP requests for this application
	HandleHTTP(
		ctx context.Context,
		resp http.ResponseWriter,
		req *http.Request,
	) error
}

// App defines the interface that all applications must implement.
// This interface defines what applications can do within the server context.
// Consumers (like HTTP layer) may define their own structurally identical interfaces.
type App interface {
	fmt.Stringer // String() string - returns the unique identifier of the application
	HTTPHandler
}
