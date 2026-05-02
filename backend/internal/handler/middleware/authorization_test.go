package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reqWithRole builds a request whose context already carries the
// primary role set by the Auth middleware. Mirrors the pattern in
// permission_test.go.
func reqWithRole(role string) *http.Request {
	ctx := context.WithValue(context.Background(), ContextKeyRole, role)
	return httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
}

func TestRequireRole_AllowsMatchingRole(t *testing.T) {
	rec := httptest.NewRecorder()
	RequireRole("admin")(ok200).ServeHTTP(rec, reqWithRole("admin"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_DeniesNonMatchingRole(t *testing.T) {
	rec := httptest.NewRecorder()
	RequireRole("admin")(ok200).ServeHTTP(rec, reqWithRole("agency"))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "insufficient_role", body["error"])
}

func TestRequireRole_AllowsAnyOfMultipleRoles(t *testing.T) {
	allowed := RequireRole("agency", "provider")
	for _, role := range []string{"agency", "provider"} {
		t.Run(role, func(t *testing.T) {
			rec := httptest.NewRecorder()
			allowed(ok200).ServeHTTP(rec, reqWithRole(role))
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
	t.Run("denies enterprise when only agency+provider allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		allowed(ok200).ServeHTTP(rec, reqWithRole("enterprise"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRequireRole_DeniesUnauthenticatedRequest(t *testing.T) {
	// No ContextKeyRole on the context — the Auth middleware was
	// not chained or did not match. Must return 401, not 403, so
	// the client knows to authenticate first.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	RequireRole("admin")(ok200).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
}

func TestRequireRole_RoleStringIsCaseSensitive(t *testing.T) {
	// We do NOT lowercase the role server-side — the auth flow signs
	// canonical lower-case roles. A request carrying `Admin` must
	// fail closed so a mis-cased token cannot bypass.
	rec := httptest.NewRecorder()
	RequireRole("admin")(ok200).ServeHTTP(rec, reqWithRole("Admin"))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireRole_EmptyRoleAllowListPanics(t *testing.T) {
	assert.Panics(t, func() {
		RequireRole()
	})
}

func TestRequireRole_EmptyRoleStringPanics(t *testing.T) {
	assert.Panics(t, func() {
		RequireRole("admin", "")
	})
}

func TestRequireRole_DoesNotMutateRequest(t *testing.T) {
	// A simple sanity check: middleware must forward the request
	// unchanged when allowed.
	captured := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = true
		assert.Equal(t, "admin", GetRole(r.Context()))
	})

	rec := httptest.NewRecorder()
	RequireRole("admin")(next).ServeHTTP(rec, reqWithRole("admin"))
	assert.True(t, captured)
}

// TestRequireRole_AllowsAllStandardRoles ensures the middleware works
// with each canonical role string emitted by the auth flow. Catches
// future drift between domain.Role values and the strings the
// middleware sees.
func TestRequireRole_AllowsAllStandardRoles(t *testing.T) {
	roles := []string{"agency", "provider", "enterprise", "admin"}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequireRole(role)(ok200).ServeHTTP(rec, reqWithRole(role))
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
