package handler

import "net/http"

type CookieConfig struct {
	Secure bool
	Domain string
	MaxAge int // seconds
}

func (c *CookieConfig) SetSession(w http.ResponseWriter, sessionID string, role string) {
	// httpOnly session cookie (secure, not accessible by JS)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   c.MaxAge,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
		Domain:   c.Domain,
	})

	// Non-httpOnly role cookie (for frontend UI rendering, not security)
	http.SetCookie(w, &http.Cookie{
		Name:     "user_role",
		Value:    role,
		Path:     "/",
		MaxAge:   c.MaxAge,
		HttpOnly: false,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
		Domain:   c.Domain,
	})
}

func (c *CookieConfig) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Domain:   c.Domain,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "user_role",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Domain:   c.Domain,
	})
}
