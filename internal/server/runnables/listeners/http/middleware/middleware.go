package middleware

import (
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// Middleware is a type alias for go-supervisor's middleware type
type Middleware = httpserver.HandlerFunc
