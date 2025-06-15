package writers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriterType represents the type of writer to create
type WriterType string

const (
	WriterTypeStdout WriterType = "stdout"
	WriterTypeStderr WriterType = "stderr"
	WriterTypeFile   WriterType = "file"
)

// CreateWriter creates an io.Writer based on the output specification
// Supported formats:
//   - "stdout" or "" - writes to os.Stdout
//   - "stderr" - writes to os.Stderr
//   - "file:/path/to/file" - writes to file (creates directories if needed)
//   - "/path/to/file" - writes to file (creates directories if needed)
func CreateWriter(output string) (io.Writer, error) {
	switch {
	case output == "" || output == "stdout":
		return os.Stdout, nil
	case output == "stderr":
		return os.Stderr, nil
	case strings.HasPrefix(output, "file://"):
		filePath := strings.TrimPrefix(output, "file://")
		return createFileWriter(filePath)
	case isFilePath(output):
		// Direct file path
		return createFileWriter(output)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", output)
	}
}

// isFilePath determines if the string represents a local file path
func isFilePath(path string) bool {
	// Reject URLs with schemes other than file://
	if strings.Contains(path, "://") && !strings.HasPrefix(path, "file://") {
		return false
	}

	// Check for path-like patterns
	return strings.Contains(path, "/") || strings.Contains(path, "\\")
}

// createFileWriter creates a file writer, ensuring the directory exists
func createFileWriter(filePath string) (io.Writer, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Open file for writing (create if not exists, append if exists)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	return file, nil
}

// ParseWriterType determines the writer type from an output string
func ParseWriterType(output string) WriterType {
	if output == "" || output == "stdout" {
		return WriterTypeStdout
	}
	if output == "stderr" {
		return WriterTypeStderr
	}
	return WriterTypeFile
}
