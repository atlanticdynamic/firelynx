package headers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaders(t *testing.T) {
	t.Parallel()

	headers := NewHeaders(nil, nil)
	assert.Nil(t, headers.Request)
	assert.Nil(t, headers.Response)
}

func TestNewHeaderOperations(t *testing.T) {
	t.Parallel()

	ops := NewHeaderOperations(RequestHeaderOperationsType)
	assert.Equal(t, RequestHeaderOperationsType, ops.Title)
	assert.NotNil(t, ops.SetHeaders)
	assert.NotNil(t, ops.AddHeaders)
	assert.NotNil(t, ops.RemoveHeaders)
	assert.Empty(t, ops.SetHeaders)
	assert.Empty(t, ops.AddHeaders)
	assert.Empty(t, ops.RemoveHeaders)
}

func TestHeaders_Type(t *testing.T) {
	t.Parallel()

	headers := NewHeaders(nil, nil)
	assert.Equal(t, "headers", headers.Type())
}

func TestHeaderOperations_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid operations", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
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

		err := ops.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid set header - empty name", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			SetHeaders: map[string]string{
				"": "value",
			},
		}

		err := ops.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "header name cannot be empty")
	})

	t.Run("invalid add header - empty name", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			AddHeaders: map[string]string{
				"  ": "value",
			},
		}

		err := ops.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "header name cannot be empty")
	})

	t.Run("invalid remove header - empty name", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			RemoveHeaders: []string{"Valid-Header", "", "Another-Valid"},
		}

		err := ops.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "remove header name cannot be empty")
	})

	t.Run("invalid header characters", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			SetHeaders: map[string]string{
				"Invalid\nHeader": "value",
			},
		}

		err := ops.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid header name")
	})
}

func TestHeaders_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid configuration with both request and response", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
				RemoveHeaders: []string{"X-Forwarded-For"},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
					"X-Frame-Options":        "DENY",
				},
				RemoveHeaders: []string{"Server", "X-Powered-By"},
			},
		}

		err := headers.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid configuration with only request", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
			},
		}

		err := headers.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid configuration with only response", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
			},
		}

		err := headers.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid configuration - no operations", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders(nil, nil)
		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(
			t,
			err.Error(),
			"at least one of request or response operations must be configured",
		)
	})

	t.Run("invalid request operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"": "value",
				},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"Valid-Header": "value",
				},
			},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid request header operations")
	})

	t.Run("invalid response operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"Valid-Header": "value",
				},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"": "value",
				},
			},
		}

		err := headers.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid response header operations")
	})
}

func TestHeaderOperations_String(t *testing.T) {
	t.Parallel()

	t.Run("empty operations", func(t *testing.T) {
		t.Parallel()

		ops := NewHeaderOperations("Test")
		assert.Equal(t, "No operations", ops.String())
	})

	t.Run("only set headers", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
		}

		result := ops.String()
		assert.Equal(t, "Set: 2 headers", result)
	})

	t.Run("all operations", func(t *testing.T) {
		t.Parallel()

		ops := &HeaderOperations{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
				"X-Custom":   "value",
			},
			RemoveHeaders: []string{"Server"},
		}

		result := ops.String()
		assert.Equal(t, "Set: 1 headers, Add: 2 headers, Remove: 1 headers", result)
	})
}

func TestHeaders_String(t *testing.T) {
	t.Parallel()

	t.Run("empty configuration", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders(nil, nil)
		assert.Equal(t, "No header operations configured", headers.String())
	})

	t.Run("only request operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
			},
		}

		result := headers.String()
		assert.Equal(t, "Request: Set: 1 headers", result)
	})

	t.Run("only response operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
			},
		}

		result := headers.String()
		assert.Equal(t, "Response: Set: 1 headers", result)
	})

	t.Run("both request and response operations", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
				RemoveHeaders: []string{"Server"},
			},
		}

		result := headers.String()
		assert.Equal(
			t,
			"Request: Set: 1 headers, Response: Set: 1 headers, Remove: 1 headers",
			result,
		)
	})
}

func TestHeaders_ToTree(t *testing.T) {
	t.Parallel()

	t.Run("empty configuration", func(t *testing.T) {
		t.Parallel()

		headers := NewHeaders(nil, nil)
		tree := headers.ToTree()

		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())

		// Empty configuration returns empty tree since no operations exist
		treeStr := tree.Tree().String()
		assert.Equal(t, "", treeStr)
	})

	t.Run("configuration with request and response operations", func(t *testing.T) {
		t.Parallel()

		// Create proper HeaderOperations with titles
		requestOps := NewHeaderOperations("Request")
		requestOps.SetHeaders["X-Real-IP"] = "127.0.0.1"
		requestOps.RemoveHeaders = []string{"X-Forwarded-For"}

		responseOps := NewHeaderOperations("Response")
		responseOps.SetHeaders["X-Content-Type-Options"] = "nosniff"
		responseOps.AddHeaders["Set-Cookie"] = "session=abc123"
		responseOps.RemoveHeaders = []string{"Server", "X-Powered-By"}

		headers := NewHeaders(requestOps, responseOps)

		tree := headers.ToTree()
		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())

		treeStr := tree.Tree().String()
		// New format has Request and Response sections with operations directly underneath
		assert.Contains(t, treeStr, "Request")
		assert.Contains(t, treeStr, "Set: \"X-Real-IP: 127.0.0.1\"")
		assert.Contains(t, treeStr, "Remove: \"X-Forwarded-For\"")
		assert.Contains(t, treeStr, "Response")
		assert.Contains(t, treeStr, "Set: \"X-Content-Type-Options: nosniff\"")
		assert.Contains(t, treeStr, "Add: \"Set-Cookie: session=abc123\"")
		assert.Contains(t, treeStr, "Remove: \"Server\"")
		assert.Contains(t, treeStr, "Remove: \"X-Powered-By\"")
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
