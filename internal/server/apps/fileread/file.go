package fileread

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	errReadFile           = errors.New("failed to read file")
	errTargetIsDirectory  = errors.New("target is a directory")
)

// ResolvedFile is a sandboxed, opened, validated file ready to be served
// over HTTP or read as a string for MCP. Construct with ResolveFile.
// Caller MUST Close to release the underlying handle.
type ResolvedFile struct {
	requestedPath string
	realPath      string
	info          os.FileInfo
	handle        *os.File
}

// ResolveFile validates requestedPath against baseDirectory, resolves
// symlinks (rejecting escapes unless allowExternalSymlinks is set),
// opens the file, stats it, and rejects directories. The returned
// *ResolvedFile carries everything callers need to serve or read it.
func ResolveFile(
	baseDirectory, requestedPath string,
	allowExternalSymlinks bool,
) (*ResolvedFile, error) {
	realTarget, err := resolveSafePath(baseDirectory, requestedPath, allowExternalSymlinks)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(realTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", errFileNotFound, requestedPath)
		}
		return nil, fmt.Errorf("%w: %s", errReadFile, requestedPath)
	}

	info, err := f.Stat()
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("%w: %s", errReadFile, requestedPath),
			f.Close(),
		)
	}

	if info.IsDir() {
		return nil, errors.Join(
			fmt.Errorf("%w: %s", errTargetIsDirectory, requestedPath),
			f.Close(),
		)
	}

	return &ResolvedFile{
		requestedPath: requestedPath,
		realPath:      realTarget,
		info:          info,
		handle:        f,
	}, nil
}

func (f *ResolvedFile) Close() error       { return f.handle.Close() }
func (f *ResolvedFile) Name() string       { return filepath.Base(f.realPath) }
func (f *ResolvedFile) ModTime() time.Time { return f.info.ModTime() }
func (f *ResolvedFile) Size() int64        { return f.info.Size() }

// ServeHTTP writes the file to w with auto-detected Content-Type
// (mime.TypeByExtension first, then http.DetectContentType sniffing
// the first 512 bytes), Content-Length, Last-Modified, Range, and 304
// support — all delegated to http.ServeContent. Passing Name as the
// content name ensures Content-Disposition and ServeContent's own
// plain-text error responses never leak the resolved host path.
func (f *ResolvedFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, f.Name(), f.ModTime(), f.handle)
}

// ReadAllString reads the file from the current handle position to
// EOF and returns it as a string. Used by the MCP path. The returned
// error wraps errReadFile and contains only the requested path.
func (f *ResolvedFile) ReadAllString() (string, error) {
	b, err := io.ReadAll(f.handle)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errReadFile, f.requestedPath)
	}
	return string(b), nil
}

// resolveSafePath performs the lex + symlink + prefix-check pipeline.
// Returns the canonical absolute path to the requested file or one of
// the sentinel errors above.
func resolveSafePath(
	baseDirectory, requestedPath string,
	allowExternalSymlinks bool,
) (string, error) {
	if strings.TrimSpace(baseDirectory) == "" {
		return "", errMissingBaseDirectory
	}

	baseInfo, err := os.Stat(baseDirectory)
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
	absBase, err := filepath.Abs(baseDirectory)
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
	if !allowExternalSymlinks {
		if realTarget != realBase &&
			!strings.HasPrefix(realTarget, realBase+string(os.PathSeparator)) {
			return "", errSymlinkEscape
		}
	}

	return realTarget, nil
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
