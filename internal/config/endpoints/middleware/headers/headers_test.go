package headers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaders(t *testing.T) {
	t.Parallel()

	headers := NewHeaders()
	assert.NotNil(t, headers.SetHeaders)
	assert.NotNil(t, headers.AddHeaders)
	assert.NotNil(t, headers.RemoveHeaders)
	assert.Empty(t, headers.SetHeaders)
	assert.Empty(t, headers.AddHeaders)
	assert.Empty(t, headers.RemoveHeaders)
}

func TestHeaders_Type(t *testing.T) {
	t.Parallel()

	headers := NewHeaders()
	assert.Equal(t, "headers", headers.Type())
}

func TestHeaders_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid configuration", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Content-Type":    "application/json",
				"Cache-Control":   "no-cache",
				"X-Custom-Header": "custom-value",
			},
			AddHeaders: map[string]string{
				"Set-Cookie":     "session=abc123",
				"X-Multi-Header": "value1",
			},
			RemoveHeaders: []string{
				"Server",
				"X-Powered-By",
			},
		}

		err := headers.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty configuration is valid", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders()
		err := headers.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid set header - empty name", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"": "value",
			},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "header name cannot be empty")
	})

	t.Run("invalid add header - empty name", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			AddHeaders: map[string]string{
				"  ": "value",
			},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "header name cannot be empty")
	})

	t.Run("invalid remove header - empty name", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			RemoveHeaders: []string{"Valid-Header", "", "Another-Valid"},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "remove header name cannot be empty")
	})

	t.Run("invalid header characters", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Invalid\nHeader": "value",
			},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid header name")
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"": "empty-name",
			},
			AddHeaders: map[string]string{
				"Invalid\nHeader": "bad-char",
			},
			RemoveHeaders: []string{"", "Valid"},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "header name cannot be empty")
		assert.Contains(t, err.Error(), "remove header name cannot be empty")
	})
}

func TestHeaders_String(t *testing.T) {
	t.Parallel()

	t.Run("empty configuration", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders()
		assert.Equal(t, "No header operations configured", headers.String())
	})

	t.Run("only set headers", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
		}

		result := headers.String()
		assert.Equal(t, "Set: 2 headers", result)
	})

	t.Run("only add headers", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
			},
		}

		result := headers.String()
		assert.Equal(t, "Add: 1 headers", result)
	})

	t.Run("only remove headers", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			RemoveHeaders: []string{"Server", "X-Powered-By", "X-AspNet-Version"},
		}

		result := headers.String()
		assert.Equal(t, "Remove: 3 headers", result)
	})

	t.Run("all operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
				"X-Custom":   "value",
			},
			RemoveHeaders: []string{"Server"},
		}

		result := headers.String()
		assert.Equal(t, "Set: 1 headers, Add: 2 headers, Remove: 1 headers", result)
	})
}

func TestHeaders_ToTree(t *testing.T) {
	t.Parallel()

	t.Run("empty configuration", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders()
		tree := headers.ToTree()

		// Check that tree was created and contains expected content
		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())

		// Convert tree to string to verify content
		treeStr := tree.Tree().String()
		assert.Contains(t, treeStr, "Headers Middleware")
		assert.Contains(t, treeStr, "No header operations configured")
	})

	t.Run("configuration with all operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "custom-value",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
			},
			RemoveHeaders: []string{"Server", "X-Powered-By"},
		}

		tree := headers.ToTree()
		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())

		// Convert tree to string to verify content
		treeStr := tree.Tree().String()
		assert.Contains(t, treeStr, "Headers Middleware")
		assert.Contains(t, treeStr, "Set Headers:")
		assert.Contains(t, treeStr, "Add Headers:")
		assert.Contains(t, treeStr, "Remove Headers:")
		assert.Contains(t, treeStr, "Content-Type: application/json")
		assert.Contains(t, treeStr, "Set-Cookie: session=abc123")
		assert.Contains(t, treeStr, "Server")
		assert.Contains(t, treeStr, "X-Powered-By")
	})
}

func TestValidateHeader(t *testing.T) {
	t.Parallel()

	t.Run("valid headers", func(t *testing.T) {
		t.Parallel()

		validHeaders := map[string]string{
			"Content-Type":    "application/json",
			"Cache-Control":   "no-cache",
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
			"Set-Cookie":      "session=abc123; Path=/; HttpOnly",
		}

		for key, value := range validHeaders {
			err := validateHeader(key, value)
			assert.NoError(t, err, "Header '%s: %s' should be valid", key, value)
		}
	})

	t.Run("invalid header names", func(t *testing.T) {
		t.Parallel()

		invalidNames := []string{
			"",         // empty
			"  ",       // whitespace only
			"Header\n", // newline
			"Header\r", // carriage return
			"Header\t", // tab
		}

		for _, name := range invalidNames {
			err := validateHeader(name, "value")
			assert.Error(t, err, "Header name '%s' should be invalid", name)
		}
	})

	t.Run("edge case header values", func(t *testing.T) {
		t.Parallel()

		// Empty value should be valid
		err := validateHeader("X-Empty", "")
		assert.NoError(t, err)

		// Unicode value should be valid
		err = validateHeader("X-Unicode", "café")
		assert.NoError(t, err)

		// Long value should be valid
		longValue := "very-long-value-" + strings.Repeat("x", 1000)
		err = validateHeader("X-Long", longValue)
		assert.NoError(t, err)
	})
}
