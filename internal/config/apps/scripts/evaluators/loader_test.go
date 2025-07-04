package evaluators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLoaderFromSource(t *testing.T) {
	t.Run("code only", func(t *testing.T) {
		code := `func main() { return "hello" }`
		loader, err := createLoaderFromSource(code, "")

		require.NoError(t, err)
		assert.NotNil(t, loader)
	})

	t.Run("uri only - http", func(t *testing.T) {
		uri := "http://example.com/script.js"
		loader, err := createLoaderFromSource("", uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
	})

	t.Run("uri only - https", func(t *testing.T) {
		uri := "https://example.com/script.js"
		loader, err := createLoaderFromSource("", uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
	})

	t.Run("uri only - file without prefix", func(t *testing.T) {
		uri := "/path/to/script.js"
		loader, err := createLoaderFromSource("", uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
	})

	t.Run("uri only - file with prefix", func(t *testing.T) {
		uri := "file:///path/to/script.js"
		loader, err := createLoaderFromSource("", uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
	})

	t.Run("uri only - relative file with prefix resolves to absolute", func(t *testing.T) {
		uri := "file://relative/path/script.js"
		loader, err := createLoaderFromSource("", uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
		// Should resolve relative path to absolute and create loader successfully
	})

	t.Run("neither code nor uri", func(t *testing.T) {
		loader, err := createLoaderFromSource("", "")

		require.Error(t, err)
		assert.Nil(t, loader)
		assert.Contains(t, err.Error(), "neither code nor URI provided")
	})

	t.Run("both code and uri - code takes precedence", func(t *testing.T) {
		code := `func main() { return "hello" }`
		uri := "https://example.com/script.js"
		loader, err := createLoaderFromSource(code, uri)

		require.NoError(t, err)
		assert.NotNil(t, loader)
		// Code should take precedence, so this should create a string loader
	})
}
