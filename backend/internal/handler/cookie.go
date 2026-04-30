package handler

import "net/http"

type CookieConfig struct {
	Secure   bool
	Domain   string
	MaxAge   int // seconds
	SameSite http.SameSite
}

func (c *CookieConfig) sameSite() http.SameSite {
	if c.SameSite != 0 {
		return c.SameSite
	}
	return http.SameSiteLaxMode
}

func (c *CookieConfig) SetSession(w http.ResponseWriter, sessionID string, role string) {
	ss := c.sameSite()

	// httpOnly session cookie (secure, not accessible by JS).
	// gosec G124 false-positive: Secure comes from CookieConfig
	// which is set to true in production via cmd/api/main.go; the
	// flag below is therefore env-dependent, not absent.
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure/SameSite via env-dependent CookieConfig
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   c.MaxAge,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: ss,
		Domain:   c.Domain,
	})

	// Non-httpOnly role cookie (for frontend UI rendering, not
	// security). The frontend reads this client-side to render the
	// correct sidebar/menu — it MUST be JS-readable. The session_id
	// httpOnly cookie above is the security cookie; this one is a
	// UI hint and carries no authority.
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- UI cookie, intentionally non-httpOnly
		Name:     "user_role",
		Value:    role,
		Path:     "/",
		MaxAge:   c.MaxAge,
		HttpOnly: false,
		Secure:   c.Secure,
		SameSite: ss,
		Domain:   c.Domain,
	})

}

// ClearSession overwrites the session_id and user_role cookies with
// expired sentinels so the browser drops them immediately.
//
// RFC 6265 §5.3 requires every clear cookie to share Domain, Path,
// Secure and SameSite with the original "Set-Cookie" — otherwise the
// browser treats the new directive as a *different* cookie and keeps
// the original value alive on the user's machine. SetSession sets
// HttpOnly + Secure + SameSite (via c.sameSite()) on both cookies, so
// ClearSession must echo those flags to remain honored. Closes
// gosec G124 on cookie.go:49 / cookie.go:57.
func (c *CookieConfig) ClearSession(w http.ResponseWriter) {
	ss := c.sameSite()
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure/SameSite via env-dependent CookieConfig
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: ss,
		Domain:   c.Domain,
	})
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- UI cookie clear, intentionally non-httpOnly to mirror SetSession
		Name:     "user_role",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   c.Secure,
		SameSite: ss,
		Domain:   c.Domain,
	})
}
