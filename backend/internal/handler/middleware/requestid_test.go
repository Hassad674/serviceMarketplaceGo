package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesUUID(t *testing.T) {
	var ctxID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxID = GetRequestID(r.Context())
	})

	handler := RequestID(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-Request-ID header provided.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	headerID := rec.Header().Get("X-Request-ID")
	require.NotEmpty(t, headerID, "must generate a request ID")
	assert.Len(t, headerID, 36, "generated ID must be a UUID (36 chars)")
	assert.Equal(t, headerID, ctxID, "context ID must match response header")
}

func TestRequestID_UsesProvidedValue(t *testing.T) {
	provided := "custom-trace-id-12345"

	var ctxID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxID = GetRequestID(r.Context())
	})

	handler := RequestID(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", provided)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, provided, rec.Header().Get("X-Request-ID"),
		"response header must echo the provided ID")
	assert.Equal(t, provided, ctxID,
		"context must contain the provided ID")
}

func TestRequestID_PropagatedInContext(t *testing.T) {
	var ctxID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxID = GetRequestID(r.Context())
	})

	handler := RequestID(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.NotEmpty(t, ctxID, "request ID must be in context")
	assert.Equal(t, rec.Header().Get("X-Request-ID"), ctxID,
		"context value and response header must be identical")
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := GetRequestID(req.Context())
	assert.Empty(t, id, "empty context must return empty string")
}

func TestGetUserID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, ok := GetUserID(req.Context())
	assert.False(t, ok, "empty context must return false")
}

func TestGetRole_EmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	role := GetRole(req.Context())
	assert.Empty(t, role, "empty context must return empty string")
}
