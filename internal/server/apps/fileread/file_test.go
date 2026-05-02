package fileread

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- §A. Constructor: happy paths -------------------------------------------

func TestResolveFile_HappyPath(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hello.txt"), []byte("hello"), 0o600))

	f, err := ResolveFile(baseDir, "hello.txt", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })

	assert.Equal(t, "hello.txt", f.Name())
	assert.Equal(t, int64(5), f.Size())
	assert.WithinDuration(t, time.Now(), f.ModTime(), 5*time.Second)

	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestResolveFile_EmptyFile(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "empty.txt"), nil, 0o600))

	f, err := ResolveFile(baseDir, "empty.txt", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })

	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, int64(0), f.Size())
}

// --- §B. Constructor: input validation --------------------------------------

func TestResolveFile_MissingBaseDirectory(t *testing.T) {
	_, err := ResolveFile("", "anything.txt", false)
	require.ErrorIs(t, err, errMissingBaseDirectory)
}

func TestResolveFile_UnusableBaseDirectory(t *testing.T) {
	tmp := t.TempDir()
	notADir := filepath.Join(tmp, "regular_file.txt")
	require.NoError(t, os.WriteFile(notADir, []byte("data"), 0o600))

	_, err := ResolveFile(notADir, "anything.txt", false)
	require.ErrorIs(t, err, errUnusableBaseDirectory)
}

func TestResolveFile_MissingPath(t *testing.T) {
	baseDir := t.TempDir()
	_, err := ResolveFile(baseDir, "", false)
	require.ErrorIs(t, err, errMissingPath)
}

func TestResolveFile_AbsolutePath(t *testing.T) {
	baseDir := t.TempDir()
	_, err := ResolveFile(baseDir, "/etc/passwd", false)
	require.ErrorIs(t, err, errAbsolutePath)
}

func TestResolveFile_DirectoryTraversal(t *testing.T) {
	baseDir := t.TempDir()
	_, err := ResolveFile(baseDir, "../secret.txt", false)
	require.ErrorIs(t, err, errDirectoryTraversal)
}

// --- §C. Constructor: symlink topology --------------------------------------

func TestResolveFile_SymlinkSandbox(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "ok.txt"), []byte("ok"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))
	require.NoError(t, os.Symlink(
		filepath.Join(outside, "secret.txt"),
		filepath.Join(baseDir, "escape_file"),
	))
	require.NoError(t, os.Symlink("ok.txt", filepath.Join(baseDir, "alias.txt")))

	t.Run("symlinked dir escape blocked", func(t *testing.T) {
		_, err := ResolveFile(baseDir, "escape_dir/secret.txt", false)
		require.ErrorIs(t, err, errSymlinkEscape)
	})

	t.Run("direct symlink to outside file blocked", func(t *testing.T) {
		_, err := ResolveFile(baseDir, "escape_file", false)
		require.ErrorIs(t, err, errSymlinkEscape)
	})

	t.Run("symlink within sandbox works", func(t *testing.T) {
		f, err := ResolveFile(baseDir, "alias.txt", false)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, f.Close()) })
		got, err := f.ReadAllString()
		require.NoError(t, err)
		assert.Equal(t, "ok", got)
	})

	t.Run("missing file surfaces not-found", func(t *testing.T) {
		_, err := ResolveFile(baseDir, "missing.txt", false)
		require.ErrorIs(t, err, errFileNotFound)
	})
}

func TestResolveFile_AllowExternalSymlinks(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape_dir")))
	require.NoError(t, os.Symlink(
		filepath.Join(outside, "secret.txt"),
		filepath.Join(baseDir, "escape_file"),
	))

	f, err := ResolveFile(baseDir, "escape_dir/secret.txt", true)
	require.NoError(t, err)
	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.NoError(t, f.Close())
	assert.Equal(t, "secret", got)

	f, err = ResolveFile(baseDir, "escape_file", true)
	require.NoError(t, err)
	got, err = f.ReadAllString()
	require.NoError(t, err)
	assert.NoError(t, f.Close())
	assert.Equal(t, "secret", got)

	// Lexical defenses still apply.
	_, err = ResolveFile(baseDir, "../whatever", true)
	require.ErrorIs(t, err, errDirectoryTraversal)
	_, err = ResolveFile(baseDir, "/etc/passwd", true)
	require.ErrorIs(t, err, errAbsolutePath)
}

func TestResolveFile_SymlinkedBaseDirectory(t *testing.T) {
	realBase := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(realBase, "hello.txt"), []byte("hi"), 0o600))

	parent := t.TempDir()
	linkBase := filepath.Join(parent, "linked_base")
	require.NoError(t, os.Symlink(realBase, linkBase))

	f, err := ResolveFile(linkBase, "hello.txt", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })
	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, "hi", got)
}

func TestResolveFile_SymlinkChainEscapes(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(
		filepath.Join(outside, "secret.txt"),
		filepath.Join(baseDir, "b"),
	))
	require.NoError(t, os.Symlink(filepath.Join(baseDir, "b"), filepath.Join(baseDir, "a")))

	_, err := ResolveFile(baseDir, "a", false)
	require.ErrorIs(t, err, errSymlinkEscape)
}

func TestResolveFile_SymlinkChainReentersBase(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "file.txt"), []byte("inside"), 0o600))

	outside := t.TempDir()
	require.NoError(
		t,
		os.Symlink(filepath.Join(baseDir, "file.txt"), filepath.Join(outside, "loop")),
	)
	require.NoError(t, os.Symlink(filepath.Join(outside, "loop"), filepath.Join(baseDir, "in")))

	f, err := ResolveFile(baseDir, "in", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })
	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, "inside", got)
}

func TestResolveFile_RelativeSymlinkStaysInside(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "sibling.txt"), []byte("hello"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "sub"), 0o700))
	require.NoError(t, os.Symlink("../sibling.txt", filepath.Join(baseDir, "sub", "link")))

	f, err := ResolveFile(baseDir, "sub/link", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })
	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestResolveFile_RelativeSymlinkEscapes(t *testing.T) {
	shared := t.TempDir()
	baseDir := filepath.Join(shared, "base")
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "sub"), 0o700))
	outsideDir := filepath.Join(shared, "outside")
	require.NoError(t, os.MkdirAll(outsideDir, 0o700))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o600),
	)
	require.NoError(t, os.Symlink(
		"../../outside/secret.txt",
		filepath.Join(baseDir, "sub", "link"),
	))

	_, err := ResolveFile(baseDir, "sub/link", false)
	require.ErrorIs(t, err, errSymlinkEscape)
}

func TestResolveFile_DanglingSymlinkOutsideReturnsNotFound(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(
		t,
		os.Symlink("/nonexistent_dangling_target_xyz", filepath.Join(baseDir, "dangling")),
	)

	_, err := ResolveFile(baseDir, "dangling", false)
	require.ErrorIs(t, err, errFileNotFound)
}

func TestResolveFile_SymlinkLoop(t *testing.T) {
	baseDir := t.TempDir()
	loopPath := filepath.Join(baseDir, "loop")
	require.NoError(t, os.Symlink(loopPath, loopPath))

	_, err := ResolveFile(baseDir, "loop", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve target path")
}

// --- §D. Constructor: directory + missing-parent + perms --------------------

func TestResolveFile_TargetIsDirectory(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "subdir"), 0o700))

	_, err := ResolveFile(baseDir, "subdir", false)
	require.ErrorIs(t, err, errTargetIsDirectory)
	require.NotErrorIs(t, err, errFileNotFound)
}

func TestResolveFile_MissingParent(t *testing.T) {
	baseDir := t.TempDir()
	_, err := ResolveFile(baseDir, "nonexistent_dir/file.txt", false)
	require.ErrorIs(t, err, errFileNotFound)
}

func TestResolveFile_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission semantics not portable to Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission bits not enforced for root")
	}

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "locked.txt")
	require.NoError(t, os.WriteFile(target, []byte("nope"), 0o000))
	t.Cleanup(func() { assert.NoError(t, os.Chmod(target, 0o600)) })

	_, err := ResolveFile(baseDir, "locked.txt", false)
	require.ErrorIs(t, err, errReadFile)
	require.NotErrorIs(t, err, errFileNotFound)
	assert.NotContains(t, err.Error(), baseDir,
		"err must not leak resolved host path: %v", err)
	assert.NotContains(t, err.Error(), target,
		"err must not leak resolved host path: %v", err)
}

func TestResolveFile_AllowExternalSymlinks_MissingFileStillNotFound(t *testing.T) {
	outside := t.TempDir()
	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape")))

	_, err := ResolveFile(baseDir, "escape/missing.txt", true)
	require.ErrorIs(t, err, errFileNotFound)
}

func TestResolveFile_AllowExternalSymlinks_WithSymlinkedBase(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))

	realBase := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(realBase, "escape")))

	parent := t.TempDir()
	linkBase := filepath.Join(parent, "linked_base")
	require.NoError(t, os.Symlink(realBase, linkBase))

	f, err := ResolveFile(linkBase, "escape/secret.txt", true)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })
	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, "secret", got)
}

// --- §E. ServeHTTP method ---------------------------------------------------

func resolveAndServe(
	t *testing.T,
	baseDir, requestedPath, method string,
	headers map[string]string,
) *http.Response {
	t.Helper()
	f, err := ResolveFile(baseDir, requestedPath, false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })

	r := httptest.NewRequest(method, "/files?path="+requestedPath, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	f.ServeHTTP(rr, r)
	return rr.Result()
}

func TestResolvedFile_ServeHTTP_TextFile(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hi.txt"), []byte("hello"), 0o600))

	res := resolveAndServe(t, baseDir, "hi.txt", http.MethodGet, nil)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), body)
}

func TestResolvedFile_ServeHTTP_BinaryFile(t *testing.T) {
	baseDir := t.TempDir()
	// Repeat a byte sequence with high-bit bytes that DetectContentType
	// reliably classifies as application/octet-stream.
	payload := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd, 0xfc}, 64)
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "data.bin"), payload, 0o600))

	res := resolveAndServe(t, baseDir, "data.bin", http.MethodGet, nil)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/octet-stream", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, body, "bytes must round-trip exactly")
}

func TestResolvedFile_ServeHTTP_PNG(t *testing.T) {
	baseDir := t.TempDir()
	// Real 8-byte PNG signature followed by a minimal IHDR chunk so the
	// magic bytes are unambiguous to DetectContentType.
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // signature
		0x00, 0x00, 0x00, 0x0d, // IHDR length
		0x49, 0x48, 0x44, 0x52, // "IHDR"
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x06, 0x00, 0x00, 0x00, // bit depth, color type, etc.
		0x1f, 0x15, 0xc4, 0x89, // CRC
	}
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "logo.png"), png, 0o600))

	res := resolveAndServe(t, baseDir, "logo.png", http.MethodGet, nil)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "image/png", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, png, body)
}

func TestResolvedFile_ServeHTTP_HTMLExtensionBeatsSniffing(t *testing.T) {
	baseDir := t.TempDir()
	// .json with HTML content: extension wins, so Content-Type is JSON.
	require.NoError(t, os.WriteFile(
		filepath.Join(baseDir, "weird.json"),
		[]byte("<html><body>hi</body></html>"),
		0o600,
	))
	// .css with JSON-ish content: extension wins, so Content-Type is CSS.
	require.NoError(t, os.WriteFile(
		filepath.Join(baseDir, "weird.css"),
		[]byte(`{"a":1}`),
		0o600,
	))

	res := resolveAndServe(t, baseDir, "weird.json", http.MethodGet, nil)
	defer func() { assert.NoError(t, res.Body.Close()) }()
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

	res2 := resolveAndServe(t, baseDir, "weird.css", http.MethodGet, nil)
	defer func() { assert.NoError(t, res2.Body.Close()) }()
	assert.Equal(t, "text/css; charset=utf-8", res2.Header.Get("Content-Type"))
}

func TestResolvedFile_ServeHTTP_Range(t *testing.T) {
	baseDir := t.TempDir()
	payload := bytes.Repeat([]byte("abcd"), 256) // 1024 bytes
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "blob"), payload, 0o600))

	res := resolveAndServe(t, baseDir, "blob", http.MethodGet, map[string]string{
		"Range": "bytes=0-3",
	})
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusPartialContent, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), body)
}

func TestResolvedFile_ServeHTTP_IfModifiedSince(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hi.txt"), []byte("hello"), 0o600))

	// Baseline GET to capture Last-Modified.
	res := resolveAndServe(t, baseDir, "hi.txt", http.MethodGet, nil)
	require.NoError(t, res.Body.Close())
	lm := res.Header.Get("Last-Modified")
	require.NotEmpty(t, lm, "ServeContent must emit Last-Modified")

	// Repeat with If-Modified-Since: <Last-Modified>.
	res2 := resolveAndServe(t, baseDir, "hi.txt", http.MethodGet, map[string]string{
		"If-Modified-Since": lm,
	})
	defer func() { assert.NoError(t, res2.Body.Close()) }()
	assert.Equal(t, http.StatusNotModified, res2.StatusCode)
}

func TestResolvedFile_ServeHTTP_HEAD(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hi.txt"), []byte("hello"), 0o600))

	res := resolveAndServe(t, baseDir, "hi.txt", http.MethodHead, nil)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "5", res.Header.Get("Content-Length"))
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Empty(t, body, "HEAD must return empty body")
}

// --- §F. ReadAllString method -----------------------------------------------

func TestResolvedFile_ReadAllString(t *testing.T) {
	baseDir := t.TempDir()
	want := strings.Repeat("hello world\n", 100)
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "data.txt"), []byte(want), 0o600))

	f, err := ResolveFile(baseDir, "data.txt", false)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, f.Close()) })

	got, err := f.ReadAllString()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestResolvedFile_Error_NoHostPath asserts error messages from
// ReadAllString / Close paths never include the host path. This is the
// invariant other tests rely on for sanitization.
func TestResolvedFile_Error_NoHostPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission semantics not portable to Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission bits not enforced for root")
	}

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "locked.txt")
	require.NoError(t, os.WriteFile(target, []byte("nope"), 0o000))
	t.Cleanup(func() { assert.NoError(t, os.Chmod(target, 0o600)) })

	_, err := ResolveFile(baseDir, "locked.txt", false)
	require.ErrorIs(t, err, errReadFile)
	assert.NotContains(t, err.Error(), baseDir)
}
