package fileread

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Validate(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{name: "valid", app: &App{ID: "files", BaseDirectory: baseDir}},
		{name: "missing id", app: &App{BaseDirectory: baseDir}, wantErr: "missing required field: fileread app ID"},
		{name: "missing base directory", app: &App{ID: "files"}, wantErr: "missing required field: fileread base_directory"},
		{name: "base directory missing", app: &App{ID: "files", BaseDirectory: filepath.Join(baseDir, "missing")}, wantErr: "fileread base_directory is unusable"},
		{name: "base directory is file", app: &App{ID: "files", BaseDirectory: filePath}, wantErr: "fileread base_directory is not a directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// TestApp_Validate_BaseDirIsSymlinkToFile documents that pointing
// BaseDirectory at a symlink whose target is a regular file (not a
// directory) is rejected by domain validation. os.Stat follows the symlink
// to its target, so the IsDir() check catches the misuse before the runtime
// ever calls readFile.
func TestApp_Validate_BaseDirIsSymlinkToFile(t *testing.T) {
	scratch := t.TempDir()
	regularFile := filepath.Join(scratch, "regular.txt")
	require.NoError(t, os.WriteFile(regularFile, []byte("hi"), 0o600))

	linkPath := filepath.Join(scratch, "base_link")
	require.NoError(t, os.Symlink(regularFile, linkPath))

	app := &App{ID: "files", BaseDirectory: linkPath}
	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fileread base_directory is not a directory")
}

func TestApp_Interpolation(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.Setenv("FILEREAD_TEST_DIR", baseDir))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("FILEREAD_TEST_DIR"))
	})

	app := &App{
		ID:            "files-${FILEREAD_TEST_DIR}",
		BaseDirectory: "${FILEREAD_TEST_DIR}",
	}

	require.NoError(t, app.Validate())
	assert.Equal(t, "files-${FILEREAD_TEST_DIR}", app.ID)
	assert.Equal(t, baseDir, app.BaseDirectory)
}
