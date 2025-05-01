package core

import (
	"github.com/atlanticdynamic/firelynx/internal/config"
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

		// Map HTTP listeners from domain config to HTTP-specific config
		for _, l := range r.currentConfig.Listeners {
			// Skip non-HTTP listeners
			if l.Type != config.ListenerTypeHTTP {
				continue
			}

			// Get HTTP options
			httpOpts, ok := l.Options.(config.HTTPListenerOptions)
			if !ok {
				r.logger.Error("Invalid options type for HTTP listener", "id", l.ID)
				continue
			}

			// Extract timeouts with defaults
			readTimeout := http.DefaultReadTimeout
			if httpOpts.ReadTimeout != nil && httpOpts.ReadTimeout.AsDuration() > 0 {
				readTimeout = httpOpts.ReadTimeout.AsDuration()
			}

			writeTimeout := http.DefaultWriteTimeout
			if httpOpts.WriteTimeout != nil && httpOpts.WriteTimeout.AsDuration() > 0 {
				writeTimeout = httpOpts.WriteTimeout.AsDuration()
			}

			idleTimeout := http.DefaultIdleTimeout
			if httpOpts.IdleTimeout != nil && httpOpts.IdleTimeout.AsDuration() > 0 {
				idleTimeout = httpOpts.IdleTimeout.AsDuration()
			}

			drainTimeout := http.DefaultDrainTimeout
			if httpOpts.DrainTimeout != nil && httpOpts.DrainTimeout.AsDuration() > 0 {
				drainTimeout = httpOpts.DrainTimeout.AsDuration()
			}

			// Create route mapper to map endpoints
			routeMapper := http.NewRouteMapper(r.appRegistry, r.logger)

			// Map routes from domain config for this listener
			routes := routeMapper.MapRoutesFromDomainConfig(r.currentConfig, l.ID)

			// Create listener config
			listenerConfig := http.ListenerConfig{
				ID:           l.ID,
				Address:      l.Address,
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
				IdleTimeout:  idleTimeout,
				DrainTimeout: drainTimeout,
				Routes:       routes,
			}

			// Add to HTTP config
			httpConfig.Listeners = append(httpConfig.Listeners, listenerConfig)
		}

		r.logger.Debug("Built HTTP config",
			"listeners", len(httpConfig.Listeners))

		return httpConfig, nil
	}
}
