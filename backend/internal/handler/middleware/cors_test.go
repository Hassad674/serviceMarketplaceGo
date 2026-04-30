package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_WhitelistedOrigin(t *testing.T) {
	origins := []string{"https://app.example.com", "http://localhost:3000"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORS_NonWhitelistedOrigin(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "next must still be called for non-preflight")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"non-whitelisted origin must not receive Allow-Origin header")
}

func TestCORS_OptionsPreflight(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "preflight must not reach the next handler")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	// Audit SEC-36: lowered from 86400 (24h) to 600 (10 min) so allowlist
	// changes propagate to clients within minutes instead of a full day.
	assert.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_NoOriginHeader(t *testing.T) {
	origins := []string{"https://app.example.com"}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(origins)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"no origin header means no Allow-Origin in response")
}

func TestCORS_MultipleOrigins(t *testing.T) {
	origins := []string{
		"https://app.example.com",
		"http://localhost:3000",
		"http://localhost:5173",
	}

	tests := []struct {
		name       string
		origin     string
		wantAllow  string
	}{
		{
			name:      "first origin",
			origin:    "https://app.example.com",
			wantAllow: "https://app.example.com",
		},
		{
			name:      "second origin",
			origin:    "http://localhost:3000",
			wantAllow: "http://localhost:3000",
		},
		{
			name:      "third origin",
			origin:    "http://localhost:5173",
			wantAllow: "http://localhost:5173",
		},
		{
			name:      "unknown origin",
			origin:    "https://unknown.com",
			wantAllow: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(origins)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantAllow, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
