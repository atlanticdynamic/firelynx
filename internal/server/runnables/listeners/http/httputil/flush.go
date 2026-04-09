// Package httputil provides HTTP utility helpers used across the firelynx HTTP
// listener stack.
package httputil

import (
	"net/http"
	"reflect"
)

// FindFlusher searches for an http.Flusher in the response writer chain.
//
// It first checks if the writer directly implements http.Flusher, then walks
// through any Unwrap() chain (standard Go 1.20+ convention), and finally falls
// back to reflection to access an embedded ResponseWriter field. The reflection
// fallback handles middleware wrappers, such as go-supervisor's responseWriter,
// that embed http.ResponseWriter as an interface field without implementing
// http.Flusher or Unwrap() themselves.
func FindFlusher(w http.ResponseWriter) http.Flusher {
	if f, ok := w.(http.Flusher); ok {
		return f
	}
	// Walk through Unwrap() chains (standard Go 1.20+ convention).
	if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
		return FindFlusher(u.Unwrap())
	}
	// Fallback: use reflection to access an embedded ResponseWriter field.
	// This handles the case where the wrapper struct has an exported embedded
	// http.ResponseWriter field (named "ResponseWriter") but neither implements
	// http.Flusher nor provides an Unwrap() method.
	rv := reflect.ValueOf(w)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName("ResponseWriter")
		if field.IsValid() && !field.IsNil() {
			if inner, ok := field.Interface().(http.ResponseWriter); ok {
				return FindFlusher(inner)
			}
		}
	}
	return nil
}
