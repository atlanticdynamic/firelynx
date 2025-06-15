package middleware

import "github.com/robbyt/go-supervisor/runnables/httpserver"

// Instance is the interface all middleware instances must implement
type Instance interface {
	Middleware() httpserver.HandlerFunc
}
