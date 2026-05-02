package fileread

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HandleHTTP-level tests live here; resolution-logic and method-level
// tests for *ResolvedFile live in file_test.go.

func TestFileRead_String(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	assert.Equal(t, "files", app.String())
}

// --- Success paths ----------------------------------------------------------

func TestFileRead_HandleHTTP_RawFiles(t *testing.T) {
	baseDir := t.TempDir()

	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00,
		0x1f, 0x15, 0xc4, 0x89,
	}
	binBytes := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd, 0xfc}, 64)

	type fileFixture struct {
		name        string
		body        []byte
		wantContent string
	}
	fixtures := []fileFixture{
		{"hi.txt", []byte("hello"), "text/plain; charset=utf-8"},
		{"page.html", []byte("<html><body>hi</body></html>"), "text/html; charset=utf-8"},
		{"styles.css", []byte("body{}"), "text/css; charset=utf-8"},
		{"logo.png", pngBytes, "image/png"},
		{"data.bin", binBytes, "application/octet-stream"},
	}
	for _, f := range fixtures {
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, f.name), f.body, 0o600))
	}

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/files?path="+f.name, nil)
			rr := httptest.NewRecorder()

			require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

			res := rr.Result()
			defer func() { assert.NoError(t, res.Body.Close()) }()

			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, f.wantContent, res.Header.Get("Content-Type"))
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, f.body, body, "body bytes must round-trip exactly")
		})
	}
}

func TestFileRead_HandleHTTP_HEAD(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hi.txt"), []byte("hello"), 0o600))
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(http.MethodHead, "/files?path=hi.txt", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "5", res.Header.Get("Content-Length"))
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestFileRead_HandleHTTP_RangeRequest(t *testing.T) {
	baseDir := t.TempDir()
	payload := bytes.Repeat([]byte("abcd"), 256)
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "blob"), payload, 0o600))
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(http.MethodGet, "/files?path=blob", nil)
	req.Header.Set("Range", "bytes=0-3")
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusPartialContent, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), body)
}

func TestFileRead_HandleHTTP_IfModifiedSince(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hi.txt"), []byte("hello"), 0o600))
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	// Baseline GET captures Last-Modified.
	req := httptest.NewRequest(http.MethodGet, "/files?path=hi.txt", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))
	res := rr.Result()
	require.NoError(t, res.Body.Close())
	lm := res.Header.Get("Last-Modified")
	require.NotEmpty(t, lm)

	// Repeat with If-Modified-Since: should be 304.
	req2 := httptest.NewRequest(http.MethodGet, "/files?path=hi.txt", nil)
	req2.Header.Set("If-Modified-Since", lm)
	rr2 := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr2, req2))
	res2 := rr2.Result()
	defer func() { assert.NoError(t, res2.Body.Close()) }()
	assert.Equal(t, http.StatusNotModified, res2.StatusCode)
}

func TestFileRead_HandleHTTP_AllowExternalSymlinks(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape")))

	app := New(&Config{
		ID:                    "files",
		BaseDirectory:         baseDir,
		AllowExternalSymlinks: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/files?path=escape/secret.txt", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("secret"), body)
}

// --- Error paths ------------------------------------------------------------

func TestFileRead_HandleHTTP_MethodNotAllowed(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	req := httptest.NewRequest(http.MethodPost, "/files", nil)
	rr := httptest.NewRecorder()

	require.NoError(t, app.HandleHTTP(t.Context(), rr, req),
		"HandleHTTP must return nil after writing the 405 response — non-nil triggers a double-write 500 in the adapter")

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	assert.Equal(t, "GET, HEAD", res.Header.Get("Allow"))
}

func TestFileRead_HandleHTTP_MissingBaseDirectory(t *testing.T) {
	app := New(&Config{ID: "files"})
	req := httptest.NewRequest(http.MethodGet, "/files?path=hi.txt", nil)
	rr := httptest.NewRecorder()

	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "base_directory is required")
}

func TestFileRead_HandleHTTP_MissingPathParam(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	rr := httptest.NewRecorder()

	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "path parameter is required")
}

func TestFileRead_HandleHTTP_NotFoundReturns404(t *testing.T) {
	baseDir := t.TempDir()
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(http.MethodGet, "/files?path=missing.txt", nil)
	rr := httptest.NewRecorder()

	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "file not found: missing.txt")
}

func TestFileRead_HandleHTTP_SymlinkEscapeReturns400(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(http.MethodGet, "/files?path=escape_dir/secret.txt", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "symlink escapes base directory")
}

func TestFileRead_HandleHTTP_TargetIsDirectoryReturns400(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "subdir"), 0o700))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(http.MethodGet, "/files?path=subdir", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, app.HandleHTTP(t.Context(), rr, req))

	res := rr.Result()
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "target is a directory")
	assert.NotContains(t, bodyStr, baseDir,
		"response must not leak resolved host path: %s", bodyStr)
}
