package core

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	listenerHTTP "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http/wrapper"
	"github.com/robbyt/go-supervisor/runnables/composite"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// httpWrapper is a type alias for *wrapper.HttpServer, used for the generic type signature in the
// callback to ensure this callback is only used for configuring our wrapper implementation.
type httpWrapper = *wrapper.HttpServer

// GetHTTPListenersConfigCallback returns a callback function suitable for creating a composite.Runner
// that manages HTTP listeners. This callback config is what connects this "core" to the listeners,
// it maps the domain config to the actual runnable implementation.
func (r *Runner) GetHTTPListenersConfigCallback() func() (*composite.Config[httpWrapper], error) {
	return func() (*composite.Config[httpWrapper], error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		if r.currentConfig == nil {
			// Create empty config if none exists yet
			r.logger.Debug("No configuration available, returning empty listeners config")
			return composite.NewConfig[httpWrapper]("http-listeners", nil)
		}

		r.logger.Debug("Building HTTP listeners configuration",
			"listeners", len(r.currentConfig.Listeners))

		// Map each HTTP listener to a runnable entry
		var entries []composite.RunnableEntry[httpWrapper]

		// Create new HTTP servers for each listener in the config
		for _, l := range r.currentConfig.Listeners {
			// Skip non-HTTP listeners
			if l.Type != config.ListenerTypeHTTP {
				continue
			}

			// Create a route mapper to map endpoints to routes for this listener
			routeMapper := listenerHTTP.NewRouteMapper(r.appRegistry, r.logger)
			localRoutes := routeMapper.MapEndpointsForListener(r.currentConfig, l.ID)

			// Convert routes to the supervisor httpserver Routes format
			httpRoutes := convertRoutesToHttpServer(localRoutes, l.ID, r.appRegistry, r.logger)

			// Create a server wrapper for this listener
			serverWrapper, err := wrapper.NewHttpServer(&l, httpRoutes,
				wrapper.WithLogger(r.logger))
			if err != nil {
				r.logger.Error("Failed to create HTTP server wrapper",
					"id", l.ID, "error", err)
				continue
			}

			// Create a runnable entry
			entry := composite.RunnableEntry[httpWrapper]{
				Runnable: serverWrapper,
				Config:   nil, // No per-listener config needed for now
			}

			entries = append(entries, entry)
		}

		return composite.NewConfig[httpWrapper]("http-listeners", entries)
	}
}

// convertRoutesToHttpServer converts from localhttp.Route to suphttp.Route
func convertRoutesToHttpServer(
	routes []listenerHTTP.Route,
	listenerID string,
	registry apps.Registry,
	logger *slog.Logger,
) []httpserver.Route {
	var httpRoutes []httpserver.Route

	// Create a single app handler that will manage all routes
	appHandler := listenerHTTP.NewAppHandler(registry, routes, logger)

	// Convert to suphttp.Route objects for each route path
	for _, route := range routes {
		// Capture the current route for the closure
		routePath := route.Path

		// Create a handler function for this specific route
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			appHandler.ServeHTTP(w, r)
		}

		// Create a unique route ID by combining listener ID and path
		routeID := fmt.Sprintf("%s:%s", listenerID, routePath)

		// Create a new suphttp route with the path and handler
		httpRoute, err := httpserver.NewRoute(
			routeID,
			routePath,
			handlerFunc,
		)
		if err != nil {
			logger.Error("Failed to create HTTP route",
				"listener", listenerID,
				"path", routePath,
				"error", err,
			)
			continue
		}

		httpRoutes = append(httpRoutes, *httpRoute)
	}

	return httpRoutes
}
