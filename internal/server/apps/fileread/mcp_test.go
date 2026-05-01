package fileread

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRead_MCPToolOption_Registers(t *testing.T) {
	app := New(&Config{ID: "files", BaseDirectory: t.TempDir()})
	opt := app.MCPToolOption(app.MCPToolName())
	require.NotNil(t, opt)

	h, err := mcpio.NewHandler(opt, mcpio.WithName("test"))
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestFileRead_ToolFunc(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "hello.txt"), []byte("hello"), 0o600))
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	tests := []struct {
		name        string
		input       Request
		wantContent string
		wantErr     string
		wantCode    string // empty when no error expected
	}{
		{name: "valid read", input: Request{Path: "hello.txt"}, wantContent: "hello"},
		{name: "missing path", input: Request{}, wantErr: "path parameter is required", wantCode: mcpio.ErrorCodeValidation},
		{name: "absolute path", input: Request{Path: "/etc/passwd"}, wantErr: "absolute paths not allowed", wantCode: mcpio.ErrorCodeValidation},
		{name: "traversal", input: Request{Path: "../secret.txt"}, wantErr: "directory traversal not allowed", wantCode: mcpio.ErrorCodeValidation},
		{name: "missing file", input: Request{Path: "missing.txt"}, wantErr: "file not found: missing.txt", wantCode: mcpio.ErrorCodeValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := app.filereadToolFunc(t.Context(), nil, tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, out.Content)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			var te *mcpio.ToolError
			require.ErrorAs(t, err, &te, "input errors must surface as mcpio.ToolError")
			assert.Equal(t, tt.wantCode, te.Code)
		})
	}
}

// TestFileRead_ToolFunc_ServerSideErrorBecomesProcessingError verifies that
// configuration/I/O failures (here: a missing base_directory) surface as
// PROCESSING_ERROR rather than VALIDATION_ERROR — input validation errors
// are things the LLM can fix by retrying with a different path; server-side
// failures aren't.
func TestFileRead_ToolFunc_ServerSideErrorBecomesProcessingError(t *testing.T) {
	app := New(&Config{ID: "files"}) // No BaseDirectory.

	_, err := app.filereadToolFunc(t.Context(), nil, Request{Path: "anything.txt"})
	require.Error(t, err)
	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te, "server-side errors must still wrap as mcpio.ToolError")
	assert.Equal(t, mcpio.ErrorCodeProcessing, te.Code, "missing base_directory is a server-side failure")
}

// TestFileRead_ToolFunc_SymlinkEscapeIsValidationError ensures the new
// errSymlinkEscape sentinel routes to VALIDATION_ERROR (the LLM picked a
// path; it can pick a different one).
func TestFileRead_ToolFunc_SymlinkEscapeIsValidationError(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))
	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape")))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	_, err := app.filereadToolFunc(t.Context(), nil, Request{Path: "escape/secret.txt"})
	require.Error(t, err)
	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Equal(t, mcpio.ErrorCodeValidation, te.Code)
}

// TestFileRead_ToolFunc_UnusableBaseIsProcessingError verifies that pointing
// BaseDirectory at a regular file (not a directory) classifies as
// PROCESSING_ERROR — the LLM cannot fix server misconfiguration.
func TestFileRead_ToolFunc_UnusableBaseIsProcessingError(t *testing.T) {
	tmp := t.TempDir()
	notADir := filepath.Join(tmp, "regular_file.txt")
	require.NoError(t, os.WriteFile(notADir, []byte("data"), 0o600))

	app := New(&Config{ID: "files", BaseDirectory: notADir})

	_, err := app.filereadToolFunc(t.Context(), nil, Request{Path: "anything.txt"})
	require.Error(t, err)
	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Equal(t, mcpio.ErrorCodeProcessing, te.Code,
		"unusable base_directory is a server-side fault")
}

// TestFileRead_ToolFunc_SymlinkLoopIsProcessingError verifies the
// "failed to resolve target path" branch of readFile classifies as
// PROCESSING_ERROR rather than ValidationError.
func TestFileRead_ToolFunc_SymlinkLoopIsProcessingError(t *testing.T) {
	baseDir := t.TempDir()
	loopPath := filepath.Join(baseDir, "loop")
	require.NoError(t, os.Symlink(loopPath, loopPath))

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	_, err := app.filereadToolFunc(t.Context(), nil, Request{Path: "loop"})
	require.Error(t, err)
	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Equal(t, mcpio.ErrorCodeProcessing, te.Code)
}

// TestFileRead_ToolFunc_PermissionDeniedIsProcessingError verifies the
// "failed to read file" branch (real I/O failure) classifies as
// PROCESSING_ERROR. Skipped on Windows and when running as root.
func TestFileRead_ToolFunc_PermissionDeniedIsProcessingError(t *testing.T) {
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
		assert.NoError(t, os.Chmod(target, 0o600))
	})

	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	_, err := app.filereadToolFunc(t.Context(), nil, Request{Path: "locked.txt"})
	require.Error(t, err)
	var te *mcpio.ToolError
	require.ErrorAs(t, err, &te)
	assert.Equal(t, mcpio.ErrorCodeProcessing, te.Code)
}

// TestFileRead_ToolFunc_InputErrorsAllValidation is a regression guard for the
// inputErrors sentinel set in mcp.go. For every sentinel entry the table must
// contain a row that:
//
//  1. Drives ResolveFile down a path that returns that sentinel (errors.Is
//     match) — confirms the sentinel is reachable from a real client input.
//  2. Routes through filereadToolFunc as VALIDATION_ERROR — confirms the
//     classifier still treats that sentinel as input-class.
//
// If a future contributor adds a new sentinel to inputErrors but forgets to
// add a row, the closing seen[]-vs-inputErrors check fails. The split is
// deliberate: mcpio.ToolError doesn't implement Unwrap, so the wrapped error
// returned by the tool func cannot be matched with errors.Is once it has
// been recoded — we have to test the underlying ResolveFile separately.
func TestFileRead_ToolFunc_InputErrorsAllValidation(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o600))
	baseDir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(baseDir, "escape")))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "subdir"), 0o700))
	app := New(&Config{ID: "files", BaseDirectory: baseDir})

	cases := []struct {
		sentinel error
		input    Request
	}{
		{errMissingPath, Request{}},
		{errAbsolutePath, Request{Path: "/etc/passwd"}},
		{errDirectoryTraversal, Request{Path: "../x"}},
		{errSymlinkEscape, Request{Path: "escape/secret.txt"}},
		{errFileNotFound, Request{Path: "missing.txt"}},
		{errTargetIsDirectory, Request{Path: "subdir"}},
	}

	seen := map[error]bool{}
	for _, c := range cases {
		t.Run(c.sentinel.Error(), func(t *testing.T) {
			// (1) ResolveFile produces the expected sentinel.
			f, rawErr := ResolveFile(baseDir, c.input.Path, false)
			if f != nil {
				require.NoError(t, f.Close())
			}
			require.Error(t, rawErr)
			require.ErrorIs(t, rawErr, c.sentinel,
				"row meant to trigger %v but ResolveFile produced %v", c.sentinel, rawErr)

			// (2) filereadToolFunc classifies it as VALIDATION_ERROR.
			_, toolErr := app.filereadToolFunc(t.Context(), nil, c.input)
			require.Error(t, toolErr)
			var te *mcpio.ToolError
			require.ErrorAs(t, toolErr, &te)
			assert.Equal(t, mcpio.ErrorCodeValidation, te.Code,
				"sentinel %v must classify as VALIDATION_ERROR", c.sentinel)

			seen[c.sentinel] = true
		})
	}

	for _, sentinel := range inputErrors {
		assert.Truef(t, seen[sentinel],
			"inputErrors contains %v but no audit row drives a request that produces it; "+
				"add a row to TestFileRead_ToolFunc_InputErrorsAllValidation",
			sentinel)
	}
}
