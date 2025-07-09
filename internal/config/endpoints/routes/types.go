package routes

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
)

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	PathPrefix  string
	Method      string
	AppID       string
	App         *apps.App
	StaticData  map[string]any
	Middlewares middleware.MiddlewareCollection
}
