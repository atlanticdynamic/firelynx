package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProtoFromFile(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.toml")

	// Write a simple test configuration
	testConfig := `
version = "v1"

[logging]
format = "json"
level = "info"
`
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err, "Failed to write test config file")

	// Test loading from file
	t.Run("ValidFile", func(t *testing.T) {
		config, err := LoadProtoFromFile(configPath)
		require.NoError(t, err, "Failed to load config from file")
		require.NotNil(t, config, "Config should not be nil")
		
		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
		require.NotNil(t, config.Logging, "Logging config should not be nil")
		assert.Equal(t, int32(2), int32(config.Logging.GetFormat()), "Expected JSON format")
		assert.Equal(t, int32(2), int32(config.Logging.GetLevel()), "Expected INFO level")
	})
	
	// Test file not found
	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.toml")
		_, err := LoadProtoFromFile(nonExistentPath)
		require.Error(t, err, "Expected error for non-existent file")
		assert.Contains(t, err.Error(), "config file does not exist", "Error should indicate file not found")
	})
	
	// Test unsupported file extension
	t.Run("UnsupportedExtension", func(t *testing.T) {
		// Create a file with wrong extension
		wrongExtPath := filepath.Join(tempDir, "test_config.json")
		err := os.WriteFile(wrongExtPath, []byte(testConfig), 0644)
		require.NoError(t, err, "Failed to write test file with wrong extension")
		
		_, err = LoadProtoFromFile(wrongExtPath)
		require.Error(t, err, "Expected error for unsupported file extension")
		assert.Contains(t, err.Error(), "unsupported config format", "Error should indicate unsupported format")
		assert.Contains(t, err.Error(), "only .toml is supported", "Error should mention TOML is the only supported format")
	})
}

func TestLoadProtoFromReader(t *testing.T) {
	// Test loading from reader with valid config
	t.Run("ValidReader", func(t *testing.T) {
		tomlConfig := `
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "reader_listener"
address = ":9090"
`
		reader := strings.NewReader(tomlConfig)
		config, err := LoadProtoFromReader(reader)
		require.NoError(t, err, "Failed to load config from reader")
		require.NotNil(t, config, "Config should not be nil after loading from reader")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
		
		// Check logging options
		require.NotNil(t, config.Logging, "Logging config should not be nil")
		assert.Equal(t, int32(2), int32(config.Logging.GetFormat()), "Expected JSON format")
		assert.Equal(t, int32(2), int32(config.Logging.GetLevel()), "Expected INFO level")

		// Check listener
		require.Len(t, config.Listeners, 1, "Expected 1 listener")
		assert.Equal(t, "reader_listener", config.Listeners[0].GetId(), "Expected listener ID 'reader_listener'")
		assert.Equal(t, ":9090", config.Listeners[0].GetAddress(), "Expected listener address ':9090'")
	})
	
	// Test reader error
	t.Run("ReaderError", func(t *testing.T) {
		// Create a reader that returns an error
		errReader := &errorReader{err: assert.AnError}
		_, err := LoadProtoFromReader(errReader)
		require.Error(t, err, "Expected error from reader")
		assert.Contains(t, err.Error(), "failed to read config data from reader", "Error should indicate reader failure")
	})
}

func TestLoadProtoFromBytes(t *testing.T) {
	// Test loading from bytes with valid config
	t.Run("ValidBytes", func(t *testing.T) {
		configBytes := []byte(`
version = "v1"

[logging]
format = "txt"
level = "debug"
`)

		config, err := LoadProtoFromBytes(configBytes)
		require.NoError(t, err, "Failed to load config from bytes")
		require.NotNil(t, config, "Config should not be nil after loading from bytes")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
		
		// Check logging options
		require.NotNil(t, config.Logging, "Logging config should not be nil")
		assert.Equal(t, int32(1), int32(config.Logging.GetFormat()), "Expected TXT format")
		assert.Equal(t, int32(1), int32(config.Logging.GetLevel()), "Expected DEBUG level")
	})
}

func TestNewLoaderFromBytes(t *testing.T) {
	validBytes := []byte(`version = "v1"`)
	
	loader, err := NewLoaderFromBytes(validBytes)
	require.NoError(t, err, "Failed to create loader from bytes")
	require.NotNil(t, loader, "Loader should not be nil")
	
	// Verify the loader works
	config, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config from loader")
	assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
}

func TestNewLoaderFromFilePath(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "loader_test.toml")
	
	testConfig := `version = "v1"`
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err, "Failed to write test config file")
	
	// Test valid file path
	t.Run("ValidFilePath", func(t *testing.T) {
		loader, err := NewLoaderFromFilePath(configPath)
		require.NoError(t, err, "Failed to create loader from file path")
		require.NotNil(t, loader, "Loader should not be nil")
		
		// Verify the loader works
		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config from loader")
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
	})
	
	// Test file not found
	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.toml")
		_, err := NewLoaderFromFilePath(nonExistentPath)
		require.Error(t, err, "Expected error for non-existent file")
		assert.Contains(t, err.Error(), "config file does not exist", "Error should indicate file not found")
	})
	
	// Test unsupported file extension
	t.Run("UnsupportedExtension", func(t *testing.T) {
		// Create a file with wrong extension
		wrongExtPath := filepath.Join(tempDir, "test_config.json")
		err := os.WriteFile(wrongExtPath, []byte(testConfig), 0644)
		require.NoError(t, err, "Failed to write test file with wrong extension")
		
		_, err = NewLoaderFromFilePath(wrongExtPath)
		require.Error(t, err, "Expected error for unsupported file extension")
		assert.Contains(t, err.Error(), "unsupported config format", "Error should indicate unsupported format")
	})
}

func TestNewLoaderFromReader(t *testing.T) {
	// Test valid reader
	t.Run("ValidReader", func(t *testing.T) {
		reader := strings.NewReader(`version = "v1"`)
		loader, err := NewLoaderFromReader(reader)
		require.NoError(t, err, "Failed to create loader from reader")
		require.NotNil(t, loader, "Loader should not be nil")
		
		// Verify the loader works
		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config from loader")
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
	})
	
	// Test reader error
	t.Run("ReaderError", func(t *testing.T) {
		// Create a reader that returns an error
		errReader := &errorReader{err: assert.AnError}
		_, err := NewLoaderFromReader(errReader)
		require.Error(t, err, "Expected error from reader")
		assert.Contains(t, err.Error(), "failed to read config data from reader", "Error should indicate reader failure")
	})
}

// errorReader implements a simple io.Reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}