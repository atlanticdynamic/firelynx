package fileread

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	assert.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)
	assert.Contains(t, rr.Body.String(), "base_directory is required")
}
