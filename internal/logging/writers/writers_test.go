package writers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWriter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		output     string
		wantType   WriterType
		shouldFail bool
	}{
		{
			name:     "empty string defaults to stdout",
			output:   "",
			wantType: WriterTypeStdout,
		},
		{
			name:     "stdout",
			output:   "stdout",
			wantType: WriterTypeStdout,
		},
		{
			name:     "stderr",
			output:   "stderr",
			wantType: WriterTypeStderr,
		},
		{
			name:     "file path",
			output:   "/tmp/test.log",
			wantType: WriterTypeFile,
		},
		{
			name:     "file protocol",
			output:   "file:///tmp/test.log",
			wantType: WriterTypeFile,
		},
		{
			name:       "unsupported format",
			output:     "redis://localhost:6379",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := CreateWriter(tt.output)

			if tt.shouldFail {
				require.Error(t, err)
				require.Nil(t, writer)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, writer)

			// Verify writer type by checking the underlying type
			switch tt.wantType {
			case WriterTypeStdout:
				assert.Equal(t, os.Stdout, writer)
			case WriterTypeStderr:
				assert.Equal(t, os.Stderr, writer)
			case WriterTypeFile:
				// For file writers, just verify it's not stdout/stderr
				assert.NotEqual(t, os.Stdout, writer)
				assert.NotEqual(t, os.Stderr, writer)
			}
		})
	}
}

func TestCreateFileWriter(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filePath string
		setup    func() error
		cleanup  func() error
	}{
		{
			name:     "create file in existing directory",
			filePath: filepath.Join(tmpDir, "test.log"),
		},
		{
			name:     "create file with nested directories",
			filePath: filepath.Join(tmpDir, "nested", "dir", "test.log"),
		},
		{
			name:     "append to existing file",
			filePath: filepath.Join(tmpDir, "existing.log"),
			setup: func() error {
				return os.WriteFile(
					filepath.Join(tmpDir, "existing.log"),
					[]byte("existing content\n"),
					0o644,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = ctx

			if tt.setup != nil {
				require.NoError(t, tt.setup())
			}

			writer, err := createFileWriter(tt.filePath)
			require.NoError(t, err)
			require.NotNil(t, writer)

			// Test writing to the file
			testContent := "test content\n"
			n, err := writer.Write([]byte(testContent))
			require.NoError(t, err)
			assert.Equal(t, len(testContent), n)

			// Verify file exists and has content
			content, err := os.ReadFile(tt.filePath)
			require.NoError(t, err)
			assert.Contains(t, string(content), testContent)

			// Close the file if it implements io.Closer
			if closer, ok := writer.(interface{ Close() error }); ok {
				require.NoError(t, closer.Close())
			}

			if tt.cleanup != nil {
				require.NoError(t, tt.cleanup())
			}
		})
	}
}

func TestParseWriterType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		output   string
		expected WriterType
	}{
		{
			name:     "empty string",
			output:   "",
			expected: WriterTypeStdout,
		},
		{
			name:     "stdout",
			output:   "stdout",
			expected: WriterTypeStdout,
		},
		{
			name:     "stderr",
			output:   "stderr",
			expected: WriterTypeStderr,
		},
		{
			name:     "file path",
			output:   "/var/log/app.log",
			expected: WriterTypeFile,
		},
		{
			name:     "file protocol",
			output:   "file:///var/log/app.log",
			expected: WriterTypeFile,
		},
		{
			name:     "relative file path",
			output:   "./logs/app.log",
			expected: WriterTypeFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWriterType(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}
