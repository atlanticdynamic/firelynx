package fileread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	errMissingBaseDirectory  = errors.New("base_directory is required")
	errUnusableBaseDirectory = errors.New(
		"base_directory must exist and be a directory",
	)
	errMissingPath        = errors.New("path parameter is required")
	errAbsolutePath       = errors.New("absolute paths not allowed")
	errDirectoryTraversal = errors.New("directory traversal not allowed")
	errSymlinkEscape      = errors.New("symlink escapes base directory")
	errFileNotFound       = errors.New("file not found")
)

// App is a file reading application that reads files from a configured base directory.
type App struct {
	id                    string
	baseDirectory         string
	allowExternalSymlinks bool
}

// Request defines the typed input parameters for file read requests.
type Request struct {
	Path string `json:"path" jsonschema:"Path to read, relative to base directory"`
}

// Response defines the typed output structure for file read responses.
type Response struct {
	Content string `json:"content"         jsonschema:"Contents of the requested file"`
	Error   string `json:"error,omitempty"`
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
func (a *App) String() string {
	return a.id
}

// HandleHTTP processes HTTP requests by reading files from the configured base directory.
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return fmt.Errorf("invalid method %s", r.Method)
	}

	w.Header().Set("Content-Type", "application/json")

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if writeErr := writeFileReadError(w, http.StatusBadRequest, "invalid JSON request"); writeErr != nil {
			return writeErr
		}
		return fmt.Errorf("failed to decode request: %w", err)
	}

	content, err := a.readFile(req.Path)
	if err != nil {
		if writeErr := writeFileReadError(w, http.StatusBadRequest, err.Error()); writeErr != nil {
			return writeErr
		}
		return fmt.Errorf("file read failed: %w", err)
	}

	if err := json.NewEncoder(w).Encode(Response{Content: content}); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

func (a *App) readFile(requestedPath string) (string, error) {
	if strings.TrimSpace(a.baseDirectory) == "" {
		return "", errMissingBaseDirectory
	}

	baseInfo, err := os.Stat(a.baseDirectory)
	if err != nil || !baseInfo.IsDir() {
		return "", errUnusableBaseDirectory
	}

	if strings.TrimSpace(requestedPath) == "" {
		return "", errMissingPath
	}

	if filepath.IsAbs(requestedPath) {
		return "", errAbsolutePath
	}
	if hasTraversalSegment(requestedPath) {
		return "", errDirectoryTraversal
	}

	cleanPath := filepath.Clean(requestedPath)
	absBase, err := filepath.Abs(a.baseDirectory)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}

	// Resolve symlinks in the configured base so the prefix check below
	// compares canonical paths. Otherwise base_directory itself being a
	// symlink trivially defeats the prefix test.
	realBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		return "", errUnusableBaseDirectory
	}

	absTarget := filepath.Join(realBase, cleanPath)

	// Resolve symlinks on the requested target so we detect any escape via
	// symlinks living inside the base directory (e.g. base/escape -> /etc).
	// If the file does not yet exist, fall back to resolving its parent so
	// a legitimate "missing file" still surfaces a clean not-found error.
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to resolve target path: %w", err)
		}
		parent, perr := filepath.EvalSymlinks(filepath.Dir(absTarget))
		if perr != nil {
			return "", fmt.Errorf("%w: %s", errFileNotFound, requestedPath)
		}
		realTarget = filepath.Join(parent, filepath.Base(absTarget))
	}

	// When the operator has opted in, symlinks pointing outside the
	// resolved base are intentionally trusted; skip the prefix check.
	if !a.allowExternalSymlinks {
		if realTarget != realBase && !strings.HasPrefix(realTarget, realBase+string(os.PathSeparator)) {
			return "", errSymlinkEscape
		}
	}

	contentBytes, err := os.ReadFile(realTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", errFileNotFound, requestedPath)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(contentBytes), nil
}

func hasTraversalSegment(path string) bool {
	for _, segment := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if segment == ".." {
			return true
		}
	}
	return false
}

func writeFileReadError(w http.ResponseWriter, status int, message string) error {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{Error: message}); err != nil {
		return fmt.Errorf("failed to encode error response: %w", err)
	}
	return nil
}
