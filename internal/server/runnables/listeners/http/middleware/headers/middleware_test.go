package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/headers"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeadersMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("valid configuration", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			SetHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
			},
			RemoveHeaders: []string{"Server"},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)
		assert.NotNil(t, middleware)
		assert.Equal(t, "test", middleware.id)
	})

	t.Run("nil configuration", func(t *testing.T) {
		t.Parallel()

		middleware, err := NewHeadersMiddleware("test", nil)
		assert.Error(t, err)
		assert.Nil(t, middleware)
		assert.Contains(t, err.Error(), "headers config cannot be nil")
	})

	t.Run("invalid configuration", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			SetHeaders: map[string]string{
				"": "invalid-empty-name",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		assert.Error(t, err)
		assert.Nil(t, middleware)
		assert.Contains(t, err.Error(), "invalid headers config")
	})

	t.Run("empty configuration is valid", func(t *testing.T) {
		t.Parallel()

		cfg := headers.NewHeaders()
		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)
		assert.NotNil(t, middleware)
	})
}

func TestHeadersMiddleware_Integration(t *testing.T) {
	t.Parallel()

	t.Run("set headers only", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			SetHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "no-cache",
				"X-API-Version": "v2.1",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		assert.Equal(t, "v2.1", rec.Header().Get("X-API-Version"))
	})

	t.Run("add headers only", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			AddHeaders: map[string]string{
				"Set-Cookie":     "theme=dark", // Only one Set-Cookie in the map
				"X-Multi-Header": "value1",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Pre-set a cookie that should be preserved
		rec.Header().Set("Set-Cookie", "existing=cookie")

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		cookies := rec.Header().Values("Set-Cookie")
		assert.Len(t, cookies, 2) // existing + added
		assert.Contains(t, cookies, "existing=cookie")
		assert.Contains(t, cookies, "theme=dark")

		assert.Equal(t, "value1", rec.Header().Get("X-Multi-Header"))
	})

	t.Run("remove headers only", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			RemoveHeaders: []string{"Server", "X-Powered-By", "X-AspNet-Version"},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Pre-set headers that should be removed
		rec.Header().Set("Server", "Apache/2.4")
		rec.Header().Set("X-Powered-By", "PHP/8.0")
		rec.Header().Set("Content-Type", "text/html") // Should remain

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Server"))
		assert.Empty(t, rec.Header().Get("X-Powered-By"))
		assert.Empty(
			t,
			rec.Header().Get("X-AspNet-Version"),
		) // Removing non-existent is OK
		assert.Equal(t, "text/html", rec.Header().Get("Content-Type")) // Should remain
	})

	t.Run("all operations combined", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			RemoveHeaders: []string{"Server", "X-Powered-By"},
			SetHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "secure=true; HttpOnly",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Pre-set some headers
		rec.Header().Set("Server", "Apache/2.4")
		rec.Header().Set("X-Powered-By", "Express")
		rec.Header().Set("Set-Cookie", "session=abc123")

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		// Verify removals
		assert.Empty(t, rec.Header().Get("Server"))
		assert.Empty(t, rec.Header().Get("X-Powered-By"))

		// Verify sets
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))

		// Verify adds
		cookies := rec.Header().Values("Set-Cookie")
		assert.Len(t, cookies, 2)
		assert.Contains(t, cookies, "session=abc123")
		assert.Contains(t, cookies, "secure=true; HttpOnly")
	})

	t.Run("operation ordering: remove → set → add", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			RemoveHeaders: []string{"X-Test"},
			SetHeaders: map[string]string{
				"X-Test": "set-value",
			},
			AddHeaders: map[string]string{
				"X-Test": "add-value",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Pre-set header that should be removed first
		rec.Header().Set("X-Test", "original-value")

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		values := rec.Header().Values("X-Test")
		assert.Len(t, values, 2)
		assert.Contains(t, values, "set-value")
		assert.Contains(t, values, "add-value")
		assert.NotContains(t, values, "original-value") // Should be removed
	})

	t.Run("empty configuration does nothing", func(t *testing.T) {
		t.Parallel()

		cfg := headers.NewHeaders()
		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Pre-set a header that should remain unchanged
		rec.Header().Set("Existing", "value")

		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		assert.Equal(t, "value", rec.Header().Get("Existing"))
	})

	t.Run("middleware chain continues", func(t *testing.T) {
		t.Parallel()

		cfg := &headers.Headers{
			SetHeaders: map[string]string{
				"X-Test": "middleware-value",
			},
		}

		middleware, err := NewHeadersMiddleware("test", cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handlerCalled := false
		route, err := httpserver.NewRouteFromHandlerFunc("test", "/test",
			func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("response"))
				assert.NoError(t, err)
			}, middleware.Middleware())
		require.NoError(t, err)

		route.ServeHTTP(rec, req)

		assert.True(t, handlerCalled, "handler should be called")
		assert.Equal(t, "middleware-value", rec.Header().Get("X-Test"))
		assert.Equal(t, "response", rec.Body.String())
	})
}
