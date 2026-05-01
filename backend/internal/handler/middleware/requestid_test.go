package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
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

// TestMustGetOrgID_ReturnsValueWhenSet verifies the happy path
// where the auth middleware has stamped the org id into the
// context. This is the canonical user-facing handler scenario.
func TestMustGetOrgID_ReturnsValueWhenSet(t *testing.T) {
	orgID := uuid.New()
	ctx := context.WithValue(context.Background(), ContextKeyOrganizationID, orgID)
	got := MustGetOrgID(ctx)
	assert.Equal(t, orgID, got,
		"MustGetOrgID must return the value stamped by the auth middleware")
}

// TestMustGetOrgID_PanicsOnEmpty verifies the panic contract.
// Calling MustGetOrgID without an authenticated org context is a
// programming bug — the panic surfaces it during unit tests
// rather than letting the call silently degrade to uuid.Nil.
func TestMustGetOrgID_PanicsOnEmpty(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "MustGetOrgID must panic when the org id is missing")
		msg, ok := r.(string)
		require.True(t, ok, "panic value must be a string for clarity")
		assert.Contains(t, msg, "organization id missing",
			"panic message must point at the missing org id, not be a generic nil deref")
	}()
	_ = MustGetOrgID(context.Background())
}

// TestMustGetOrgID_PanicsOnNilUUID guards the edge case where
// the auth middleware populated the key but with uuid.Nil — that
// is legitimate for solo providers without an org, and the
// MustGet contract is "this code path requires an org", so a
// solo-provider call here must still panic.
func TestMustGetOrgID_PanicsOnNilUUID(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r,
			"MustGetOrgID must panic when the org id is uuid.Nil — "+
				"a solo-provider hitting an org-required code path is a bug")
	}()
	ctx := context.WithValue(context.Background(), ContextKeyOrganizationID, uuid.Nil)
	_ = MustGetOrgID(ctx)
}
