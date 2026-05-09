package middleware

// admin_test.go covers RequireAdmin in isolation. The middleware is
// trivially small but absolutely critical: every /admin/* endpoint
// chains it after the live UserStateChecker (auth.go), so a regression
// here opens every admin surface to non-admin callers.
//
// Tests:
//   - Allow when ctx is_admin = true.
//   - Deny with 403 + canonical error envelope when ctx is_admin = false.
//   - Deny with 403 when the is_admin context key is unset (no Auth ahead).
//   - Deny is independent of role / user id / org context — the flag
//     is the only signal RequireAdmin reads.
//   - The denial body matches the project's canonical {error, message}
//     envelope so the admin SPA's `adminApi` 401/403 handler can parse
//     it without special-casing.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reqWithIsAdmin builds a request whose context already carries the
// is_admin flag the Auth middleware would normally stamp. The role and
// user id are populated to ensure RequireAdmin does NOT incidentally
// rely on them — only the flag matters.
func reqWithIsAdmin(isAdmin bool) *http.Request {
	ctx := context.WithValue(context.Background(), ContextKeyIsAdmin, isAdmin)
	ctx = context.WithValue(ctx, ContextKeyRole, "agency")
	ctx = context.WithValue(ctx, ContextKeyUserID, uuid.New())
	return httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil).WithContext(ctx)
}

func TestRequireAdmin_AllowsAdminFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	RequireAdmin()(next).ServeHTTP(rec, reqWithIsAdmin(true))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "next handler must run when is_admin=true")
}

func TestRequireAdmin_DeniesNonAdminFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	RequireAdmin()(next).ServeHTTP(rec, reqWithIsAdmin(false))

	require.Equal(t, http.StatusForbidden, rec.Code,
		"is_admin=false MUST return 403 (not 401, which would leak that the route exists)")
	assert.False(t, called, "next handler MUST NOT run when is_admin=false")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "forbidden", body["error"])
	assert.Equal(t, "admin access required", body["message"])
}

// TestRequireAdmin_DeniesUnsetFlag — defense-in-depth. If the Auth
// middleware was misrouted (a future refactor accidentally drops
// is_admin off the context), a non-admin must NOT slip through. The
// `false` default of `GetIsAdmin` is the safe-by-default choice.
func TestRequireAdmin_DeniesUnsetFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })

	// No ContextKeyIsAdmin set on the request.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	RequireAdmin()(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code,
		"missing is_admin context key MUST default to deny — never trust an unset flag")
	assert.False(t, called)
}

// TestRequireAdmin_DeniesWrongTypeOnContext — paranoia. If a future
// change accidentally stores the flag as a non-bool (e.g. string
// "true"), GetIsAdmin returns false and we deny — no type-coercion
// surprises that would let a stringy "true" sneak through.
func TestRequireAdmin_DeniesWrongTypeOnContext(t *testing.T) {
	rec := httptest.NewRecorder()
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })

	ctx := context.WithValue(context.Background(), ContextKeyIsAdmin, "true") // wrong type on purpose
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil).WithContext(ctx)
	RequireAdmin()(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.False(t, called)
}

// TestRequireAdmin_FlagIsTheOnlySignal — independence assertion. The
// gate must NOT secretly read role / org_role / permissions. Locking
// this down keeps the contract simple and prevents future drift where
// someone adds a "if role == 'admin' also allow" branch (which would
// be wrong: marketplace roles never include 'admin' per the schema).
func TestRequireAdmin_FlagIsTheOnlySignal(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		expect int
	}{
		{
			name: "is_admin=true even with empty role still allowed",
			ctx: context.WithValue(
				context.WithValue(context.Background(), ContextKeyIsAdmin, true),
				ContextKeyRole, "",
			),
			expect: http.StatusOK,
		},
		{
			name: "is_admin=true even with role='enterprise' still allowed",
			ctx: context.WithValue(
				context.WithValue(context.Background(), ContextKeyIsAdmin, true),
				ContextKeyRole, "enterprise",
			),
			expect: http.StatusOK,
		},
		{
			name: "is_admin=false with role='admin' (impossible per schema, but defensive) still denied",
			ctx: context.WithValue(
				context.WithValue(context.Background(), ContextKeyIsAdmin, false),
				ContextKeyRole, "admin",
			),
			expect: http.StatusForbidden,
		},
		{
			name: "no role, is_admin=true allowed (admin SPA does not require a marketplace role)",
			ctx:  context.WithValue(context.Background(), ContextKeyIsAdmin, true),
			expect: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil).WithContext(tc.ctx)
			RequireAdmin()(next).ServeHTTP(rec, req)
			assert.Equal(t, tc.expect, rec.Code)
		})
	}
}

// TestRequireAdmin_DenialBodyIsCanonicalEnvelope — the admin SPA's
// `adminApi` parses the body shape on every non-2xx. A drift in the
// envelope shape (e.g. {message: "..."}-only without the error code)
// would silently break the SPA's error rendering even though the gate
// itself stays correct.
func TestRequireAdmin_DenialBodyIsCanonicalEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	RequireAdmin()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, reqWithIsAdmin(false))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json",
		"denial body MUST be JSON so the SPA can parse it")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Contains(t, body, "error", "envelope MUST carry an error code field")
	require.Contains(t, body, "message", "envelope MUST carry a human-readable message field")
	assert.Equal(t, "forbidden", body["error"])
}

// TestRequireAdmin_DoesNotConsumeRequestBody — the middleware must
// stay transparent to next-handler request body decoding. A regression
// here (e.g. accidentally reading the body to log) would silently
// break every admin POST that decodes JSON.
func TestRequireAdmin_DoesNotConsumeRequestBody(t *testing.T) {
	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/x/suspend",
		strings.NewReader(body))
	ctx := context.WithValue(req.Context(), ContextKeyIsAdmin, true)
	req = req.WithContext(ctx)

	var seenBody string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		seenBody = string(buf[:n])
	})
	rec := httptest.NewRecorder()
	RequireAdmin()(next).ServeHTTP(rec, req)

	assert.Equal(t, body, seenBody,
		"RequireAdmin MUST forward the request body unchanged to the protected handler")
}
