package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/loader/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoaderFromFilePath(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.toml")

	// Write a simple test configuration
	testConfig := `
version = "v1"
`
	err := os.WriteFile(configPath, []byte(testConfig), 0o644)
	require.NoError(t, err, "Failed to write test config file")

	// Test loading from file
	t.Run("ValidFile", func(t *testing.T) {
		loader, err := NewLoaderFromFilePath(configPath)
		require.NoError(t, err, "Failed to create loader from file path")

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config from loader")
		require.NotNil(t, config, "Config should not be nil")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
	})

	// Test file not found
	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.toml")
		l, err := NewLoaderFromFilePath(nonExistentPath)
		assert.Nil(t, l, "Loader should be nil for non-existent file")
		require.Error(t, err, "Expected error for non-existent file")
		assert.ErrorIs(t, err, os.ErrNotExist, "Error should be os.ErrNotExist")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig, "Error should be ErrFailedToLoadConfig")
	})

	// Test unsupported file extension
	t.Run("UnsupportedExtension", func(t *testing.T) {
		// Create a file with wrong extension
		wrongExtPath := filepath.Join(tempDir, "test_config.json")
		err := os.WriteFile(wrongExtPath, []byte(testConfig), 0o644)
		require.NoError(t, err, "Failed to write test file with wrong extension")

		_, err = NewLoaderFromFilePath(wrongExtPath)
		require.Error(t, err, "Expected error for unsupported file extension")
		assert.ErrorIs(t, err, ErrUnsupportedExtension, "Error should be ErrUnsupportedExtension")
	})
}

func TestNewLoaderFromReader(t *testing.T) {
	// Test loading from reader with valid config
	t.Run("ValidReader", func(t *testing.T) {
		tomlConfig := `
version = "v1"

[[listeners]]
id = "reader_listener"
address = ":9090"
`
		reader := strings.NewReader(tomlConfig)
		loader, err := NewLoaderFromReader(reader, func(data []byte) Loader {
			return toml.NewTomlLoader(data)
		})
		require.NoError(t, err, "Failed to create loader from reader")
		require.NotNil(t, loader, "Loader should not be nil")

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config from reader")
		require.NotNil(t, config, "Config should not be nil after loading from reader")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")

		// Check logging options

		// Check listener
		require.Len(t, config.Listeners, 1, "Expected 1 listener")
		assert.Equal(
			t,
			"reader_listener",
			config.Listeners[0].GetId(),
			"Expected listener ID 'reader_listener'",
		)
		assert.Equal(
			t,
			":9090",
			config.Listeners[0].GetAddress(),
			"Expected listener address ':9090'",
		)
	})

	// Test reader error
	t.Run("ReaderError", func(t *testing.T) {
		// Create a reader that returns an error
		errReader := &errorReader{err: assert.AnError}
		_, err := NewLoaderFromReader(errReader, func(data []byte) Loader {
			return toml.NewTomlLoader(data)
		})
		require.Error(t, err, "Expected error from reader")
		assert.ErrorIs(t, err, ErrFailedToLoadConfig)
		assert.ErrorIs(t, err, assert.AnError, "Error should match the mock reader error")
	})
}

func TestNewLoaderFromBytes(t *testing.T) {
	// Test loading from bytes with valid config
	t.Run("ValidBytes", func(t *testing.T) {
		configBytes := []byte(`
version = "v1"
`)

		loader, err := NewLoaderFromBytes(configBytes, func(data []byte) Loader {
			return toml.NewTomlLoader(data)
		})
		require.NoError(t, err, "Failed to create loader from bytes")
		require.NotNil(t, loader, "Loader should not be nil")

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config from loader")
		require.NotNil(t, config, "Config should not be nil after loading from bytes")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
	})
}

// errorReader implements a simple io.Reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// TestLoaderFunc tests the LoaderFunc type
func TestLoaderFunc(t *testing.T) {
	// Create a custom loader implementation using a lambda
	customLoader := func(data []byte) Loader {
		// We can add custom processing logic here
		return toml.NewTomlLoader(data)
	}

	// Create a loader with custom implementation
	loader, err := NewLoaderFromBytes([]byte(`version = "v1"`), customLoader)
	require.NoError(t, err, "Failed to create loader from bytes with custom LoaderFunc")
	require.NotNil(t, loader, "Loader should not be nil")

	// Verify the loader works
	config, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config from loader")
	assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")

	// Test with an empty loader implementation
	emptyLoader := func(data []byte) Loader {
		// Return a minimal implementation that doesn't parse anything
		return &testLoader{}
	}

	loader, err = NewLoaderFromBytes([]byte(`version = "v1"`), emptyLoader)
	require.NoError(t, err, "Failed to create loader from bytes with empty LoaderFunc")

	// This should return nil config since our testLoader does nothing
	config = loader.GetProtoConfig()
	assert.Nil(t, config, "Config should be nil from empty loader")
}

// testLoader is a minimal implementation of Loader for testing
type testLoader struct{}

func (l *testLoader) LoadProto() (*pbSettings.ServerConfig, error) {
	return nil, nil
}

func (l *testLoader) GetProtoConfig() *pbSettings.ServerConfig {
	return nil
}
