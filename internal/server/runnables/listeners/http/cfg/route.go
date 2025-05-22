package cfg

import (
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// AppRoute represents a route with associated app information.
// This enhances the standard httpserver.Route with app instance linking.
type AppRoute struct {
	// ID is a unique identifier for the route
	ID string

	// Path is the URL path prefix the route matches
	Path string

	// AppID is the ID of the application that handles this route
	AppID string

	// App is the actual application instance that will handle requests
	App apps.App

	// StaticData contains any static data to be included with the request
	StaticData map[string]any
}

// RouteCollection represents a set of routes for a listener
type RouteCollection struct {
	// ListenerID is the ID of the listener these routes belong to
	ListenerID string

	// Routes are the routes for this listener
	Routes []AppRoute
}

// ToHTTPServerRoutes converts AppRoutes to standard httpserver.Route objects
// for use with the HTTP server. This is typically called after app instances
// have been linked to the routes.
func ToHTTPServerRoutes(routes []AppRoute) []httpserver.Route {
	httpRoutes := make([]httpserver.Route, 0, len(routes))

	for _, route := range routes {
		if route.App == nil {
			// Skip routes without app instances
			continue
		}

		// Create a handler function that calls the app
		handler := func(w http.ResponseWriter, r *http.Request) {
			// Create data map for the app
			data := make(map[string]any)

			// Add static data
			if route.StaticData != nil {
				for k, v := range route.StaticData {
					data[k] = v
				}
			}

			// Add path parameters if needed
			if len(r.URL.Path) > len(route.Path) {
				data["remainingPath"] = r.URL.Path[len(route.Path):]
			}

			// Call the app handler
			err := route.App.HandleHTTP(r.Context(), w, r, data)
			if err != nil {
				// Handle error with a 500
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}

		// Create the HTTP route with our handler
		httpRoute, err := httpserver.NewRoute(route.ID, route.Path, handler)
		if err != nil {
			// This shouldn't happen if the route was already validated
			continue
		}

		httpRoutes = append(httpRoutes, *httpRoute)
	}

	return httpRoutes
}
