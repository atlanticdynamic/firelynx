package fileread

import (
	"os"
	"path/filepath"
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
	}{
		{name: "valid read", input: Request{Path: "hello.txt"}, wantContent: "hello"},
		{name: "missing path", input: Request{}, wantErr: "path parameter is required"},
		{name: "absolute path", input: Request{Path: "/etc/passwd"}, wantErr: "absolute paths not allowed"},
		{name: "traversal", input: Request{Path: "../secret.txt"}, wantErr: "directory traversal not allowed"},
		{name: "missing file", input: Request{Path: "missing.txt"}, wantErr: "file not found: missing.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := app.filereadToolFunc(t.Context(), nil, tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, out.Content)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
