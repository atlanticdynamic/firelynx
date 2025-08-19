package listeners

import (
	"slices"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenerCollection_All(t *testing.T) {
	t.Parallel()

	// Create test listeners
	listener1 := Listener{
		ID:      "listener-1",
		Address: "0.0.0.0:8080",
		Type:    TypeHTTP,
	}
	listener2 := Listener{
		ID:      "listener-2",
		Address: "0.0.0.0:8443",
		Type:    TypeHTTP,
	}
	listener3 := Listener{
		ID:      "listener-3",
		Address: "0.0.0.0:9090",
		Type:    TypeUnspecified,
	}

	collection := ListenerCollection{listener1, listener2, listener3}

	t.Run("Iterate over all listeners", func(t *testing.T) {
		var result []Listener
		for listener := range collection.All() {
			result = append(result, listener)
		}

		assert.Len(t, result, 3)
		assert.Equal(t, listener1, result[0])
		assert.Equal(t, listener2, result[1])
		assert.Equal(t, listener3, result[2])
	})

	t.Run("Early termination", func(t *testing.T) {
		var count int
		for listener := range collection.All() {
			count++
			if listener.ID == "listener-2" {
				break // Early termination
			}
		}
		assert.Equal(t, 2, count)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := ListenerCollection{}
		var result []Listener
		for listener := range emptyCollection.All() {
			result = append(result, listener)
		}
		assert.Empty(t, result)
	})

	t.Run("Use with slices.Collect", func(t *testing.T) {
		collected := slices.Collect(collection.All())
		assert.Len(t, collected, 3)
		assert.Equal(t, listener1, collected[0])
		assert.Equal(t, listener2, collected[1])
		assert.Equal(t, listener3, collected[2])
	})
}

func TestListenerCollection_FindByID(t *testing.T) {
	t.Parallel()

	// Create test listeners with different types
	httpListener := Listener{
		ID:      "http-listener",
		Address: "0.0.0.0:8080",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout: 30 * time.Second,
		},
	}
	httpListener2 := Listener{
		ID:      "http-listener-2",
		Address: "0.0.0.0:8443",
		Type:    TypeHTTP,
	}
	unspecifiedListener := Listener{
		ID:      "unspecified-listener",
		Address: "0.0.0.0:9090",
		Type:    TypeUnspecified,
	}

	collection := ListenerCollection{httpListener, httpListener2, unspecifiedListener}

	t.Run("Find existing HTTP listener", func(t *testing.T) {
		result, found := collection.FindByID("http-listener")
		require.True(t, found)
		assert.Equal(t, httpListener, result)
		assert.Equal(t, TypeHTTP, result.Type)
	})

	t.Run("Find second HTTP listener", func(t *testing.T) {
		result, found := collection.FindByID("http-listener-2")
		require.True(t, found)
		assert.Equal(t, httpListener2, result)
	})

	t.Run("Find unspecified type listener", func(t *testing.T) {
		result, found := collection.FindByID("unspecified-listener")
		require.True(t, found)
		assert.Equal(t, unspecifiedListener, result)
		assert.Equal(t, TypeUnspecified, result.Type)
	})

	t.Run("Listener not found", func(t *testing.T) {
		result, found := collection.FindByID("non-existent")
		assert.False(t, found)
		assert.Equal(t, Listener{}, result) // Zero value
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := ListenerCollection{}
		result, found := emptyCollection.FindByID("any-id")
		assert.False(t, found)
		assert.Equal(t, Listener{}, result)
	})

	t.Run("Empty ID search", func(t *testing.T) {
		result, found := collection.FindByID("")
		assert.False(t, found)
		assert.Equal(t, Listener{}, result)
	})
}

func TestListenerCollection_FindByType(t *testing.T) {
	t.Parallel()

	// Create test listeners with different types
	httpListener1 := Listener{
		ID:      "http-1",
		Address: "0.0.0.0:8080",
		Type:    TypeHTTP,
	}
	httpListener2 := Listener{
		ID:      "http-2",
		Address: "0.0.0.0:8443",
		Type:    TypeHTTP,
	}
	unspecifiedListener1 := Listener{
		ID:      "unspecified-1",
		Address: "0.0.0.0:9090",
		Type:    TypeUnspecified,
	}
	httpListener3 := Listener{
		ID:      "http-3",
		Address: "0.0.0.0:8444",
		Type:    TypeHTTP,
	}

	collection := ListenerCollection{httpListener1, httpListener2, unspecifiedListener1, httpListener3}

	t.Run("Find HTTP type listeners", func(t *testing.T) {
		var result []Listener
		for listener := range collection.FindByType(TypeHTTP) {
			result = append(result, listener)
		}

		assert.Len(t, result, 3)
		assert.Equal(t, httpListener1, result[0])
		assert.Equal(t, httpListener2, result[1])
		assert.Equal(t, httpListener3, result[2])
	})

	t.Run("Find Unspecified type listeners", func(t *testing.T) {
		var result []Listener
		for listener := range collection.FindByType(TypeUnspecified) {
			result = append(result, listener)
		}

		assert.Len(t, result, 1)
		assert.Equal(t, unspecifiedListener1, result[0])
	})

	t.Run("No listeners of type", func(t *testing.T) {
		// Try to find a type that doesn't exist in our collection
		var result []Listener
		for listener := range collection.FindByType(Type(999)) {
			result = append(result, listener)
		}
		assert.Empty(t, result)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := ListenerCollection{}
		var result []Listener
		for listener := range emptyCollection.FindByType(TypeHTTP) {
			result = append(result, listener)
		}
		assert.Empty(t, result)
	})

	t.Run("Early termination", func(t *testing.T) {
		var count int
		for listener := range collection.FindByType(TypeHTTP) {
			count++
			if listener.ID == "http-1" {
				break // Stop after first
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("Use with slices.Collect", func(t *testing.T) {
		collected := slices.Collect(collection.FindByType(TypeHTTP))
		assert.Len(t, collected, 3)
		assert.Equal(t, httpListener1, collected[0])
		assert.Equal(t, httpListener2, collected[1])
		assert.Equal(t, httpListener3, collected[2])
	})
}

func TestListenerCollection_GetHTTPListeners(t *testing.T) {
	t.Parallel()

	// Create mixed type listeners
	httpListener1 := Listener{
		ID:      "http-1",
		Address: "0.0.0.0:8080",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout: 30 * time.Second,
		},
	}
	unspecifiedListener := Listener{
		ID:      "unspecified-1",
		Address: "0.0.0.0:9090",
		Type:    TypeUnspecified,
	}
	httpListener2 := Listener{
		ID:      "http-2",
		Address: "0.0.0.0:8443",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout: 60 * time.Second,
		},
	}

	collection := ListenerCollection{httpListener1, unspecifiedListener, httpListener2}

	t.Run("Get only HTTP listeners", func(t *testing.T) {
		httpListeners := collection.GetHTTPListeners()
		assert.Len(t, httpListeners, 2)
		assert.Equal(t, httpListener1, httpListeners[0])
		assert.Equal(t, httpListener2, httpListeners[1])
	})

	t.Run("No HTTP listeners", func(t *testing.T) {
		noHttpCollection := ListenerCollection{unspecifiedListener}
		httpListeners := noHttpCollection.GetHTTPListeners()
		assert.Empty(t, httpListeners)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := ListenerCollection{}
		httpListeners := emptyCollection.GetHTTPListeners()
		assert.Empty(t, httpListeners)
	})

	t.Run("All listeners are HTTP", func(t *testing.T) {
		allHttpCollection := ListenerCollection{httpListener1, httpListener2}
		httpListeners := allHttpCollection.GetHTTPListeners()
		assert.Len(t, httpListeners, 2)
		assert.Equal(t, httpListener1, httpListeners[0])
		assert.Equal(t, httpListener2, httpListeners[1])
	})
}

func TestListener_GetOptionsType(t *testing.T) {
	t.Parallel()

	t.Run("HTTP options type", func(t *testing.T) {
		listener := Listener{
			ID:      "http-listener",
			Options: options.HTTP{},
		}
		assert.Equal(t, options.TypeHTTP, listener.GetOptionsType())
	})

	t.Run("Nil options", func(t *testing.T) {
		listener := Listener{
			ID:      "no-options",
			Options: nil,
		}
		assert.Equal(t, options.Unknown, listener.GetOptionsType())
	})

	t.Run("Custom options type", func(t *testing.T) {
		// Create a mock options type for testing
		mockOptions := options.HTTP{
			ReadTimeout: 30 * time.Second,
		}
		listener := Listener{
			ID:      "custom-listener",
			Options: mockOptions,
		}
		assert.Equal(t, options.TypeHTTP, listener.GetOptionsType())
	})
}

func TestListener_GetTimeouts_NilOptions(t *testing.T) {
	t.Parallel()

	// Test timeout methods when Options is nil (should return defaults)
	listener := Listener{
		ID:      "nil-options",
		Options: nil,
	}

	t.Run("GetReadTimeout with nil options", func(t *testing.T) {
		timeout := listener.GetReadTimeout()
		assert.Equal(t, options.DefaultHTTPReadTimeout, timeout)
	})

	t.Run("GetWriteTimeout with nil options", func(t *testing.T) {
		timeout := listener.GetWriteTimeout()
		assert.Equal(t, options.DefaultHTTPWriteTimeout, timeout)
	})

	t.Run("GetDrainTimeout with nil options", func(t *testing.T) {
		timeout := listener.GetDrainTimeout()
		assert.Equal(t, options.DefaultHTTPDrainTimeout, timeout)
	})

	t.Run("GetIdleTimeout with nil options", func(t *testing.T) {
		timeout := listener.GetIdleTimeout()
		assert.Equal(t, options.DefaultHTTPIdleTimeout, timeout)
	})
}

// testCustomOptions implements options.Options for testing
type testCustomOptions struct{}

func (c testCustomOptions) Type() options.Type           { return options.Type("test-custom") }
func (c testCustomOptions) Validate() error              { return nil }
func (c testCustomOptions) String() string               { return "test-custom" }
func (c testCustomOptions) ToTree() *fancy.ComponentTree { return nil }

func TestListener_GetTimeouts_NonHTTPOptions(t *testing.T) {
	t.Parallel()

	listener := Listener{
		ID:      "non-http",
		Options: testCustomOptions{},
	}

	t.Run("GetReadTimeout with non-HTTP options", func(t *testing.T) {
		timeout := listener.GetReadTimeout()
		assert.Equal(t, options.DefaultHTTPReadTimeout, timeout)
	})

	t.Run("GetWriteTimeout with non-HTTP options", func(t *testing.T) {
		timeout := listener.GetWriteTimeout()
		assert.Equal(t, options.DefaultHTTPWriteTimeout, timeout)
	})

	t.Run("GetDrainTimeout with non-HTTP options", func(t *testing.T) {
		timeout := listener.GetDrainTimeout()
		assert.Equal(t, options.DefaultHTTPDrainTimeout, timeout)
	})

	t.Run("GetIdleTimeout with non-HTTP options", func(t *testing.T) {
		timeout := listener.GetIdleTimeout()
		assert.Equal(t, options.DefaultHTTPIdleTimeout, timeout)
	})
}

func TestListenerCollection_ComplexScenario(t *testing.T) {
	t.Parallel()

	// Create a complex scenario with various listener configurations
	apiListener := Listener{
		ID:      "api-https",
		Address: "0.0.0.0:443",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			DrainTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}

	adminListener := Listener{
		ID:      "admin-https",
		Address: "127.0.0.1:8443",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
	}

	internalListener := Listener{
		ID:      "internal-http",
		Address: "localhost:8080",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout: 10 * time.Second,
		},
	}

	debugListener := Listener{
		ID:      "debug",
		Address: "localhost:9999",
		Type:    TypeUnspecified,
	}

	collection := ListenerCollection{apiListener, adminListener, internalListener, debugListener}

	t.Run("Find specific listener by ID", func(t *testing.T) {
		listener, found := collection.FindByID("admin-https")
		require.True(t, found)
		assert.Equal(t, adminListener, listener)
		assert.Equal(t, "127.0.0.1:8443", listener.Address)
	})

	t.Run("Get all HTTP listeners", func(t *testing.T) {
		httpListeners := collection.GetHTTPListeners()
		assert.Len(t, httpListeners, 3) // api, admin, internal
		// Verify they are in the correct order
		assert.Equal(t, "api-https", httpListeners[0].ID)
		assert.Equal(t, "admin-https", httpListeners[1].ID)
		assert.Equal(t, "internal-http", httpListeners[2].ID)
	})

	t.Run("Get Unspecified type listeners", func(t *testing.T) {
		unspecified := slices.Collect(collection.FindByType(TypeUnspecified))
		assert.Len(t, unspecified, 1)
		assert.Equal(t, debugListener, unspecified[0])
	})

	t.Run("Verify timeout configurations", func(t *testing.T) {
		// API listener should have all timeouts configured
		assert.Equal(t, 30*time.Second, apiListener.GetReadTimeout())
		assert.Equal(t, 30*time.Second, apiListener.GetWriteTimeout())
		assert.Equal(t, 60*time.Second, apiListener.GetDrainTimeout())
		assert.Equal(t, 120*time.Second, apiListener.GetIdleTimeout())

		// Admin listener has partial configuration
		assert.Equal(t, 60*time.Second, adminListener.GetReadTimeout())
		assert.Equal(t, 60*time.Second, adminListener.GetWriteTimeout())
		assert.Equal(t, options.DefaultHTTPDrainTimeout, adminListener.GetDrainTimeout())
		assert.Equal(t, options.DefaultHTTPIdleTimeout, adminListener.GetIdleTimeout())

		// Internal listener has minimal configuration
		assert.Equal(t, 10*time.Second, internalListener.GetReadTimeout())
		assert.Equal(t, options.DefaultHTTPWriteTimeout, internalListener.GetWriteTimeout())

		// Debug listener should use all defaults
		assert.Equal(t, options.DefaultHTTPReadTimeout, debugListener.GetReadTimeout())
		assert.Equal(t, options.DefaultHTTPWriteTimeout, debugListener.GetWriteTimeout())
	})

	t.Run("Iterate and filter simultaneously", func(t *testing.T) {
		// Count HTTP listeners with read timeout > 20s
		var count int
		for listener := range collection.FindByType(TypeHTTP) {
			if listener.GetReadTimeout() > 20*time.Second {
				count++
			}
		}
		assert.Equal(t, 2, count) // api (30s) and admin (60s)
	})

	t.Run("Get options type for each listener", func(t *testing.T) {
		assert.Equal(t, options.TypeHTTP, apiListener.GetOptionsType())
		assert.Equal(t, options.TypeHTTP, adminListener.GetOptionsType())
		assert.Equal(t, options.TypeHTTP, internalListener.GetOptionsType())
		assert.Equal(t, options.Unknown, debugListener.GetOptionsType()) // No options set
	})
}

func TestListener_GetHTTPOptions_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Type mismatch detection", func(t *testing.T) {
		// Use the custom options type defined at package level
		listener := Listener{
			ID:      "custom",
			Options: testCustomOptions{},
		}

		httpOpts, ok := listener.GetHTTPOptions()
		assert.False(t, ok)
		assert.Equal(t, options.HTTP{}, httpOpts)
	})

	t.Run("Type assertion failure", func(t *testing.T) {
		// This tests the type assertion in GetHTTPOptions
		listener := Listener{
			ID:      "http",
			Options: options.HTTP{ReadTimeout: 30 * time.Second},
		}

		httpOpts, ok := listener.GetHTTPOptions()
		assert.True(t, ok)
		assert.Equal(t, 30*time.Second, httpOpts.ReadTimeout)
	})
}
