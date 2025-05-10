package matcher

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPPathMatcher_Matches(t *testing.T) {
	tests := []struct {
		name       string
		pathPrefix string
		requestURL string
		want       bool
	}{
		{
			name:       "exact match",
			pathPrefix: "/api/v1",
			requestURL: "/api/v1",
			want:       true,
		},
		{
			name:       "prefix match",
			pathPrefix: "/api/v1",
			requestURL: "/api/v1/users",
			want:       true,
		},
		{
			name:       "no match",
			pathPrefix: "/api/v1",
			requestURL: "/api/v2",
			want:       false,
		},
		{
			name:       "empty path prefix always matches",
			pathPrefix: "",
			requestURL: "/any/path",
			want:       true,
		},
		{
			name:       "nil request",
			pathPrefix: "/api",
			requestURL: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewHTTPPathMatcher(tt.pathPrefix)

			var req *http.Request
			if tt.requestURL != "" {
				u, err := url.Parse("http://example.com" + tt.requestURL)
				require.NoError(t, err)
				req = &http.Request{URL: u}
			}

			if got := m.Matches(req); got != tt.want {
				t.Errorf("HTTPPathMatcher.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPPathMatcher_ExtractParams(t *testing.T) {
	m := NewHTTPPathMatcher("/api")
	u, err := url.Parse("http://example.com/api/users")
	require.NoError(t, err)
	req := &http.Request{URL: u}

	params := m.ExtractParams(req)
	if len(params) != 0 {
		t.Errorf("Expected empty params map, got %v", params)
	}
}

func TestHTTPPathMatcher_PathPrefix(t *testing.T) {
	prefix := "/api/v1"
	m := NewHTTPPathMatcher(prefix)

	if got := m.PathPrefix(); got != prefix {
		t.Errorf("PathPrefix() = %v, want %v", got, prefix)
	}
}
