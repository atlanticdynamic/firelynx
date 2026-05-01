package fileread

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRead_HandleHTTP(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hello.txt"), []byte("hello"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "nested"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "nested", "file.txt"), []byte("nested"), 0o600))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantContent string
		wantError   string
	}{
		{name: "valid read", body: `{"path":"hello.txt"}`, wantStatus: http.StatusOK, wantContent: "hello"},
		{name: "missing path", body: `{}`, wantStatus: http.StatusBadRequest, wantError: "path parameter is required"},
		{name: "absolute path", body: `{"path":"/etc/passwd"}`, wantStatus: http.StatusBadRequest, wantError: "absolute paths not allowed"},
		{name: "traversal", body: `{"path":"../secret.txt"}`, wantStatus: http.StatusBadRequest, wantError: "directory traversal not allowed"},
		{name: "missing file", body: `{"path":"missing.txt"}`, wantStatus: http.StatusBadRequest, wantError: "file not found: missing.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			err := app.HandleHTTP(t.Context(), rr, req)

			res := rr.Result()
			defer func() {
				require.NoError(t, res.Body.Close())
			}()
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

			var got Response
			require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
			if tt.wantError == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, got.Content)
				assert.Empty(t, got.Error)
			} else {
				require.Error(t, err)
				assert.Contains(t, got.Error, tt.wantError)
			}
		})
	}
}

func TestFileRead_HandleHTTP_MissingBaseDirectory(t *testing.T) {
	app := New(&Config{ID: "files"})
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewBufferString(`{"path":"hello.txt"}`))
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)

	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode)
	assert.Contains(t, rr.Body.String(), "base_directory is required")
}

func TestFileRead_String(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	assert.Equal(t, "files", app.String())
}

func TestFileRead_HandleHTTP_MethodNotAllowed(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)

	require.Error(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Result().StatusCode)
}

func TestFileRead_HandleHTTP_InvalidJSON(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewBufferString("{not json"))
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)
	require.Error(t, err)

	res := rr.Result()
	defer func() {
		require.NoError(t, res.Body.Close())
	}()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var got Response
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Contains(t, got.Error, "invalid JSON request")
}

func TestFileRead_readFile_SymlinkSandbox(t *testing.T) {
	// outside is a sibling tempdir holding a "secret" file the sandbox
	// must never expose, even via a symlink planted inside the base dir.
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	// In-sandbox file we expect to read normally.
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "ok.txt"), []byte("ok"), 0o600))
	// Symlink that points to a directory outside the sandbox.
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))
	// Symlink whose name is the file itself, pointing at outside content.
	require.NoError(
		t,
		os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(baseDir, "escape_file")),
	)
	// Legit symlink that stays inside the sandbox.
	require.NoError(t, os.Symlink("ok.txt", filepath.Join(baseDir, "alias.txt")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	t.Run("symlinked dir escape is blocked", func(t *testing.T) {
		got, err := app.readFile("escape_dir/secret.txt")
		require.ErrorIs(t, err, errSymlinkEscape)
		assert.Empty(t, got)
	})

	t.Run("direct symlink to outside file is blocked", func(t *testing.T) {
		got, err := app.readFile("escape_file")
		require.ErrorIs(t, err, errSymlinkEscape)
		assert.Empty(t, got)
	})

	t.Run("symlink within the sandbox still works", func(t *testing.T) {
		got, err := app.readFile("alias.txt")
		require.NoError(t, err)
		assert.Equal(t, "ok", got)
	})

	t.Run("missing file still surfaces not-found", func(t *testing.T) {
		got, err := app.readFile("missing.txt")
		require.ErrorIs(t, err, errFileNotFound)
		assert.Empty(t, got)
	})
}

// TestFileRead_readFile_AllowExternalSymlinks verifies the opt-in escape:
// when AllowExternalSymlinks is true, a symlinked path that resolves outside
// the base directory is followed instead of being rejected.
func TestFileRead_readFile_AllowExternalSymlinks(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))
	require.NoError(
		t,
		os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(baseDir, "escape_file")),
	)

	app := New(&Config{
		ID:                    "files",
		BaseDirectory:         baseDir,
		AllowExternalSymlinks: true,
	})

	got, err := app.readFile("escape_dir/secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "secret", got)

	got, err = app.readFile("escape_file")
	require.NoError(t, err)
	assert.Equal(t, "secret", got)

	// Lexical defenses still apply even with the escape allowed.
	_, err = app.readFile("../whatever")
	require.ErrorIs(t, err, errDirectoryTraversal)
	_, err = app.readFile("/etc/passwd")
	require.ErrorIs(t, err, errAbsolutePath)
}

// TestFileRead_readFile_SymlinkedBaseDirectory exercises the case where
// base_directory itself is a symlink. The prefix check has to compare
// resolved paths, otherwise legitimate reads under the resolved base would
// fail.
func TestFileRead_readFile_SymlinkedBaseDirectory(t *testing.T) {
	realBase := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(realBase, "hello.txt"), []byte("hi"), 0o600))

	parent := t.TempDir()
	linkBase := filepath.Join(parent, "linked_base")
	require.NoError(t, os.Symlink(realBase, linkBase))

	app := New(&Config{ID: "files", BaseDirectory: linkBase})
	got, err := app.readFile("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "hi", got)
}

// --- §A. Symlink topology ---------------------------------------------------

// TestFileRead_readFile_SymlinkChainEscapes proves the resolver follows the
// full symlink chain, not just one hop: base/a -> base/b -> outside/secret.
func TestFileRead_readFile_SymlinkChainEscapes(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	// b in base points at the outside file; a in base points at b.
	require.NoError(
		t,
		os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(baseDir, "b")),
	)
	require.NoError(t, os.Symlink(filepath.Join(baseDir, "b"), filepath.Join(baseDir, "a")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("a")
	require.ErrorIs(t, err, errSymlinkEscape)
	assert.Empty(t, got)
}

// TestFileRead_readFile_SymlinkChainReentersBase proves the policy checks the
// final resolved target, not intermediate hops: a chain that leaves and
// re-enters base is allowed because the final destination is in-bounds.
func TestFileRead_readFile_SymlinkChainReentersBase(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "file.txt"), []byte("inside"), 0o600))

	outside := t.TempDir()
	// Hop through outside, then back into base.
	require.NoError(t, os.Symlink(filepath.Join(baseDir, "file.txt"), filepath.Join(outside, "loop")))
	require.NoError(t, os.Symlink(filepath.Join(outside, "loop"), filepath.Join(baseDir, "in")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("in")
	require.NoError(t, err)
	assert.Equal(t, "inside", got)
}

// TestFileRead_readFile_RelativeSymlinkStaysInside proves that ".." inside a
// symlink's target is fine when the resolved path lands back in base. This is
// distinct from ".." in the request path itself, which is rejected lexically.
func TestFileRead_readFile_RelativeSymlinkStaysInside(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "sibling.txt"), []byte("hello"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "sub"), 0o700))
	require.NoError(t, os.Symlink("../sibling.txt", filepath.Join(baseDir, "sub", "link")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("sub/link")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

// TestFileRead_readFile_RelativeSymlinkEscapes proves that a relative symlink
// whose ".." chain escapes the base directory is blocked.
func TestFileRead_readFile_RelativeSymlinkEscapes(t *testing.T) {
	shared := t.TempDir()
	baseDir := filepath.Join(shared, "base")
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "sub"), 0o700))
	outsideDir := filepath.Join(shared, "outside")
	require.NoError(t, os.MkdirAll(outsideDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o600))

	// base/sub/link -> ../../outside/secret.txt resolves to shared/outside/secret.txt
	require.NoError(t, os.Symlink("../../outside/secret.txt", filepath.Join(baseDir, "sub", "link")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("sub/link")
	require.ErrorIs(t, err, errSymlinkEscape)
	assert.Empty(t, got)
}

// TestFileRead_readFile_DanglingSymlinkOutsideReturnsNotFound documents the
// behavior for a dangling symlink whose target is outside the base directory:
// the parent-fallback puts realTarget back inside base, the prefix check
// passes, then os.ReadFile fails with not-found because the dangling target
// doesn't exist. End result: errFileNotFound, not silent success.
func TestFileRead_readFile_DanglingSymlinkOutsideReturnsNotFound(t *testing.T) {
	baseDir := t.TempDir()
	// Pick a target path that almost certainly does not exist.
	require.NoError(
		t,
		os.Symlink("/nonexistent_dangling_target_xyz", filepath.Join(baseDir, "dangling")),
	)

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("dangling")
	require.ErrorIs(t, err, errFileNotFound)
	assert.Empty(t, got)
}

// TestFileRead_readFile_SymlinkLoop proves the resolver propagates ELOOP as
// "failed to resolve target path" rather than panicking or hanging.
func TestFileRead_readFile_SymlinkLoop(t *testing.T) {
	baseDir := t.TempDir()
	loopPath := filepath.Join(baseDir, "loop")
	require.NoError(t, os.Symlink(loopPath, loopPath))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("loop")
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "failed to resolve target path")
}

// --- §B. Path-handling edge cases ------------------------------------------

// TestFileRead_readFile_TargetIsDirectory documents the behavior when the
// resolved target is a directory: os.ReadFile fails and the error is wrapped
// with errReadFile (a server-side fault, not errFileNotFound).
func TestFileRead_readFile_TargetIsDirectory(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "subdir"), 0o700))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("subdir")
	require.ErrorIs(t, err, errReadFile)
	assert.NotErrorIs(t, err, errFileNotFound)
	assert.Empty(t, got)
}

// TestFileRead_readFile_EmptyFile proves an empty file reads cleanly as the
// empty string with no error.
func TestFileRead_readFile_EmptyFile(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "empty.txt"), nil, 0o600))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("empty.txt")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestFileRead_readFile_MissingParent exercises the parent-fallback branch
// when even the parent directory does not exist.
func TestFileRead_readFile_MissingParent(t *testing.T) {
	baseDir := t.TempDir()
	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("nonexistent_dir/file.txt")
	require.ErrorIs(t, err, errFileNotFound)
	assert.Empty(t, got)
}

// TestFileRead_readFile_PermissionDenied exercises a real I/O failure path
// (os.ReadFile fails after every check passed). On Unix-like systems with a
// non-root euid, a 0o000 file is unreadable. Skip on Windows or when running
// as root.
func TestFileRead_readFile_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission semantics not portable to Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission bits not enforced for root")
	}

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "locked.txt")
	require.NoError(t, os.WriteFile(target, []byte("nope"), 0o000))
	t.Cleanup(func() {
		// Restore perms so t.TempDir cleanup can remove it.
		assert.NoError(t, os.Chmod(target, 0o600))
	})

	app := New(&Config{ID: "files", BaseDirectory: baseDir})
	got, err := app.readFile("locked.txt")
	require.ErrorIs(t, err, errReadFile)
	require.NotErrorIs(t, err, errFileNotFound)
	assert.Empty(t, got)
	// Sanitization: the user-facing message must not leak the resolved
	// host path or the raw OS-level cause string.
	assert.NotContains(t, err.Error(), baseDir,
		"err must not leak resolved host path: %v", err)
}

// --- §C. allow_external_symlinks gating ------------------------------------

// TestFileRead_readFile_AllowExternalSymlinks_MissingFileStillNotFound proves
// the opt-in does not silently succeed for a missing file under an externally
// symlinked directory.
func TestFileRead_readFile_AllowExternalSymlinks_MissingFileStillNotFound(t *testing.T) {
	outside := t.TempDir()
	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape")))

	app := New(&Config{
		ID:                    "files",
		BaseDirectory:         baseDir,
		AllowExternalSymlinks: true,
	})
	got, err := app.readFile("escape/missing.txt")
	require.ErrorIs(t, err, errFileNotFound)
	assert.Empty(t, got)
}

// TestFileRead_readFile_AllowExternalSymlinks_WithSymlinkedBase proves the
// two symlink resolutions compose: the base is itself a symlink AND the
// requested target escapes via another symlink, and the opt-in still works.
func TestFileRead_readFile_AllowExternalSymlinks_WithSymlinkedBase(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	realBase := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(realBase, "escape")))

	parent := t.TempDir()
	linkBase := filepath.Join(parent, "linked_base")
	require.NoError(t, os.Symlink(realBase, linkBase))

	app := New(&Config{
		ID:                    "files",
		BaseDirectory:         linkBase,
		AllowExternalSymlinks: true,
	})
	got, err := app.readFile("escape/secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "secret", got)
}

// --- §D. HandleHTTP entry-point coverage -----------------------------------

// TestFileRead_HandleHTTP_SymlinkEscapeReturns400 proves the HTTP path is
// also defended; not just the MCP tool path.
func TestFileRead_HandleHTTP_SymlinkEscapeReturns400(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(
		http.MethodPost,
		"/files",
		bytes.NewBufferString(`{"path":"escape_dir/secret.txt"}`),
	)
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)
	require.Error(t, err)

	res := rr.Result()
	defer func() { require.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

	var got Response
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Empty(t, got.Content)
	assert.Contains(t, got.Error, "symlink escapes base directory")
}

// TestFileRead_HandleHTTP_TargetIsDirectoryReturns500 proves that server-side
// I/O failures (here: target resolves to a directory) map to 500, not 400 —
// and that the response body does not leak the resolved host path.
func TestFileRead_HandleHTTP_TargetIsDirectoryReturns500(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "subdir"), 0o700))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	req := httptest.NewRequest(
		http.MethodPost,
		"/files",
		bytes.NewBufferString(`{"path":"subdir"}`),
	)
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)
	require.Error(t, err)

	res := rr.Result()
	defer func() { require.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

	var got Response
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Empty(t, got.Content)
	assert.Contains(t, got.Error, "failed to read file")
	assert.NotContains(t, got.Error, baseDir,
		"response must not leak resolved host path: %s", got.Error)
}
