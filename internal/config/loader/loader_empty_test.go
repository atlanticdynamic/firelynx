package loader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLoaderFromBytesEmpty tests NewLoaderFromBytes with empty data
func TestNewLoaderFromBytesEmpty(t *testing.T) {
	// Test with nil data
	t.Run("NilData", func(t *testing.T) {
		_, err := NewLoaderFromBytes(nil, func(data []byte) Loader {
			return &testLoader{}
		})
		require.Error(t, err, "Expected error for nil data")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig, "Error should be ErrFailedToLoadConfig")
		assert.ErrorIs(t, err, ErrNoSourceProvided, "Error should be ErrNoSourceProvided")
		assert.Contains(t, err.Error(), "no source provided to loader")
	})

	// Test with empty slice
	t.Run("EmptySlice", func(t *testing.T) {
		_, err := NewLoaderFromBytes([]byte{}, func(data []byte) Loader {
			return &testLoader{}
		})
		require.Error(t, err, "Expected error for empty slice")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig, "Error should be ErrFailedToLoadConfig")
		assert.ErrorIs(t, err, ErrNoSourceProvided, "Error should be ErrNoSourceProvided")
	})
}

// TestNewLoaderFromReaderEmpty tests NewLoaderFromReader with error cases
func TestNewLoaderFromReaderEmpty(t *testing.T) {
	// We can't test with a nil reader as it causes a panic in io.ReadAll
	// Let's use an error reader instead
	t.Run("ErrorReader", func(t *testing.T) {
		errReader := &errorReader{err: assert.AnError}
		_, err := NewLoaderFromReader(errReader, func(data []byte) Loader {
			return &testLoader{}
		})
		require.Error(t, err, "Expected error from reader")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig, "Error should be ErrFailedToLoadConfig")
		assert.ErrorIs(t, err, assert.AnError, "Error should match the mock reader error")
	})
}

// TestNewLoaderFromFilePathErrors tests additional error cases for NewLoaderFromFilePath
func TestNewLoaderFromFilePathErrors(t *testing.T) {
	// Test with directory instead of file
	t.Run("DirectoryInsteadOfFile", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := NewLoaderFromFilePath(tempDir)
		require.Error(t, err, "Expected error when trying to load a directory")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig, "Error should be ErrFailedToLoadConfig")
	})

	// Test with extension that doesn't match supported loaders
	t.Run("UnsupportedExtension", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := tempDir + "/config.unsupported"

		// Create the file so it exists
		err := os.WriteFile(configPath, []byte{}, 0o644)
		require.NoError(t, err, "Failed to create test file")

		_, err = NewLoaderFromFilePath(configPath)
		require.Error(t, err, "Expected error for unsupported extension")
		assert.ErrorIs(t, err, ErrUnsupportedExtension, "Error should be ErrUnsupportedExtension")
		assert.Contains(t, err.Error(), ".unsupported", "Error should contain the extension")
	})
}
