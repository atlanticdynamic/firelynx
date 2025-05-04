package core

import (
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
)

// GetHTTPConfigCallback returns a callback function that transforms the domain config
// into the HTTP-specific config format for the HTTP runner.
// This is the bridge between the domain config and the HTTP-specific config.
func (r *Runner) GetHTTPConfigCallback() http.ConfigCallback {
	return func() (*http.Config, error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Get latest domain config
		if r.currentConfig == nil {
			// No config available yet
			r.logger.Debug("No domain configuration available for HTTP config")
			return nil, nil
		}

		// Create a new HTTP-specific config
		httpConfig := &http.Config{
			Registry:  r.appRegistry,
			Listeners: []http.ListenerConfig{},
		}

		// Create route mapper for app validation
		routeMapper := http.NewRouteMapper(r.appRegistry, r.logger)

		// Map HTTP listeners from domain config to HTTP-specific config
		for _, l := range r.currentConfig.Listeners {
			// Skip non-HTTP listeners
			if l.GetType() != options.TypeHTTP {
				continue
			}

			// Get HTTP options
			httpOpts, ok := l.Options.(options.HTTP)
			if !ok {
				r.logger.Error("Invalid options type for HTTP listener", "id", l.ID)
				continue
			}

			// Collect HTTP routes for this listener
			var httpRouteConfigs []http.RouteConfig

			// Get all endpoints for this listener
			endpoints := r.currentConfig.GetEndpointsForListener(l.ID)
			for _, endpoint := range endpoints {
				r.logger.Debug("Processing endpoint", "listenerID", l.ID, "endpointID", endpoint.ID)

				// Get HTTP routes from this endpoint
				httpRoutes := endpoint.GetStructuredHTTPRoutes()
				for _, route := range httpRoutes {
					// Convert config.HTTPRoute to http.RouteConfig
					httpRouteConfig := http.RouteConfig{
						Path:       route.Path,
						AppID:      route.AppID,
						StaticData: route.StaticData,
					}
					httpRouteConfigs = append(httpRouteConfigs, httpRouteConfig)
				}
			}

			// Validate routes against app registry
			validRoutes := routeMapper.ValidateRoutes(httpRouteConfigs)

			// Create listener config
			listenerConfig := http.ListenerConfig{
				ID:           l.ID,
				Address:      l.Address,
				ReadTimeout:  httpOpts.GetReadTimeout(),
				WriteTimeout: httpOpts.GetWriteTimeout(),
				IdleTimeout:  httpOpts.GetIdleTimeout(),
				DrainTimeout: httpOpts.GetDrainTimeout(),
				Routes:       validRoutes,
			}

			// Add to HTTP config
			httpConfig.Listeners = append(httpConfig.Listeners, listenerConfig)
		}

		r.logger.Debug("Built HTTP config",
			"listeners", len(httpConfig.Listeners))

		return httpConfig, nil
	}
}
