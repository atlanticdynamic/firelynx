package fileread

import (
	"context"
	"errors"
	"io"
	"net/http"
)

// httpServableFile is what HandleHTTP needs from a resolved file.
// *ResolvedFile satisfies it.
type httpServableFile interface {
	io.Closer
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// App is a file reading application that reads files from a configured base directory.
type App struct {
	id                    string
	baseDirectory         string
	allowExternalSymlinks bool
}

// New creates a new fileread app from a Config DTO.
func New(cfg *Config) *App {
	return &App{
		id:                    cfg.ID,
		baseDirectory:         cfg.BaseDirectory,
		allowExternalSymlinks: cfg.AllowExternalSymlinks,
	}
}

// String returns the unique identifier of the application.
func (a *App) String() string { return a.id }

// resolveForHTTP is the App's hook to ResolveFile. Returning the
// interface (not the concrete type) keeps HandleHTTP coupled only to
// the surface it needs and makes the boundary explicit.
func (a *App) resolveForHTTP(requestedPath string) (httpServableFile, error) {
	return ResolveFile(a.baseDirectory, requestedPath, a.allowExternalSymlinks)
}

// HandleHTTP serves the requested file as a raw HTTP response. Method
// must be GET or HEAD; the file path is passed via ?path=… query
// parameter. Content-Type is auto-detected (extension first, then
// sniffing). Range, conditional GET, and HEAD are handled via
// http.ServeContent.
//
// HandleHTTP always returns nil after writing a response — the HTTP
// adapter writes its own 500 on any non-nil return, which would
// double-write atop our response.
func (a *App) HandleHTTP(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	f, err := a.resolveForHTTP(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), httpStatusFor(err))
		return nil
	}
	defer func() {
		// Close error is not actionable: the response is already written
		// (returning a non-nil error here would cause the adapter to
		// double-write a 500 atop our reply), and the underlying *os.File
		// is read-only so Close has no flush to fail.
		_ = f.Close() //nolint:errcheck
	}()

	f.ServeHTTP(w, r)
	return nil
}

// httpStatusFor maps the ResolveFile sentinels to HTTP status codes.
// errFileNotFound has its own 404 case ahead of the inputErrors loop;
// it stays in inputErrors so the MCP path still classifies it as
// ValidationError.
func httpStatusFor(err error) int {
	if errors.Is(err, errFileNotFound) {
		return http.StatusNotFound
	}
	for _, sentinel := range inputErrors {
		if errors.Is(err, sentinel) {
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}
