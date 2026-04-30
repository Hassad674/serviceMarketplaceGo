package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookieConfig_SetSession verifies that the httpOnly session_id
// cookie carries Secure + SameSite + HttpOnly flags exactly as the
// CookieConfig instructs, and that the role cookie carries the same
// SameSite/Secure but is intentionally NOT HttpOnly (the frontend
// needs to read it for UI rendering).
func TestCookieConfig_SetSession(t *testing.T) {
	tests := []struct {
		name       string
		cfg        CookieConfig
		wantSecure bool
		wantSS     http.SameSite
	}{
		{
			name:       "production: secure + strict",
			cfg:        CookieConfig{Secure: true, Domain: "example.com", MaxAge: 3600, SameSite: http.SameSiteStrictMode},
			wantSecure: true,
			wantSS:     http.SameSiteStrictMode,
		},
		{
			name:       "dev: insecure + lax",
			cfg:        CookieConfig{Secure: false, MaxAge: 3600, SameSite: http.SameSiteLaxMode},
			wantSecure: false,
			wantSS:     http.SameSiteLaxMode,
		},
		{
			name:       "default samesite falls back to lax",
			cfg:        CookieConfig{Secure: true, MaxAge: 3600},
			wantSecure: true,
			wantSS:     http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.cfg.SetSession(rec, "abcd-session-id", "agency")

			cookies := rec.Result().Cookies()
			require.Len(t, cookies, 2)

			byName := map[string]*http.Cookie{cookies[0].Name: cookies[0], cookies[1].Name: cookies[1]}
			session := byName["session_id"]
			role := byName["user_role"]
			require.NotNil(t, session, "session_id cookie missing")
			require.NotNil(t, role, "user_role cookie missing")

			// session_id: httpOnly + secure + samesite + path
			assert.Equal(t, "abcd-session-id", session.Value)
			assert.True(t, session.HttpOnly)
			assert.Equal(t, tt.wantSecure, session.Secure)
			assert.Equal(t, tt.wantSS, session.SameSite)
			assert.Equal(t, "/", session.Path)
			assert.Equal(t, tt.cfg.Domain, session.Domain)
			assert.Equal(t, tt.cfg.MaxAge, session.MaxAge)

			// user_role: NOT httpOnly (intentional), but secure + samesite
			assert.Equal(t, "agency", role.Value)
			assert.False(t, role.HttpOnly, "user_role must be readable by JS for UI rendering")
			assert.Equal(t, tt.wantSecure, role.Secure)
			assert.Equal(t, tt.wantSS, role.SameSite)
		})
	}
}

// TestCookieConfig_ClearSession verifies that the clear-cookie
// instructions echo the original Secure + SameSite flags so the
// browser actually drops the live cookies. Per RFC 6265 §5.3, a
// clear directive that does NOT match the original on these
// attributes is treated as a different cookie and ignored.
//
// Closes gosec G124 (cookie.go:49 / cookie.go:57) and the SEC-related
// concern that mobile WebView sessions could survive a logout because
// the clear instruction omitted SameSite.
func TestCookieConfig_ClearSession(t *testing.T) {
	tests := []struct {
		name       string
		cfg        CookieConfig
		wantSecure bool
		wantSS     http.SameSite
	}{
		{
			name:       "production prod-like: strict + secure + domain",
			cfg:        CookieConfig{Secure: true, Domain: "example.com", SameSite: http.SameSiteStrictMode},
			wantSecure: true,
			wantSS:     http.SameSiteStrictMode,
		},
		{
			name:       "lax + secure (typical)",
			cfg:        CookieConfig{Secure: true, SameSite: http.SameSiteLaxMode},
			wantSecure: true,
			wantSS:     http.SameSiteLaxMode,
		},
		{
			name:       "dev: insecure + default (lax)",
			cfg:        CookieConfig{Secure: false},
			wantSecure: false,
			wantSS:     http.SameSiteLaxMode,
		},
		{
			name:       "none samesite (cross-site embeds): paired with secure",
			cfg:        CookieConfig{Secure: true, SameSite: http.SameSiteNoneMode},
			wantSecure: true,
			wantSS:     http.SameSiteNoneMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.cfg.ClearSession(rec)

			cookies := rec.Result().Cookies()
			require.Len(t, cookies, 2)

			byName := map[string]*http.Cookie{cookies[0].Name: cookies[0], cookies[1].Name: cookies[1]}
			session := byName["session_id"]
			role := byName["user_role"]
			require.NotNil(t, session, "session_id clear missing")
			require.NotNil(t, role, "user_role clear missing")

			// session_id clear: same flags as live + MaxAge<0 + empty value
			assert.Empty(t, session.Value)
			assert.Equal(t, -1, session.MaxAge)
			assert.True(t, session.HttpOnly)
			assert.Equal(t, tt.wantSecure, session.Secure)
			assert.Equal(t, tt.wantSS, session.SameSite,
				"SameSite must be propagated to clear (RFC 6265 §5.3)")
			assert.Equal(t, tt.cfg.Domain, session.Domain)

			// user_role clear: same flags as live + MaxAge<0 + empty value
			assert.Empty(t, role.Value)
			assert.Equal(t, -1, role.MaxAge)
			assert.False(t, role.HttpOnly)
			assert.Equal(t, tt.wantSecure, role.Secure)
			assert.Equal(t, tt.wantSS, role.SameSite,
				"SameSite must be propagated to clear (RFC 6265 §5.3)")
		})
	}
}

// TestCookieConfig_ClearSession_AttributesMatchSetSession is the
// regression guard for SEC-related cookie behavior: every attribute
// that influences how the browser keys the cookie (Domain, Path,
// Secure, SameSite) MUST match between SetSession and ClearSession.
// If the values diverge, the browser treats the clear directive as
// a separate cookie — the live session_id stays in the jar and the
// user is NOT logged out, even after hitting /logout.
func TestCookieConfig_ClearSession_AttributesMatchSetSession(t *testing.T) {
	cfg := CookieConfig{
		Secure:   true,
		Domain:   "example.com",
		MaxAge:   3600,
		SameSite: http.SameSiteStrictMode,
	}

	setRec := httptest.NewRecorder()
	cfg.SetSession(setRec, "id", "agency")
	clearRec := httptest.NewRecorder()
	cfg.ClearSession(clearRec)

	setCookies := setRec.Result().Cookies()
	clearCookies := clearRec.Result().Cookies()
	require.Len(t, setCookies, 2)
	require.Len(t, clearCookies, 2)

	matchByName := func(cs []*http.Cookie, name string) *http.Cookie {
		for _, c := range cs {
			if c.Name == name {
				return c
			}
		}
		return nil
	}

	for _, name := range []string{"session_id", "user_role"} {
		live := matchByName(setCookies, name)
		clear := matchByName(clearCookies, name)
		require.NotNil(t, live, name+" missing in set")
		require.NotNil(t, clear, name+" missing in clear")

		assert.Equal(t, live.Path, clear.Path, "path mismatch on "+name)
		assert.Equal(t, live.Domain, clear.Domain, "domain mismatch on "+name)
		assert.Equal(t, live.Secure, clear.Secure, "secure mismatch on "+name)
		assert.Equal(t, live.SameSite, clear.SameSite, "samesite mismatch on "+name)
		assert.Equal(t, live.HttpOnly, clear.HttpOnly, "httpOnly mismatch on "+name)
	}
}

// TestCookieConfig_sameSite confirms that an explicit zero falls back
// to the safe SameSiteLaxMode default rather than producing a header
// without a SameSite directive (which Chrome treats as Lax but other
// browsers may treat as None).
func TestCookieConfig_sameSite(t *testing.T) {
	tests := []struct {
		name string
		in   http.SameSite
		want http.SameSite
	}{
		{name: "explicit strict", in: http.SameSiteStrictMode, want: http.SameSiteStrictMode},
		{name: "explicit lax", in: http.SameSiteLaxMode, want: http.SameSiteLaxMode},
		{name: "explicit none", in: http.SameSiteNoneMode, want: http.SameSiteNoneMode},
		{name: "default zero falls back to lax", in: http.SameSite(0), want: http.SameSiteLaxMode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &CookieConfig{SameSite: tt.in}
			assert.Equal(t, tt.want, cfg.sameSite())
		})
	}
}
